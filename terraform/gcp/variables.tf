variable "service_account_project_id" {
  description = "Project in which to create the exporter service account."
  type        = string
}

variable "target_project_ids" {
  description = "Projects the exporter inventories. The service account receives Cloud Run Viewer in each one."
  type        = set(string)
}

variable "service_account_id" {
  description = "Account ID of the service account used by gcp-metrics-exporter."
  type        = string
  default     = "gcp-metrics-exporter"
}

variable "service_account_display_name" {
  description = "Display name of the service account used by gcp-metrics-exporter."
  type        = string
  default     = "GCP Metrics Exporter"
}
