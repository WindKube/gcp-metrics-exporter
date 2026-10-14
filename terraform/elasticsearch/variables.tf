variable "template_name" {
  description = "Name of the index template. Its index pattern must match the exporter's index_prefix."
  type        = string
  default     = "gcp-inventory"
}

variable "index_patterns" {
  description = "Index patterns to which the template applies. They must match the exporter's index_prefix."
  type        = list(string)
  default     = ["gcp-inventory-*"]
}

variable "priority" {
  description = "Priority of the Elasticsearch index template."
  type        = number
  default     = 100
}
