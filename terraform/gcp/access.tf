resource "google_service_account" "exporter" {
  project      = var.service_account_project_id
  account_id   = var.service_account_id
  display_name = var.service_account_display_name
  description  = "Read-only identity used by gcp-metrics-exporter to inventory Cloud Run resources."
}

resource "google_service_account_key" "exporter" {
  service_account_id = google_service_account.exporter.name
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

resource "google_project_iam_member" "bigquery_job_user" {
  for_each = setunion(var.target_project_ids, var.billing_query_project_ids)

  project = each.value
  role    = "roles/bigquery.jobUser"
  member  = "serviceAccount:${google_service_account.exporter.email}"
}

resource "google_bigquery_dataset_iam_member" "billing_data_viewer" {
  for_each = var.billing_datasets

  project    = each.value.project_id
  dataset_id = each.value.dataset_id
  role       = "roles/bigquery.dataViewer"
  member     = "serviceAccount:${google_service_account.exporter.email}"
}
