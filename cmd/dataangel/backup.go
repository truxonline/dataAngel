package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/charchess/dataAngel/internal/lock"
	"github.com/charchess/dataAngel/internal/sidecar"
)

func RunBackup(ctx context.Context, config Config, phaseManager *PhaseManager) error {
	lockCfg := lock.S3LockConfig{
		Bucket:   config.Bucket,
		Key:      fmt.Sprintf(".locks/%s", config.DeploymentName),
		Endpoint: config.S3Endpoint,
		TTL:      config.LockTTL,
	}

	s3Lock, err := lock.NewS3LockReal(ctx, lockCfg)
	if err != nil {
		return fmt.Errorf("failed to create lock: %w", err)
	}

	if err := s3Lock.Acquire(ctx, config.LockAcquireTimeout); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s3Lock.Release(releaseCtx); err != nil {
			log.Printf("[dataangel] Failed to release lock: %v", err)
		}
	}()

	phaseManager.SetLockAcquired(true)
	log.Println("[dataangel] Lock acquired, ready for traffic")

	renewCtx, cancelRenew := context.WithCancel(ctx)
	defer cancelRenew()
	go renewLockPeriodically(renewCtx, s3Lock, 30*time.Second, config.LockTTL)

	sidecarConfig := config.ToSidecarConfig()
	daemon := sidecar.NewDaemon(sidecarConfig)
	return daemon.Start(ctx)
}

func renewLockPeriodically(ctx context.Context, s3Lock *lock.S3LockReal, interval time.Duration, lockTTL time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	metrics := sidecar.GetMetrics()
	lastRenewed := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			renewCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			if err := s3Lock.Renew(renewCtx); err != nil {
				log.Printf("[dataangel] Failed to renew lock: %v", err)
				if metrics != nil {
					metrics.LockRenewalFailures.Inc()
				}
				// If renewal has been failing longer than TTL, assume lock
				// is lost and exit to prevent split-brain (issue #33).
				if time.Since(lastRenewed) > lockTTL {
					cancel()
					log.Printf("[dataangel] CRITICAL: lock renewal failed for longer than TTL (%v) — exiting to prevent split-brain", lockTTL)
					return
				}
			} else {
				lastRenewed = time.Now()
			}
			cancel()
		}
	}
}
