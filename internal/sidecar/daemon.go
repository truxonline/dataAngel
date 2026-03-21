package sidecar

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
	"time"

	"golang.org/x/sync/errgroup"
)

type Daemon struct {
	config     Config
	litestream []*LitestreamRunner
	rclone     *RcloneRunner
}

func NewDaemon(config Config) *Daemon {
	var litestreamRunners []*LitestreamRunner
	for i, dbPath := range config.SqlitePaths {
		configPath := fmt.Sprintf("/tmp/litestream-%s-%d.yml", filepath.Base(dbPath), i)
		litestreamRunners = append(litestreamRunners, NewLitestreamRunner(configPath, dbPath))
	}

	var rclonePaths []string
	rclonePaths = append(rclonePaths, config.FsPaths...)
	rclonePaths = append(rclonePaths, config.YAMLPaths...)

	return &Daemon{
		config:     config,
		litestream: litestreamRunners,
		rclone: NewRcloneRunner(RcloneRunnerConfig{
			Interval:        config.RcloneInterval,
			SyncTimeout:     config.SyncTimeout,
			FsPaths:         rclonePaths,
			Bucket:          config.Bucket,
			S3Endpoint:      config.S3Endpoint,
			ExcludePatterns: config.ExcludePatterns,
			Transfers:       config.RcloneTransfers,
			Checkers:        config.RcloneCheckers,
			Bwlimit:         config.RcloneBwlimit,
		}),
	}
}

func (d *Daemon) initializeConfigs() error {
	for i, dbPath := range d.config.SqlitePaths {
		configPath := d.litestream[i].ConfigPath
		if err := GenerateLitestreamConfig(dbPath, d.config.Bucket, d.config.S3Endpoint, configPath); err != nil {
			return fmt.Errorf("failed to generate litestream config for %s: %w", dbPath, err)
		}
		log.Printf("Generated litestream config: %s", configPath)
	}
	return nil
}

// Start begins the daemon with all goroutines managed by errgroup.
// Litestream replicators start first; rclone is delayed to avoid saturating
// S3/MinIO connections during the initial litestream snapshot, which would
// starve lock renewals (see issue #17).
func (d *Daemon) Start(ctx context.Context) error {
	if err := d.initializeConfigs(); err != nil {
		return fmt.Errorf("failed to initialize configs: %w", err)
	}

	eg, egCtx := errgroup.WithContext(ctx)

	// Phase 1: Start litestream replicators immediately
	for _, replicator := range d.litestream {
		r := replicator
		eg.Go(func() error {
			log.Printf("Starting litestream replicator for %s", r.ConfigPath)
			return r.Start(egCtx)
		})
	}

	// Phase 2: Start rclone after a delay to let litestream complete its
	// initial snapshot without competing for S3 bandwidth
	rcloneDelay := d.config.RcloneDelay
	if len(d.litestream) == 0 {
		rcloneDelay = 0
	}
	// Add jitter to prevent thundering herd on cluster-wide restarts (#34)
	if rcloneDelay > 0 {
		jitter := time.Duration(rand.Int63n(int64(rcloneDelay)))
		rcloneDelay += jitter
	}
	eg.Go(func() error {
		if rcloneDelay > 0 {
			log.Printf("Delaying rclone start by %v to let litestream initialize", rcloneDelay)
			select {
			case <-egCtx.Done():
				return egCtx.Err()
			case <-time.After(rcloneDelay):
			}
		}
		log.Printf("Starting rclone sync loop with interval %v", d.config.RcloneInterval)
		return d.rclone.Start(egCtx)
	})

	// Sidecar metrics are registered in the default Prometheus registry
	// and served by the readiness server in phase.go (issue #21).

	// Wait for all goroutines to complete (subprocess WaitDelay handles
	// graceful termination). Return immediately so callers can run cleanup
	// (e.g. lock release) before Kubernetes sends SIGKILL (issue #23).
	return eg.Wait()
}

// RunSidecar starts the complete sidecar daemon (convenience wrapper for Daemon.Start)
func RunSidecar(ctx context.Context, config Config) error {
	daemon := NewDaemon(config)
	return daemon.Start(ctx)
}
