resource "google_service_account" "exporter" {
  project      = var.service_account_project_id
  account_id   = var.service_account_id
  display_name = var.service_account_display_name
  description  = "Read-only identity used by gcp-metrics-exporter to inventory Cloud Run resources."
}

# This non-authoritative resource preserves the rest of each project's IAM
# policy. Cloud Run Viewer includes the service, revision, and worker-pool list
# permissions used by the exporter.
resource "google_project_iam_member" "cloud_run_viewer" {
  for_each = var.target_project_ids

  project = each.value
  role    = "roles/run.viewer"
  member  = "serviceAccount:${google_service_account.exporter.email}"
}
