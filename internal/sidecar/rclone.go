package sidecar

import (
	"context"
	"fmt"
	"time"
)

// RcloneRunner manages periodic rclone sync operations
type RcloneRunner struct {
	Interval   time.Duration
	FsPaths    []string
	Bucket     string
	S3Endpoint string
	runner     CommandRunner
}

// NewRcloneRunner creates a new runner
func NewRcloneRunner(interval time.Duration, fsPaths []string, bucket, s3Endpoint string) *RcloneRunner {
	return &RcloneRunner{
		Interval:   interval,
		FsPaths:    fsPaths,
		Bucket:     bucket,
		S3Endpoint: s3Endpoint,
		runner:     &realCommandRunner{},
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
	args := []string{
		"sync",
		r.FsPaths[0],
		fmt.Sprintf(":s3:%s/filesystem", r.Bucket),
		"--s3-provider", "AWS",
		"--s3-env-auth",
		"--checksum",
		"--min-age", "30s",
	}

	if r.S3Endpoint != "" {
		args = append(args, "--s3-endpoint", r.S3Endpoint)
	}

	return r.runner.Run(ctx, "rclone", args...)
}
