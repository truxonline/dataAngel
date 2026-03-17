package sidecar

import (
	"context"
	"fmt"
	"log"

	"golang.org/x/sync/errgroup"
)

// Daemon manages the sidecar daemon lifecycle
type Daemon struct {
	config      Config
	litestream  []*LitestreamReplicator
	rclone      *RcloneSyncer
	metricsPort int
}

// NewDaemon creates a new sidecar daemon
func NewDaemon(config Config) *Daemon {
	// Initialize litestream replicators for each SQLite path
	var litestreamReplicators []*LitestreamReplicator
	for _, dbPath := range config.SqlitePaths {
		lsConfig := LitestreamConfig{
			DBPath:     dbPath,
			S3Bucket:   config.Bucket,
			S3Path:     fmt.Sprintf("backups/%s", dbPath),
			S3Endpoint: config.S3Endpoint,
		}
		litestreamReplicators = append(litestreamReplicators, NewLitestreamReplicator(lsConfig))
	}

	// Initialize rclone syncer for file paths
	var rclonePaths []string
	rclonePaths = append(rclonePaths, config.FsPaths...)
	rclonePaths = append(rclonePaths, config.YAMLPaths...)

	rcloneConfig := RcloneConfig{
		Paths:    rclonePaths,
		S3Bucket: config.Bucket,
		Interval: config.RcloneInterval,
	}

	return &Daemon{
		config:      config,
		litestream:  litestreamReplicators,
		rclone:      NewRcloneSyncer(rcloneConfig),
		metricsPort: config.MetricsPort,
	}
}

// Start begins the daemon with all goroutines managed by errgroup
func (d *Daemon) Start(ctx context.Context) error {
	eg, egCtx := errgroup.WithContext(ctx)

	// Start litestream replicators
	for _, replicator := range d.litestream {
		r := replicator // Capture for closure
		eg.Go(func() error {
			log.Printf("Starting litestream replicator for %s", r.config.DBPath)
			return r.Start(egCtx)
		})
	}

	// Start rclone sync loop
	eg.Go(func() error {
		log.Printf("Starting rclone sync loop with interval %v", d.config.RcloneInterval)
		d.rclone.StartLoop(egCtx)
		return nil
	})

	// Start metrics server
	eg.Go(func() error {
		log.Printf("Starting metrics server on port %d", d.metricsPort)
		// Placeholder for metrics server
		<-egCtx.Done()
		return nil
	})

	// Wait for all goroutines to complete
	err := eg.Wait()

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), d.config.ShutdownTimeout)
	defer cancel()

	// Wait for shutdown context to complete
	<-shutdownCtx.Done()

	return err
}
