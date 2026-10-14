package collector

import (
	"testing"
	"time"
)

func TestRegistryTypes(t *testing.T) {
	r := NewRegistry()
	if len(r.Types()) != 0 {
		t.Fatal("expected empty registry")
	}
}

func TestInvoiceMonthStartIncludesPreviousInvoiceMonth(t *testing.T) {
	if got := invoiceMonthStart(time.Date(2026, time.January, 14, 0, 0, 0, 0, time.UTC)); got != "202512" {
		t.Fatalf("invoice month start = %q, want 202512", got)
	}
}

func TestBillingEntryIDIncludesAllDimensions(t *testing.T) {
	entry := BillingEntry{BillingMonth: "202610", Service: "Compute Engine", CostType: "regular", Currency: "USD"}
	if got, want := entry.ID(), "202610:Compute Engine:regular:USD"; got != want {
		t.Fatalf("ID = %q, want %q", got, want)
	}
}
