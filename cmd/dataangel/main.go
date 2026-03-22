package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Load configuration from environment
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("[dataangel] Failed to load configuration: %v", err)
	}

	// Setup signal handling for graceful shutdown during backup phase
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("[dataangel] Received signal: %v", sig)
		// Write clean shutdown sentinels so the next startup can skip
		// expensive DB validation (issue #42).
		writeCleanShutdownSentinels(config.SqlitePaths)
		cancel()
	}()

	// Initialize phase manager (manages readiness probe + phase metrics)
	phaseManager := NewPhaseManager(config.MetricsPort, config.MetricsEnabled)

	// Start readiness probe server in background (returns 503 during restore, 200 after)
	if config.MetricsEnabled {
		go func() {
			if err := phaseManager.StartReadinessServer(); err != nil {
				log.Printf("[dataangel] Failed to start readiness server: %v", err)
			}
		}()
	}

	// PHASE 1: RESTORE (blocking, exits on failure)
	log.Println("[dataangel] phase=restore starting")
	phaseManager.SetPhase(PhaseRestore)

	restoreStart := time.Now()
	if err := RunRestore(ctx, config); err != nil {
		log.Fatalf("[dataangel] phase=restore failed: %v", err)
	}
	restoreDuration := time.Since(restoreStart)

	log.Printf("[dataangel] phase=restore complete elapsed=%v", restoreDuration)
	phaseManager.RecordRestoreDuration(restoreDuration)

	// PHASE 2: BACKUP (continuous daemon)
	log.Println("[dataangel] phase=backup starting")
	phaseManager.SetPhase(PhaseBackup)

	if err := RunBackup(ctx, config, phaseManager); err != nil {
		log.Printf("[dataangel] phase=backup error: %v", err)
	}

	log.Println("[dataangel] Daemon stopped")
}
