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

Each project uses its own `access` block for every Google API call. Omit it to use Application Default Credentials (local ADC, an attached Cloud Run identity, or workload identity federation). Set `access.credentials` to an exact environment reference such as `${GCME_PRODUCTION_CREDENTIALS}` when a project needs different credentials; the environment value must be base64-encoded credential JSON, never a filename. It supports service-account, authorized-user, and external-account/WIF configurations. Credentials are excluded from `config show` output and are not sent to Temporal. `access.impersonation.target_service_account` optionally exchanges the base identity for a target service account; `delegates` and a `lifetime` of at most `1h` are also supported. The base identity needs `roles/iam.serviceAccountTokenCreator` on the target (and each delegation hop).

For local work run `gcloud auth application-default login`; production deployments should attach a service account with permission to list Cloud Run services, revisions, and worker pools. [`terraform/gcp`](terraform/gcp) contains a Terraform example that creates this service account and grants `roles/run.viewer` in the configured target projects. Apply the [`terraform/elasticsearch`](terraform/elasticsearch) index template before the exporter writes documents. Elasticsearch credentials are read from `GCME_STORAGE_ELASTICSEARCH_USERNAME` and `GCME_STORAGE_ELASTICSEARCH_PASSWORD`.

## Billing exports

Add `billing` to a project's resources and configure that project's `billing` block with the BigQuery `dataset` (`project.dataset`), export `table`, and BigQuery `location`. `query_project` is optional and defaults to the target project's `id`; set it when a different project should run the query. This permits different target projects to query different exports, including exports owned by separate organizations. The selected identity needs BigQuery job-user permission on the query project and read access to the billing-export table. Each run writes one document per invoice month for the configured target project; set `group_by_service: true` to write a separate document for each GCP service, and `group_by_cost_type: true` to separate regular charges, taxes, adjustments, and rounding. By default those types are included in the project total. The query includes the current and preceding invoice month to retain late credits and adjustments.

For a non-breaking migration, an existing `global.billing` block is used as the billing configuration for projects that enable `billing` but do not yet declare `projects[].billing`. New configurations should use the project-level block.

The versioned GoogleSQL sources are in [`internal/queries/queries`](internal/queries/queries), with a dedicated file for each requested service/cost-type breakdown. They are embedded in the binary at build time, so the Docker image needs no separate runtime mount. The query accepts values only through named parameters; the billing table identifier is validated before being rendered into the query.

## Local Docker Compose

`deploy/infra.yaml` defines PostgreSQL, Temporal, and Elasticsearch. The exporter is defined separately in `deploy/compose.yml`. Apply the [`terraform/elasticsearch`](terraform/elasticsearch) index template before starting the exporter. Copy `deploy/config.yaml.example` to `deploy/config.yaml`, update the BigQuery identifiers, then export the base64-encoded credential JSON:

```bash
cp deploy/config.yaml.example deploy/config.yaml
export GCME_PRODUCTION_CREDENTIALS="$(terraform -chdir=terraform/gcp output -raw service_account_key)"
docker compose -f deploy/infra.yaml -f deploy/compose.yml up --build
```

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
