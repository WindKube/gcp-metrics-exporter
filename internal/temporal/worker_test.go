package temporal_test

import (
	"github.com/windkube/gcp-metrics-exporter/internal/temporal/activities"
	"testing"
)

func TestActivityNamesAreHumanReadable(t *testing.T) {
	for _, name := range []string{activities.GetCloudRunServicesName, activities.GetCloudRunRevisionsName, activities.GetCloudRunWorkerPoolsName} {
		if name == "GetResource" {
			t.Fatal("generic activity name")
		}
	}
}
