package collector

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/windkube/gcp-metrics-exporter/internal/config"
)

func TestRegistryTypes(t *testing.T) {
	r := NewRegistry()
	if len(r.Types()) != 0 {
		t.Fatal("expected empty registry")
	}
}

func TestClientOptionsUsesADCWhenAccessIsOmitted(t *testing.T) {
	opts, err := clientOptions(context.Background(), config.Access{})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) != 0 {
		t.Fatalf("options = %d, want ADC defaults", len(opts))
	}
}

func TestClientOptionsRejectsUnsupportedCredentialType(t *testing.T) {
	_, err := clientOptions(context.Background(), config.Access{Credentials: `{"type":"unknown"}`})
	if err == nil || !strings.Contains(err.Error(), "unsupported type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProviderRejectsUnknownProject(t *testing.T) {
	_, err := NewProvider(nil).Get(context.Background(), "missing")
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("unexpected error: %v", err)
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
