package cmd

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/windkube/gcp-metrics-exporter/internal/collector"
	"github.com/windkube/gcp-metrics-exporter/internal/config"
	"github.com/windkube/gcp-metrics-exporter/internal/storage"
	temporalclient "github.com/windkube/gcp-metrics-exporter/internal/temporal"
	"github.com/windkube/gcp-metrics-exporter/internal/temporal/activities"
	"github.com/windkube/gcp-metrics-exporter/internal/temporal/schedules"
)

var workerCmd = &cobra.Command{Use: "worker", Short: "Run the Temporal worker and reconcile schedules", RunE: func(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	cfg, err := config.Load(configPath, knownResources)
	if err != nil {
		return err
	}
	store := storage.New(cfg.Storage.Elasticsearch)
	temporal, err := temporalclient.NewClient(cfg.Global.Temporal)
	if err != nil {
		return err
	}
	defer temporal.Close()
	w := temporalclient.NewWorker(temporal, cfg.Global.Temporal, activities.New(collector.NewProvider(cfg.Projects), store))
	if err := w.Start(); err != nil {
		return err
	}
	defer w.Stop()
	changes, err := schedules.NewReconciler(temporal, cfg.Global.Temporal.TaskQueue).Apply(ctx, cfg)
	if err != nil {
		return err
	}
	for _, change := range changes {
		log.Info().Str("schedule", change.ID).Str("action", change.Action).Msg("reconciled schedule")
	}
	<-ctx.Done()
	return context.Cause(ctx)
}}
