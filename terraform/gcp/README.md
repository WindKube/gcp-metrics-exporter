# Terraform example

Creates a dedicated service account for `gcp-metrics-exporter`. It receives
`roles/run.viewer` and `roles/bigquery.jobUser` in every target project, plus
`roles/bigquery.dataViewer` on configured billing-export datasets. Cloud Run
Viewer includes the list permissions for Cloud Run services, revisions, and
worker pools used by the exporter.

The Terraform identity must be allowed to create service accounts in
`service_account_project_id` and modify IAM policy in every `target_project_ids`
project. The Cloud Run Admin API must already be enabled in each target project.

Create `terraform.tfvars`:

```hcl
service_account_project_id = "exporter-platform"
target_project_ids = [
  "production-app",
  "staging-app",
]

billing_datasets = [{
  project_id = "example"
  dataset_id = "billing_export"
}]
```

Authenticate Terraform with Application Default Credentials, then apply:

```bash
gcloud auth application-default login
terraform init
terraform apply
```

Terraform also creates a service-account key for the exporter. Set
`GCME_PRODUCTION_CREDENTIALS` to the sensitive, base64-encoded
`service_account_key` output before starting the Docker Compose deployment:

```bash
export GCME_PRODUCTION_CREDENTIALS="$(terraform output -raw service_account_key)"
```

The exporter decodes this value into credential JSON. Store the environment
variable in a secret manager for production deployments. Rotate it by replacing
the Terraform-managed key when required. The private key is also stored in
Terraform state, so use encrypted state with tightly restricted access.

List the same target projects in the exporter configuration:

```yaml
projects:
  - name: production
    id: production-app
  - name: staging
    id: staging-app
```

When adding collectors, extend the role assignment or replace it with a custom
role containing the new API permissions before enabling the collector.

For billing collection, the example grants BigQuery Job User on every target
project, covering the default `billing.query_project`. Add a different project
to `billing_query_project_ids` when the configuration overrides
`billing.query_project`. The `billing_datasets` value grants read access to
each configured `billing.dataset`. When using
`projects[].access.impersonation`, grant the base identity Service Account Token
Creator on the target service account and on every delegate hop.
