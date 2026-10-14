package collector

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"

	"github.com/windkube/gcp-metrics-exporter/internal/config"
	"github.com/windkube/gcp-metrics-exporter/internal/queries"
)

const Billing = "billing"

type billing struct {
	client *bigquery.Client
	cfg    config.Billing
	query  string
}

// NewGoogleRegistry constructs collectors for all configured Google APIs. A
// BigQuery client is only created when billing collection is configured.
func NewGoogleRegistry(ctx context.Context, billingConfig config.Billing) (*Registry, error) {
	runRegistry, err := NewCloudRunRegistry(ctx)
	if err != nil {
		return nil, err
	}
	if billingConfig.Table == "" {
		return runRegistry, nil
	}
	query, err := queries.GetBillingData(queries.BillingDataOptions{
		Table:           billingConfig.Table,
		GroupByService:  billingConfig.GroupByService,
		GroupByCostType: billingConfig.GroupByCostType,
	})
	if err != nil {
		return nil, err
	}
	client, err := bigquery.NewClient(ctx, billingConfig.QueryProject)
	if err != nil {
		return nil, fmt.Errorf("create BigQuery client: %w", err)
	}
	client.Location = billingConfig.Location
	collectors := make([]Collector, 0, len(runRegistry.byType)+1)
	for _, kind := range runRegistry.Types() {
		collectors = append(collectors, runRegistry.byType[kind])
	}
	collectors = append(collectors, &billing{client: client, cfg: billingConfig, query: query})
	return NewRegistry(collectors...), nil
}

func (c *billing) Type() string { return Billing }

func (c *billing) Collect(ctx context.Context, project, _ string, emit EmitFunc) error {
	query := c.client.Query(c.query)
	query.Parameters = []bigquery.QueryParameter{
		{Name: "project_id", Value: project},
		{Name: "invoice_month_start", Value: invoiceMonthStart(time.Now().UTC())},
	}
	rows, err := query.Read(ctx)
	if err != nil {
		return fmt.Errorf("query billing export: %w", err)
	}
	for {
		var entry BillingEntry
		err := rows.Next(&entry)
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read billing query row: %w", err)
		}
		if err := emit(Resource{ID: entry.ID(), Name: entry.Service, Data: entry}); err != nil {
			return err
		}
	}
}

func invoiceMonthStart(now time.Time) string {
	return now.AddDate(0, -1, 0).Format("200601")
}

// BillingEntry is emitted once per invoice month, either for the entire target
// project or for each configured service and cost-type breakdown.
type BillingEntry struct {
	BillingMonth string `bigquery:"billing_month" json:"billing_month"`
	Service      string `bigquery:"service" json:"service"`
	CostType     string `bigquery:"cost_type" json:"cost_type"`
	Currency     string `bigquery:"currency" json:"currency"`
	Cost         string `bigquery:"cost" json:"cost"`
}

func (e BillingEntry) ID() string {
	return e.BillingMonth + ":" + e.Service + ":" + e.CostType + ":" + e.Currency
}
