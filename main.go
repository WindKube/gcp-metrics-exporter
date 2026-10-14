package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/windkube/gcp-metrics-exporter/cmd"
)

var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(cmd.Execute(ctx, version))
}
