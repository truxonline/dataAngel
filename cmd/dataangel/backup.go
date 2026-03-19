package main

import (
	"context"

	"github.com/charchess/dataAngel/internal/sidecar"
)

// RunBackup executes the backup phase (continuous daemon with litestream + rclone)
func RunBackup(ctx context.Context, config Config) error {
	// Convert to sidecar config
	sidecarConfig := config.ToSidecarConfig()

	// Create and start daemon
	daemon := sidecar.NewDaemon(sidecarConfig)
	return daemon.Start(ctx)
}
