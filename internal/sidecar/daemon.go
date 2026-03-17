package sidecar

import (
	"context"
	"fmt"
	"log"

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
	for _, dbPath := range config.SqlitePaths {
		configPath := fmt.Sprintf("/etc/litestream/%s.yml", dbPath)
		litestreamRunners = append(litestreamRunners, NewLitestreamRunner(configPath))
	}

	var rclonePaths []string
	rclonePaths = append(rclonePaths, config.FsPaths...)
	rclonePaths = append(rclonePaths, config.YAMLPaths...)

	return &Daemon{
		config:      config,
		litestream:  litestreamRunners,
		rclone:      NewRcloneRunner(config.RcloneInterval, rclonePaths, config.Bucket),
		metricsPort: config.MetricsPort,
	}
}

// Start begins the daemon with all goroutines managed by errgroup
func (d *Daemon) Start(ctx context.Context) error {
	eg, egCtx := errgroup.WithContext(ctx)

	for _, replicator := range d.litestream {
		r := replicator
		eg.Go(func() error {
			log.Printf("Starting litestream replicator for %s", r.ConfigPath)
			return r.Start(egCtx)
		})
	}

	// Start rclone sync loop
	eg.Go(func() error {
		log.Printf("Starting rclone sync loop with interval %v", d.config.RcloneInterval)
		return d.rclone.Start(egCtx)
	})

	// Start metrics server
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
