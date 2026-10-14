# Terraform example

Creates a dedicated service account for `gcp-metrics-exporter` and grants it the
least-privileged predefined role needed to inventory the configured Cloud Run
resources: `roles/run.viewer` in every target project. The role includes the
list permissions for Cloud Run services, revisions, and worker pools used by the
exporter.

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
```

Authenticate Terraform with Application Default Credentials, then apply:

```bash
gcloud auth application-default login
terraform init
terraform apply
```

Terraform also creates a service-account key for the exporter. Write the
sensitive `service_account_key` output to a JSON file and set `GCME_ADC_FILE`
to its absolute path before starting the Docker Compose deployment:

```bash
terraform output -raw service_account_key > gcp-metrics-exporter.json
export GCME_ADC_FILE="$PWD/gcp-metrics-exporter.json"
```

The exporter reads this JSON file as Application Default Credentials through
`GOOGLE_APPLICATION_CREDENTIALS`. Keep it out of version control and store it
in a secret manager for production deployments. Rotate it by replacing the
Terraform-managed key when required. The private key is also stored in
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
