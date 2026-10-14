# Elasticsearch index template

Creates the index template for documents written by GCP Metrics Exporter. The
mapping is defined in `index_template.tf`, alongside the Terraform resource, so
this module is the single source of truth.

The default template applies to `gcp-inventory-*`, which matches the example
configuration's `storage.elasticsearch.index_prefix`. If a different prefix is
configured, update `local.index_template.index_patterns` to match it before
applying the module.

```bash
export ELASTICSEARCH_ENDPOINTS=https://my-cluster.es.eu-west-1.aws.cloud.es.io:443
export ELASTICSEARCH_API_KEY=...

terraform init
terraform apply
```

Username and password authentication also work through
`ELASTICSEARCH_USERNAME` and `ELASTICSEARCH_PASSWORD`. Refer to the
[Elasticstack provider documentation](https://registry.terraform.io/providers/elastic/elasticstack/latest/docs)
for other authentication options.

The template maps the document envelope (`@timestamp`, `resource`, and `cloud`)
explicitly and stores the provider's source object in the `gcp` `flattened`
field. Apply it before the exporter writes its first document.
