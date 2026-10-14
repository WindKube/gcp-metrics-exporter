package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/windkube/gcp-metrics-exporter/internal/config"
)

var configCmd = &cobra.Command{Use: "config", Short: "Inspect and validate configuration"}
var configValidateCmd = &cobra.Command{Use: "validate", RunE: func(cmd *cobra.Command, _ []string) error {
	c, err := config.Load(configPath, knownResources)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s is valid: %d projects\n", configPath, len(c.Projects))
	return nil
}}
var configShowCmd = &cobra.Command{Use: "show", RunE: func(cmd *cobra.Command, _ []string) error {
	c, err := config.Load(configPath, knownResources)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(b))
	return nil
}}
var knownResources = []string{"cloud_run_service", "cloud_run_revision", "cloud_run_worker_pool"}

func init() { configCmd.AddCommand(configValidateCmd, configShowCmd) }
