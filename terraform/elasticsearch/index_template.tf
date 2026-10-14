locals {
  index_template = {
    index_patterns = ["gcp-inventory-*"]
    priority       = 100
    mappings = {
      properties = {
        "@timestamp" = { type = "date" }
        resource = {
          properties = {
            type = { type = "keyword" }
            id   = { type = "keyword" }
            name = { type = "keyword" }
          }
        }
        cloud = {
          properties = {
            provider = { type = "keyword" }
            project  = { type = "keyword" }
            location = { type = "keyword" }
          }
        }
        gcp = { type = "flattened" }
      }
    }
  }
}

resource "elasticstack_elasticsearch_index_template" "inventory" {
  name           = var.template_name
  index_patterns = local.index_template.index_patterns
  priority       = local.index_template.priority

  template {
    mappings = jsonencode(local.index_template.mappings)
  }
}
