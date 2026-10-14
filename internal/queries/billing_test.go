package queries

import (
	"strings"
	"testing"
)

func TestGetBillingDataSelectsRequestedBreakdown(t *testing.T) {
	tests := []struct {
		name string
		opts BillingDataOptions
		want string
	}{
		{"project total", BillingDataOptions{Table: "billing-project.dataset.export"}, "GROUP BY billing_month, currency"},
		{"service", BillingDataOptions{Table: "billing-project.dataset.export", GroupByService: true}, "GROUP BY billing_month, service, currency"},
		{"cost type", BillingDataOptions{Table: "billing-project.dataset.export", GroupByCostType: true}, "GROUP BY billing_month, cost_type, currency"},
		{"service and cost type", BillingDataOptions{Table: "billing-project.dataset.export", GroupByService: true, GroupByCostType: true}, "GROUP BY billing_month, service, cost_type, currency"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query, err := GetBillingData(test.opts)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"FROM `billing-project.dataset.export`", test.want, "invoice.month >= @invoice_month_start"} {
				if !strings.Contains(query, want) {
					t.Errorf("query missing %q:\n%s", want, query)
				}
			}
		})
	}
}

func TestGetBillingDataRejectsUnsafeTable(t *testing.T) {
	_, err := GetBillingData(BillingDataOptions{Table: "project.dataset.table` WHERE TRUE --"})
	if err == nil {
		t.Fatal("expected unsafe table identifier to be rejected")
	}
}
