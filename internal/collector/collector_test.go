package collector

import (
	"strings"
	"testing"

	"github.com/windkube/gcp-metrics-exporter/internal/config"
)

func TestRegistryTypes(t *testing.T) {
	r := NewRegistry()
	if len(r.Types()) != 0 {
		t.Fatal("expected empty registry")
	}
}

func TestBillingQueryGroupsByServiceOnlyWhenRequested(t *testing.T) {
	base := config.Billing{Table: "billing-project.dataset.export"}
	query := billingQuery(base)
	if !strings.Contains(query, "'all' AS service") || strings.Contains(query, "GROUP BY billing_month, service") {
		t.Fatalf("project-total query has unexpected grouping: %s", query)
	}
	query = billingQuery(config.Billing{Table: base.Table, GroupByService: true})
	if !strings.Contains(query, "COALESCE(service.description, 'unknown')") || !strings.Contains(query, "GROUP BY billing_month, service") {
		t.Fatalf("service query has unexpected grouping: %s", query)
	}
}
