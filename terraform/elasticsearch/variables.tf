variable "template_name" {
  description = "Name of the index template. Its index pattern must match the exporter's index_prefix."
  type        = string
  default     = "gcp-inventory"
}
