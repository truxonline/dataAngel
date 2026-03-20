package sidecar

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"golang.org/x/sync/errgroup"
)

type Daemon struct {
	config      Config
	litestream  []*LitestreamRunner
	rclone      *RcloneRunner
	metricsPort int
}

func NewDaemon(config Config) *Daemon {
	var litestreamRunners []*LitestreamRunner
	for i, dbPath := range config.SqlitePaths {
		configPath := fmt.Sprintf("/tmp/litestream-%s-%d.yml", filepath.Base(dbPath), i)
		litestreamRunners = append(litestreamRunners, NewLitestreamRunner(configPath))
	}

	var rclonePaths []string
	rclonePaths = append(rclonePaths, config.FsPaths...)
	rclonePaths = append(rclonePaths, config.YAMLPaths...)

	return &Daemon{
		config:      config,
		litestream:  litestreamRunners,
		rclone:      NewRcloneRunner(config.RcloneInterval, rclonePaths, config.Bucket, config.S3Endpoint),
		metricsPort: config.MetricsPort,
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
	rcloneDelay := 30 * time.Second
	if len(d.litestream) == 0 {
		rcloneDelay = 0
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

	// Start metrics server (if enabled)
	if d.config.MetricsEnabled {
		eg.Go(func() error {
			log.Printf("Starting metrics server on port %d", d.metricsPort)
			metrics := GetMetrics()
			addr := fmt.Sprintf(":%d", d.metricsPort)
			if err := metrics.StartServer(addr); err != nil {
				return fmt.Errorf("failed to start metrics server: %w", err)
			}
			<-egCtx.Done()
			return nil
		})
	} else {
		log.Printf("Metrics server disabled (DATA_GUARD_METRICS_ENABLED=false)")
	}

	// Wait for all goroutines to complete
	err := eg.Wait()

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), d.config.ShutdownTimeout)
	defer cancel()

	// Wait for shutdown context to complete
	<-shutdownCtx.Done()

	return err
}

// RunSidecar starts the complete sidecar daemon (convenience wrapper for Daemon.Start)
func RunSidecar(ctx context.Context, config Config) error {
	daemon := NewDaemon(config)
	return daemon.Start(ctx)
}
