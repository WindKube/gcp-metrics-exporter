package temporal

import (
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/windkube/gcp-metrics-exporter/internal/config"
	"github.com/windkube/gcp-metrics-exporter/internal/temporal/activities"
	"github.com/windkube/gcp-metrics-exporter/internal/temporal/workflows"
)

func NewClient(cfg config.Temporal) (client.Client, error) {
	return client.Dial(client.Options{HostPort: cfg.Address, Namespace: cfg.Namespace})
}
func NewWorker(c client.Client, cfg config.Temporal, acts *activities.Activities) worker.Worker {
	w := worker.New(c, cfg.TaskQueue, worker.Options{})
	w.RegisterWorkflowWithOptions(workflows.ProjectScrape, workflow.RegisterOptions{Name: workflows.ProjectScrapeName})
	w.RegisterActivityWithOptions(acts.GetCloudRunServices, activity.RegisterOptions{Name: activities.GetCloudRunServicesName})
	w.RegisterActivityWithOptions(acts.GetCloudRunRevisions, activity.RegisterOptions{Name: activities.GetCloudRunRevisionsName})
	w.RegisterActivityWithOptions(acts.GetCloudRunWorkerPools, activity.RegisterOptions{Name: activities.GetCloudRunWorkerPoolsName})
	return w
}
