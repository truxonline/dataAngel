package sidecar

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RcloneConfig holds configuration for rclone sync
type RcloneConfig struct {
	Paths    []string
	S3Bucket string
	Interval time.Duration
}

// RcloneSyncer manages rclone sync operations
type RcloneSyncer struct {
	config RcloneConfig
	runner CommandRunner
}

// NewRcloneSyncer creates a new rclone syncer
func NewRcloneSyncer(config RcloneConfig) *RcloneSyncer {
	return &RcloneSyncer{
		config: config,
		runner: &DefaultCommandRunner{},
	}
}

// validateYAMLPaths checks that all paths are YAML patterns
func (rs *RcloneSyncer) validateYAMLPaths() bool {
	for _, path := range rs.config.Paths {
		// Check if path contains .yaml or .yml pattern
		if !strings.Contains(path, "*.yaml") && !strings.Contains(path, "*.yml") {
			return false
		}
	}
	return true
}

// buildCommand constructs the rclone sync command for a given path
func (rs *RcloneSyncer) buildCommand(path string) *exec.Cmd {
	// Determine S3 destination based on path
	s3Dest := fmt.Sprintf("s3:%s/%s", rs.config.S3Bucket, filepath.Base(path))

	cmd := exec.Command("rclone", "sync", path, s3Dest, "--progress")

	// Set environment variables for S3 configuration
	env := os.Environ()
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd
}

// StartLoop begins the rclone sync loop with ticker
func (rs *RcloneSyncer) StartLoop(ctx context.Context) {
	ticker := time.NewTicker(rs.config.Interval)
	defer ticker.Stop()

	// Execute immediately on start
	rs.syncAll(ctx)

	// Then execute on each tick
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rs.syncAll(ctx)
		}
	}
}

// syncAll syncs all configured paths
func (rs *RcloneSyncer) syncAll(ctx context.Context) {
	for _, path := range rs.config.Paths {
		cmd := rs.buildCommand(path)
		cmd.Cancel = func() error {
			if cmd.Process != nil {
				return cmd.Process.Signal(os.Interrupt)
			}
			return nil
		}
		cmd.WaitDelay = 15 * time.Second

		// Execute but don't fail the loop on error
		_ = rs.runner.Run(ctx, cmd)
	}
}
