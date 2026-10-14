package temporal_test

import (
	"encoding/json"
	"testing"

	"github.com/windkube/gcp-metrics-exporter/internal/temporal/activities"
	"github.com/windkube/gcp-metrics-exporter/internal/temporal/workflows"
)

func TestActivityNamesAreHumanReadable(t *testing.T) {
	for _, name := range []string{activities.GetCloudRunServicesName, activities.GetCloudRunRevisionsName, activities.GetCloudRunWorkerPoolsName} {
		if name == "GetResource" {
			t.Fatal("generic activity name")
		}
	}
}

func TestWorkflowInputContainsNoAccessConfiguration(t *testing.T) {
	b, err := json.Marshal(workflows.ProjectScrapeInput{ProjectID: "project", ProjectName: "name"})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"ProjectID":"project","ProjectName":"name","Locations":null,"Resources":null}` {
		t.Fatalf("unexpected workflow input: %s", b)
	}
}
