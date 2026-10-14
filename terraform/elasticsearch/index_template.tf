locals {
  index_template = {
    index_patterns = ["gcp-inventory-*"]
    priority       = 100
  }
}

resource "elasticstack_elasticsearch_index_template" "inventory" {
  name           = var.template_name
  index_patterns = local.index_template.index_patterns
  priority       = local.index_template.priority

  template {
    mappings = file("${path.module}/index-mapping.json")
  }
}
