package workflows

import (
	"fmt"
	"time"

	"github.com/windkube/gcp-metrics-exporter/internal/temporal/activities"
	"go.temporal.io/sdk/workflow"
)

const ProjectScrapeName = "ScrapeGCPProject"

type ProjectScrapeInput struct {
	ProjectID, ProjectName string
	Locations, Resources   []string
}

func ProjectScrape(ctx workflow.Context, in ProjectScrapeInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 10 * time.Minute})
	for _, location := range in.Locations {
		request := activities.Request{ProjectID: in.ProjectID, ProjectName: in.ProjectName, Location: location}
		for _, resource := range in.Resources {
			var result activities.Result
			var name string
			switch resource {
			case "cloud_run_service":
				name = activities.GetCloudRunServicesName
			case "cloud_run_revision":
				name = activities.GetCloudRunRevisionsName
			case "cloud_run_worker_pool":
				name = activities.GetCloudRunWorkerPoolsName
			default:
				return fmt.Errorf("unknown resource %s", resource)
			}
			if err := workflow.ExecuteActivity(ctx, name, request).Get(ctx, &result); err != nil {
				return err
			}
		}
	}
	return nil
}
