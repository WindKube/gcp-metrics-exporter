package queries

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

const tablePlaceholder = "{{TABLE}}"

//go:embed queries/get-billing-data.sql
var getBillingDataQuery string

//go:embed queries/get-billing-data-service.sql
var getBillingDataServiceQuery string

//go:embed queries/get-billing-data-cost-type.sql
var getBillingDataCostTypeQuery string

//go:embed queries/get-billing-data-service-cost-type.sql
var getBillingDataServiceCostTypeQuery string

var tableValidator = newTableValidator()

type BillingDataOptions struct {
	Table           string `validate:"required,bigquery_table"`
	GroupByService  bool
	GroupByCostType bool
}

// ValidateTable verifies a fully-qualified GoogleSQL table or view identifier.
// Identifiers cannot be supplied as BigQuery query parameters.
func ValidateTable(table string) error {
	if err := tableValidator.Struct(BillingDataOptions{Table: table}); err != nil {
		return fmt.Errorf("billing table must be a fully-qualified project.dataset.table identifier")
	}
	return nil
}

// GetBillingData selects a versioned query for the requested breakdown. The
// only SQL substitution is the validated table or view identifier.
func GetBillingData(options BillingDataOptions) (string, error) {
	if err := ValidateTable(options.Table); err != nil {
		return "", err
	}
	return strings.ReplaceAll(selectBillingQuery(options), tablePlaceholder, options.Table), nil
}

func selectBillingQuery(options BillingDataOptions) string {
	switch {
	case options.GroupByService && options.GroupByCostType:
		return getBillingDataServiceCostTypeQuery
	case options.GroupByService:
		return getBillingDataServiceQuery
	case options.GroupByCostType:
		return getBillingDataCostTypeQuery
	default:
		return getBillingDataQuery
	}
}

func newTableValidator() *validator.Validate {
	v := validator.New()
	if err := v.RegisterValidation("bigquery_table", isBigQueryTable); err != nil {
		panic(fmt.Sprintf("register BigQuery table validator: %v", err))
	}
	return v
}

func isBigQueryTable(field validator.FieldLevel) bool {
	parts := strings.Split(field.Field().String(), ".")
	return len(parts) == 3 && validProjectID(parts[0]) && validDatasetID(parts[1]) && validTableID(parts[2])
}

func validProjectID(value string) bool { return validIdentifier(value, "-_") }
func validDatasetID(value string) bool { return validIdentifier(value, "_") }
func validTableID(value string) bool   { return validIdentifier(value, "_$-") }

func validIdentifier(value, extras string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune(extras, r)) {
			return false
		}
	}
	return true
}
