output "service_account_email" {
  description = "Email of the service account that must be attached to the exporter workload."
  value       = google_service_account.exporter.email
}

output "service_account_member" {
  description = "IAM member identifier of the exporter service account."
  value       = "serviceAccount:${google_service_account.exporter.email}"
}
