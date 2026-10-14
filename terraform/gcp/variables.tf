variable "service_account_project_id" {
  description = "Project in which to create the exporter service account."
  type        = string
}

variable "target_project_ids" {
  description = "Projects the exporter inventories. The service account receives Cloud Run Viewer and BigQuery Job User in each one."
  type        = set(string)
}

variable "billing_query_project_ids" {
  description = "Additional BigQuery query projects used by billing.query_project. Target projects are included automatically."
  type        = set(string)
  default     = []
}

variable "billing_datasets" {
  description = "BigQuery datasets containing billing exports that the exporter can read."
  type = set(object({
    project_id = string
    dataset_id = string
  }))
  default = []
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
