package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/windkube/gcp-metrics-exporter/internal/config"
	temporalclient "github.com/windkube/gcp-metrics-exporter/internal/temporal"
	"github.com/windkube/gcp-metrics-exporter/internal/temporal/schedules"
	"go.temporal.io/sdk/client"
	"strings"
)

var scheduleProject string
var scheduleCmd = &cobra.Command{Use: "schedule", Short: "Manage project Temporal schedules"}

func withTemporal(cmd *cobra.Command, run func(client.Client, *config.Config) error) error {
	cfg, err := config.Load(configPath, knownResources)
	if err != nil {
		return err
	}
	c, err := temporalclient.NewClient(cfg.Global.Temporal)
	if err != nil {
		return err
	}
	defer c.Close()
	return run(c, cfg)
}

var scheduleApplyCmd = &cobra.Command{Use: "apply", RunE: func(cmd *cobra.Command, _ []string) error {
	return withTemporal(cmd, func(c client.Client, cfg *config.Config) error {
		changes, err := schedules.NewReconciler(c, cfg.Global.Temporal.TaskQueue).Apply(cmd.Context(), cfg)
		for _, change := range changes {
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", change.Action, change.ID)
		}
		return err
	})
}}
var scheduleListCmd = &cobra.Command{Use: "list", RunE: func(cmd *cobra.Command, _ []string) error {
	return withTemporal(cmd, func(c client.Client, _ *config.Config) error {
		iter, err := c.ScheduleClient().List(cmd.Context(), client.ScheduleListOptions{PageSize: 100})
		if err != nil {
			return err
		}
		for iter.HasNext() {
			entry, err := iter.Next()
			if err != nil {
				return err
			}
			if strings.HasPrefix(entry.ID, schedules.IDPrefix) {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\tpaused=%t\tnext=%v\n", entry.ID, entry.Paused, entry.NextActionTimes)
			}
		}
		return nil
	})
}}

func actionCmd(use string, action func(client.ScheduleHandle) error) *cobra.Command {
	cmd := &cobra.Command{Use: use, RunE: func(cmd *cobra.Command, _ []string) error {
		return withTemporal(cmd, func(c client.Client, _ *config.Config) error {
			return action(c.ScheduleClient().GetHandle(cmd.Context(), schedules.ID(scheduleProject)))
		})
	}}
	cmd.Flags().StringVar(&scheduleProject, "project", "", "configured project name")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}
func init() {
	scheduleCmd.AddCommand(scheduleApplyCmd, scheduleListCmd, actionCmd("pause", func(h client.ScheduleHandle) error {
		return h.Pause(context.Background(), client.SchedulePauseOptions{Note: "paused via gcp-metrics-exporter"})
	}), actionCmd("resume", func(h client.ScheduleHandle) error {
		return h.Unpause(context.Background(), client.ScheduleUnpauseOptions{Note: "resumed via gcp-metrics-exporter"})
	}), actionCmd("trigger", func(h client.ScheduleHandle) error {
		return h.Trigger(context.Background(), client.ScheduleTriggerOptions{})
	}))
}
