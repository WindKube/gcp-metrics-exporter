package activities

import (
	"context"
	"time"

	"github.com/windkube/gcp-metrics-exporter/internal/collector"
	"github.com/windkube/gcp-metrics-exporter/internal/storage"
)

const (
	GetCloudRunServicesName    = "GetCloudRunServices"
	GetCloudRunRevisionsName   = "GetCloudRunRevisions"
	GetCloudRunWorkerPoolsName = "GetCloudRunWorkerPools"
)

type Request struct{ ProjectID, ProjectName, Location string }
type Result struct {
	Count int `json:"count"`
}
type Activities struct {
	registry *collector.Registry
	store    storage.Store
}

func New(registry *collector.Registry, store storage.Store) *Activities {
	return &Activities{registry: registry, store: store}
}
func (a *Activities) GetCloudRunServices(ctx context.Context, req Request) (Result, error) {
	return a.collect(ctx, collector.CloudRunService, req)
}
func (a *Activities) GetCloudRunRevisions(ctx context.Context, req Request) (Result, error) {
	return a.collect(ctx, collector.CloudRunRevision, req)
}
func (a *Activities) GetCloudRunWorkerPools(ctx context.Context, req Request) (Result, error) {
	return a.collect(ctx, collector.CloudRunWorkerPool, req)
}
func (a *Activities) collect(ctx context.Context, kind string, req Request) (Result, error) {
	c, err := a.registry.Get(kind)
	if err != nil {
		return Result{}, err
	}
	now := time.Now().UTC()
	docs := make([]storage.Document, 0)
	err = c.Collect(ctx, req.ProjectID, req.Location, func(r collector.Resource) error {
		d := storage.Document{ID: req.ProjectID + ":" + req.Location + ":" + kind + ":" + r.ID, Timestamp: now, GCP: r.Data}
		d.Resource.Type, d.Resource.ID, d.Resource.Name = kind, r.ID, r.Name
		d.Cloud.Provider, d.Cloud.Project, d.Cloud.Location = "gcp", req.ProjectID, req.Location
		docs = append(docs, d)
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	if len(docs) > 0 {
		if err := a.store.Write(ctx, docs); err != nil {
			return Result{}, err
		}
	}
	return Result{Count: len(docs)}, nil
}
