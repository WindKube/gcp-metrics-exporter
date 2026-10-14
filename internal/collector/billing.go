package collector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/bigquery/v2"
	"google.golang.org/api/googleapi"

	"github.com/windkube/gcp-metrics-exporter/internal/config"
)

const Billing = "billing"

type billing struct {
	api *bigquery.Service
	cfg config.Billing
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
	billingAPI, err := bigquery.NewService(ctx)
	if err != nil {
		return nil, err
	}
	collectors := make([]Collector, 0, len(runRegistry.byType)+1)
	for _, kind := range runRegistry.Types() {
		collectors = append(collectors, runRegistry.byType[kind])
	}
	collectors = append(collectors, &billing{api: billingAPI, cfg: billingConfig})
	return NewRegistry(collectors...), nil
}

func (c *billing) Type() string { return Billing }

func (c *billing) Collect(ctx context.Context, project, _ string, emit EmitFunc) error {
	response, err := c.api.Jobs.Query(c.cfg.QueryProject, &bigquery.QueryRequest{
		Query:         billingQuery(c.cfg),
		UseLegacySql:  googleapi.Bool(false),
		ParameterMode: "NAMED",
		QueryParameters: []*bigquery.QueryParameter{{
			Name:           "project_id",
			ParameterType:  &bigquery.QueryParameterType{Type: "STRING"},
			ParameterValue: &bigquery.QueryParameterValue{Value: project},
		}},
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("query billing export: %w", err)
	}
	rows, err := billingRows(ctx, c.api, response)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if len(row.F) != 4 {
			return fmt.Errorf("unexpected billing query row with %d columns", len(row.F))
		}
		entry := BillingEntry{
			BillingMonth: cellValue(row.F[0]),
			Service:      cellValue(row.F[1]),
			Cost:         cellValue(row.F[2]),
			Currency:     cellValue(row.F[3]),
		}
		if err := emit(Resource{ID: entry.BillingMonth + ":" + entry.Service, Name: entry.Service, Data: entry}); err != nil {
			return err
		}
	}
	return nil
}

func billingRows(ctx context.Context, api *bigquery.Service, response *bigquery.QueryResponse) ([]*bigquery.TableRow, error) {
	if response.JobComplete {
		return response.Rows, nil
	}
	if response.JobReference == nil {
		return nil, fmt.Errorf("query billing export did not return a job reference")
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			result, err := api.Jobs.GetQueryResults(response.JobReference.ProjectId, response.JobReference.JobId).
				Location(response.JobReference.Location).Context(ctx).Do()
			if err != nil {
				return nil, fmt.Errorf("read billing query result: %w", err)
			}
			if result.JobComplete {
				return result.Rows, nil
			}
		}
	}
}

// BillingEntry is emitted once per invoice month, either for the entire target
// project or for each service when group_by_service is enabled.
type BillingEntry struct {
	BillingMonth string `json:"billing_month"`
	Service      string `json:"service"`
	Cost         string `json:"cost"`
	Currency     string `json:"currency"`
}

func billingQuery(cfg config.Billing) string {
	service := "'all'"
	groupBy := ""
	if cfg.GroupByService {
		service = "COALESCE(service.description, 'unknown')"
		groupBy = ", service"
	}
	return fmt.Sprintf("\nSELECT\n"+
		"  invoice.month AS billing_month,\n"+
		"  %s AS service,\n"+
		"  CAST(SUM(cost + IFNULL((SELECT SUM(credit.amount) FROM UNNEST(credits) AS credit), 0)) AS STRING) AS cost,\n"+
		"  ANY_VALUE(currency) AS currency\n"+
		"FROM `%s`\n"+
		"WHERE project.id = @project_id\n"+
		"  AND invoice.month >= FORMAT_DATE('%%Y%%m', DATE_SUB(CURRENT_DATE(), INTERVAL 1 MONTH))\n"+
		"GROUP BY billing_month%s\n"+
		"ORDER BY billing_month, service", service, strings.ReplaceAll(cfg.Table, "`", ""), groupBy)
}

func cellValue(cell *bigquery.TableCell) string {
	if cell == nil || cell.V == nil {
		return ""
	}
	return fmt.Sprint(cell.V)
}
