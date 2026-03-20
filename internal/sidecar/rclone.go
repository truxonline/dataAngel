package sidecar

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
)

// RcloneRunner manages periodic rclone sync operations
type RcloneRunner struct {
	Interval   time.Duration
	FsPaths    []string
	Bucket     string
	S3Endpoint string
	runner     CommandRunner
	metrics    *SidecarMetrics
}

// NewRcloneRunner creates a new runner
func NewRcloneRunner(interval time.Duration, fsPaths []string, bucket, s3Endpoint string) *RcloneRunner {
	return &RcloneRunner{
		Interval:   interval,
		FsPaths:    fsPaths,
		Bucket:     bucket,
		S3Endpoint: s3Endpoint,
		runner:     &realCommandRunner{},
		metrics:    GetMetrics(),
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
			start := time.Now()
			err := r.syncAll(ctx)

			if r.metrics != nil {
				duration := time.Since(start).Seconds()
				r.metrics.SyncDuration.Observe(duration)

				if err != nil {
					r.metrics.SyncsFailed.Inc()
					r.metrics.RcloneUp.Set(0)
				} else {
					r.metrics.SyncsTotal.Inc()
					r.metrics.RcloneUp.Set(1)
				}
			}

			if err != nil {
				fmt.Printf("rclone sync error: %v\n", err)
			}
		}
	}
}

// syncAll syncs all configured filesystem paths to S3.
func (r *RcloneRunner) syncAll(ctx context.Context) error {
	for _, fsPath := range r.FsPaths {
		if err := r.syncOnce(ctx, fsPath); err != nil {
			return err
		}
	}
	return nil
}

// syncOnce syncs a single filesystem path to its matching S3 prefix.
// Uses filepath.Base(fsPath) as the S3 prefix to match restore.go (issue #30).
func (r *RcloneRunner) syncOnce(ctx context.Context, fsPath string) error {
	s3Prefix := filepath.Base(fsPath)
	args := []string{
		"sync",
		fsPath,
		fmt.Sprintf(":s3:%s/%s", r.Bucket, s3Prefix),
		"--s3-env-auth",
		"--exclude", "*.db*",
		"--exclude", ".*.db-litestream/**",
		"--checksum",
		"--min-age", "30s",
		"--timeout", "120s",
		"--contimeout", "30s",
		"--transfers", "1",
		"--checkers", "2",
		"--low-level-retries", "3",
	}

	if r.S3Endpoint != "" {
		args = append(args, "--s3-provider", "Minio", "--s3-endpoint", r.S3Endpoint)
	} else {
		args = append(args, "--s3-provider", "AWS")
	}

	syncCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	return r.runner.Run(syncCtx, "rclone", args...)
}
