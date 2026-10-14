package collector

import (
	"context"
	"fmt"
	"slices"

	"google.golang.org/api/option"
	"google.golang.org/api/run/v2"
)

const (
	CloudRunService    = "cloud_run_service"
	CloudRunRevision   = "cloud_run_revision"
	CloudRunWorkerPool = "cloud_run_worker_pool"
)

type Resource struct {
	ID, Name string
	Data     any
}
type EmitFunc func(Resource) error

type Collector interface {
	Type() string
	Collect(context.Context, string, string, EmitFunc) error
}

type Registry struct{ byType map[string]Collector }

func NewRegistry(collectors ...Collector) *Registry {
	r := &Registry{byType: make(map[string]Collector, len(collectors))}
	for _, c := range collectors {
		r.byType[c.Type()] = c
	}
	return r
}
func (r *Registry) Get(kind string) (Collector, error) {
	c, ok := r.byType[kind]
	if !ok {
		return nil, fmt.Errorf("unknown resource %q", kind)
	}
	return c, nil
}
func (r *Registry) Types() []string {
	out := make([]string, 0, len(r.byType))
	for k := range r.byType {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

type cloudRun struct {
	api  *run.Service
	kind string
}

func NewCloudRunRegistry(ctx context.Context, opts ...option.ClientOption) (*Registry, error) {
	api, err := run.NewService(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return NewRegistry(&cloudRun{api: api, kind: CloudRunService}, &cloudRun{api: api, kind: CloudRunRevision}, &cloudRun{api: api, kind: CloudRunWorkerPool}), nil
}
func (c *cloudRun) Type() string { return c.kind }
func (c *cloudRun) Collect(ctx context.Context, project, location string, emit EmitFunc) error {
	parent := "projects/" + project + "/locations/" + location
	switch c.kind {
	case CloudRunService:
		return c.api.Projects.Locations.Services.List(parent).Context(ctx).Pages(ctx, func(page *run.GoogleCloudRunV2ListServicesResponse) error {
			for _, x := range page.Services {
				if err := emit(Resource{ID: x.Name, Name: x.Name, Data: x}); err != nil {
					return err
				}
			}
			return nil
		})
	case CloudRunRevision:
		return c.api.Projects.Locations.Services.List(parent).Context(ctx).Pages(ctx, func(services *run.GoogleCloudRunV2ListServicesResponse) error {
			for _, service := range services.Services {
				err := c.api.Projects.Locations.Services.Revisions.List(service.Name).Context(ctx).Pages(ctx, func(page *run.GoogleCloudRunV2ListRevisionsResponse) error {
					for _, x := range page.Revisions {
						if err := emit(Resource{ID: x.Name, Name: x.Name, Data: x}); err != nil {
							return err
						}
					}
					return nil
				})
				if err != nil {
					return err
				}
			}
			return nil
		})
	case CloudRunWorkerPool:
		return c.api.Projects.Locations.WorkerPools.List(parent).Context(ctx).Pages(ctx, func(page *run.GoogleCloudRunV2ListWorkerPoolsResponse) error {
			for _, x := range page.WorkerPools {
				if err := emit(Resource{ID: x.Name, Name: x.Name, Data: x}); err != nil {
					return err
				}
			}
			return nil
		})
	}
	return fmt.Errorf("unsupported resource %q", c.kind)
}
