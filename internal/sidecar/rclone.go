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
			err := r.syncOnce(ctx)

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

func (r *RcloneRunner) syncOnce(ctx context.Context) error {
	args := []string{
		"sync",
		r.FsPaths[0],
		fmt.Sprintf(":s3:%s/filesystem", r.Bucket),
		"--s3-provider", "AWS",
		"--s3-env-auth",
		"--checksum",
		"--min-age", "30s",
		"--timeout", "120s",
		"--contimeout", "30s",
	}

	if r.S3Endpoint != "" {
		args = append(args, "--s3-endpoint", r.S3Endpoint)
	}

	return r.runner.Run(ctx, "rclone", args...)
}
