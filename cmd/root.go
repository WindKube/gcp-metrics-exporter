package cmd

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	configPath string
	version    = "dev"
)
var rootCmd = &cobra.Command{Use: "gcp-metrics-exporter", Short: "Export GCP Cloud Run inventory", SilenceUsage: true, SilenceErrors: true}

func Execute(ctx context.Context, v string) int {
	version = v
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		log.Error().Err(err).Msg("command failed")
		return 1
	}
	return 0
}
func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config.yaml", "path to the configuration file")
	rootCmd.AddCommand(versionCmd, configCmd, workerCmd, scheduleCmd)
}

var versionCmd = &cobra.Command{Use: "version", Short: "Print the build version", Run: func(cmd *cobra.Command, _ []string) { fmt.Fprintln(cmd.OutOrStdout(), version) }}
