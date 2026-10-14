# GCP Metrics Exporter

GCP Metrics Exporter inventories GCP resources and writes the API objects to Elasticsearch. Temporal schedules one workflow per configured project, so a failed collection can be retried without blocking other projects.

## Supported resources

| Resource key | Cloud Run v2 resource | Temporal activity |
|---|---|---|
| `billing` | BigQuery billing export | `GetBilling` |
| `cloud_run_service` | Service | `GetCloudRunServices` |
| `cloud_run_revision` | Revision | `GetCloudRunRevisions` |
| `cloud_run_worker_pool` | WorkerPool | `GetCloudRunWorkerPools` |

## Configuration and authentication

Start with [`configs/example.yaml`](configs/example.yaml). `scrape_schedule` uses a standard five-field cron expression and defaults to `0 * * * *`; Temporal evaluates it in `schedule_timezone` (UTC by default). Project settings inherit global locations, resources, and schedule.

The process uses Application Default Credentials. For local work run `gcloud auth application-default login`; production deployments should attach a service account with permission to list Cloud Run services, revisions, and worker pools. Elasticsearch credentials are read from `GCME_STORAGE_ELASTICSEARCH_USERNAME` and `GCME_STORAGE_ELASTICSEARCH_PASSWORD`.

## Billing exports

Add `billing` to `resources` and configure `global.billing` with the fully-qualified standard BigQuery billing-export table, its BigQuery `location`, and the project that will run the query. The service account needs BigQuery job-user permission on `query_project` and read access to the billing-export table. Each run writes one document per invoice month for the configured target project; set `group_by_service: true` to write a separate document for each GCP service, and `group_by_cost_type: true` to separate regular charges, taxes, adjustments, and rounding. By default those types are included in the project total. The query includes the current and preceding invoice month to retain late credits and adjustments.

The versioned GoogleSQL source is [`internal/queries/queries/get-billing-data.sql`](internal/queries/queries/get-billing-data.sql). It is embedded in the binary at build time, so the Docker image needs no separate runtime mount. The query accepts values only through named parameters; the billing table identifier is validated before being rendered into the query.

## Local Docker Compose

`deploy/docker-compose.yml` starts PostgreSQL, Temporal, Elasticsearch, and the exporter, and applies the included Elasticsearch index template. Copy `deploy/config.yaml.example` to `deploy/config.yaml`, update the BigQuery identifiers, then point `GCME_ADC_FILE` at an Application Default Credentials JSON file:

```bash
cp deploy/config.yaml.example deploy/config.yaml
GCME_ADC_FILE="$HOME/.config/gcloud/application_default_credentials.json" docker compose -f deploy/docker-compose.yml up --build
```

The Elasticsearch template in `deploy/elasticsearch-index-template.json` uses the default `gcp-inventory-*` index pattern. Adjust its pattern when you set a different `storage.elasticsearch.index_prefix`.

## Running

```bash
go run . config validate --config configs/example.yaml
go run . worker --config configs/example.yaml
go run . schedule list --config configs/example.yaml
go run . schedule trigger --project production --config configs/example.yaml
```

The worker reconciles `gcme-project-<name>` schedules at startup and skips overlapping runs. `schedule apply`, `pause`, `resume`, and `trigger` manage the schedules directly.

## Development

```bash
task test
task lint
task build
```

The release workflow publishes signed, attested multi-architecture images to `ghcr.io/windkube/gcp-metrics-exporter`.
