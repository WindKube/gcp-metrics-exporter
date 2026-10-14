package queries

import (
	"bytes"
	_ "embed"
	"fmt"
	"regexp"
	"text/template"
)

//go:embed queries/get-billing-data.sql
var getBillingDataTemplate string

var fullyQualifiedTable = regexp.MustCompile(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_]+\.[A-Za-z0-9_$-]+$`)

type BillingDataOptions struct {
	Table           string
	GroupByService  bool
	GroupByCostType bool
}

// ValidateTable verifies a fully-qualified GoogleSQL table or view identifier.
// Identifiers cannot be supplied as BigQuery query parameters.
func ValidateTable(table string) error {
	if !fullyQualifiedTable.MatchString(table) {
		return fmt.Errorf("billing table must be a fully-qualified project.dataset.table identifier")
	}
	return nil
}

// GetBillingData renders the versioned billing query using only validated table
// identifiers and code-controlled SQL expressions.
func GetBillingData(options BillingDataOptions) (string, error) {
	if err := ValidateTable(options.Table); err != nil {
		return "", err
	}
	data := struct {
		Table              string
		ServiceExpression  string
		CostTypeExpression string
		GroupByService     string
		GroupByCostType    string
	}{
		Table:              options.Table,
		ServiceExpression:  "'all'",
		CostTypeExpression: "'all'",
	}
	if options.GroupByService {
		data.ServiceExpression = "COALESCE(service.description, 'unknown')"
		data.GroupByService = ", service"
	}
	if options.GroupByCostType {
		data.CostTypeExpression = "COALESCE(cost_type, 'unknown')"
		data.GroupByCostType = ", cost_type"
	}
	t, err := template.New("get-billing-data.sql").Option("missingkey=error").Parse(getBillingDataTemplate)
	if err != nil {
		return "", fmt.Errorf("parse billing query: %w", err)
	}
	var query bytes.Buffer
	if err := t.Execute(&query, data); err != nil {
		return "", fmt.Errorf("render billing query: %w", err)
	}
	return query.String(), nil
}
