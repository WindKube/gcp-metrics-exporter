package queries

import (
	"strings"
	"testing"
)

func TestGetBillingDataRendersRequestedBreakdowns(t *testing.T) {
	query, err := GetBillingData(BillingDataOptions{Table: "billing-project.dataset.export", GroupByService: true, GroupByCostType: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"FROM `billing-project.dataset.export`",
		"COALESCE(service.description, 'unknown') AS service",
		"COALESCE(cost_type, 'unknown') AS cost_type",
		"GROUP BY billing_month, service, cost_type, currency",
		"invoice.month >= @invoice_month_start",
	} {
		if !strings.Contains(query, want) {
			t.Errorf("query missing %q:\n%s", want, query)
		}
	}
}

func TestGetBillingDataRejectsUnsafeTable(t *testing.T) {
	_, err := GetBillingData(BillingDataOptions{Table: "project.dataset.table` WHERE TRUE --"})
	if err == nil {
		t.Fatal("expected unsafe table identifier to be rejected")
	}
}
