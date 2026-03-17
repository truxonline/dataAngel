package sidecar

import (
	"context"
	"fmt"
	"time"
)

// RcloneRunner manages periodic rclone sync operations
type RcloneRunner struct {
	Interval time.Duration
	FsPaths  []string
	Bucket   string
	runner   CommandRunner
}

// NewRcloneRunner creates a new runner
func NewRcloneRunner(interval time.Duration, fsPaths []string, bucket string) *RcloneRunner {
	return &RcloneRunner{
		Interval: interval,
		FsPaths:  fsPaths,
		Bucket:   bucket,
		runner:   &realCommandRunner{},
	}
}

// Start runs rclone sync loop (blocking until context cancel)
func (r *RcloneRunner) Start(ctx context.Context) error {
	if len(r.FsPaths) == 0 {
		// No paths to sync - wait for context cancel
		<-ctx.Done()
		return ctx.Err()
	}

	ticker := time.NewTicker(r.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Run sync (non-blocking - errors logged but don't stop loop)
			err := r.syncOnce(ctx)
			if err != nil {
				// Log error but continue (rclone errors shouldn't stop daemon)
				fmt.Printf("rclone sync error: %v\n", err)
			}
		}
	}
}

func (r *RcloneRunner) syncOnce(ctx context.Context) error {
	// Build rclone args
	args := []string{
		"sync",
		r.FsPaths[0], // TODO: handle multiple paths
		fmt.Sprintf("s3:%s/filesystem", r.Bucket),
		"--checksum",
		"--min-age", "30s",
	}

	return r.runner.Run(ctx, "rclone", args...)
}
