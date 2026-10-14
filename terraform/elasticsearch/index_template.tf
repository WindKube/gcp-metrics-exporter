resource "elasticstack_elasticsearch_index_template" "inventory" {
  name           = var.template_name
  index_patterns = var.index_patterns
  priority       = var.priority

  template {
    mappings = file("${path.module}/index-mapping.json")
  }
}
