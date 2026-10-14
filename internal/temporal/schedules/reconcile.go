package schedules

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/windkube/gcp-metrics-exporter/internal/config"
	"github.com/windkube/gcp-metrics-exporter/internal/temporal/workflows"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

const IDPrefix = "gcme-project-"

func ID(project string) string { return IDPrefix + project }

type Change struct{ ID, Action string }
type Reconciler struct {
	client client.Client
	queue  string
}

func NewReconciler(c client.Client, queue string) *Reconciler {
	return &Reconciler{client: c, queue: queue}
}
func (r *Reconciler) Apply(ctx context.Context, cfg *config.Config) ([]Change, error) {
	existing, err := r.list(ctx)
	if err != nil {
		return nil, err
	}
	desired := map[string]config.Project{}
	for _, p := range cfg.Projects {
		desired[ID(p.Name)] = p
	}
	changes := []Change{}
	for id, p := range desired {
		action := "updated"
		if _, ok := existing[id]; !ok {
			action = "created"
		}
		if err := r.upsert(ctx, id, p, cfg.Global.ScheduleTimezone); err != nil {
			return changes, err
		}
		changes = append(changes, Change{ID: id, Action: action})
	}
	for id := range existing {
		if _, ok := desired[id]; !ok {
			if err := r.client.ScheduleClient().GetHandle(ctx, id).Delete(ctx); err != nil {
				return changes, err
			}
			changes = append(changes, Change{ID: id, Action: "deleted"})
		}
	}
	return changes, nil
}
func (r *Reconciler) list(ctx context.Context) (map[string]struct{}, error) {
	iter, err := r.client.ScheduleClient().List(ctx, client.ScheduleListOptions{PageSize: 100})
	if err != nil {
		return nil, err
	}
	ids := map[string]struct{}{}
	for iter.HasNext() {
		entry, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(entry.ID, IDPrefix) {
			ids[entry.ID] = struct{}{}
		}
	}
	return ids, nil
}
func (r *Reconciler) upsert(ctx context.Context, id string, p config.Project, timezone string) error {
	spec := client.ScheduleSpec{CronExpressions: []string{p.ScrapeSchedule}, TimeZoneName: timezone}
	action := &client.ScheduleWorkflowAction{ID: "gcme-scrape-" + p.Name, Workflow: workflows.ProjectScrapeName, TaskQueue: r.queue, Args: []any{workflows.ProjectScrapeInput{ProjectID: p.ID, ProjectName: p.Name, Locations: p.Locations, Resources: p.Resources}}}
	opts := client.ScheduleOptions{ID: id, Spec: spec, Action: action, Overlap: enumspb.SCHEDULE_OVERLAP_POLICY_SKIP, CatchupWindow: time.Hour}
	_, err := r.client.ScheduleClient().Create(ctx, opts)
	if err == nil {
		return nil
	}
	if !errors.Is(err, temporal.ErrScheduleAlreadyRunning) {
		return fmt.Errorf("creating schedule %s: %w", id, err)
	}
	return r.client.ScheduleClient().GetHandle(ctx, id).Update(ctx, client.ScheduleUpdateOptions{DoUpdate: func(in client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
		in.Description.Schedule.Spec = &spec
		in.Description.Schedule.Action = action
		if in.Description.Schedule.Policy == nil {
			in.Description.Schedule.Policy = &client.SchedulePolicies{}
		}
		in.Description.Schedule.Policy.Overlap = enumspb.SCHEDULE_OVERLAP_POLICY_SKIP
		in.Description.Schedule.Policy.CatchupWindow = time.Hour
		return &client.ScheduleUpdate{Schedule: &in.Description.Schedule}, nil
	}})
}
