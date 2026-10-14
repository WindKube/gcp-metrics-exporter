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

Attach the emitted `service_account_email` to the workload running the exporter.
For example, on GKE, map this Google service account to the Kubernetes service
account through Workload Identity. Do not create a service-account key: the
exporter already uses Application Default Credentials.

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

For billing collection, grant the exporter identity (or the impersonated target
identity) BigQuery Job User on each configured `projects[].billing.query_project`
and read access to that project's billing-export table. When using
`projects[].access.impersonation`, grant the base identity Service Account Token
Creator on the target service account and on every delegate hop.
