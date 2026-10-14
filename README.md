# GCP Metrics Exporter

GCP Metrics Exporter inventories Cloud Run resources and writes the API objects to Elasticsearch. Temporal schedules one workflow per configured project, so a failed collection can be retried without blocking other projects.

## Supported resources

| Resource key | Cloud Run v2 resource | Temporal activity |
|---|---|---|
| `cloud_run_service` | Service | `GetCloudRunServices` |
| `cloud_run_revision` | Revision | `GetCloudRunRevisions` |
| `cloud_run_worker_pool` | WorkerPool | `GetCloudRunWorkerPools` |

## Configuration and authentication

Start with [`configs/example.yaml`](configs/example.yaml). `scrape_schedule` uses a standard five-field cron expression and defaults to `0 * * * *`; Temporal evaluates it in `schedule_timezone` (UTC by default). Project settings inherit global locations, resources, and schedule.

The process uses Application Default Credentials. For local work run `gcloud auth application-default login`; production deployments should attach a service account with permission to list Cloud Run services, revisions, and worker pools. [`terraform/gcp`](terraform/gcp) contains a Terraform example that creates this service account and grants `roles/run.viewer` in the configured target projects. Elasticsearch credentials are read from `GCME_STORAGE_ELASTICSEARCH_USERNAME` and `GCME_STORAGE_ELASTICSEARCH_PASSWORD`.

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
