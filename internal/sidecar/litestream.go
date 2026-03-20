package sidecar

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) error
}

type realCommandRunner struct{}

func (r *realCommandRunner) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Cancel = func() error {
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	cmd.WaitDelay = 15 * time.Second

	if err := cmd.Run(); err != nil {
		log.Printf("[%s] Command failed: %v", name, err)
		return err
	}

	return nil
}

type LitestreamRunner struct {
	ConfigPath string
	DBPath     string
	runner     CommandRunner
	metrics    *SidecarMetrics
}

func NewLitestreamRunner(configPath, dbPath string) *LitestreamRunner {
	return &LitestreamRunner{
		ConfigPath: configPath,
		DBPath:     dbPath,
		runner:     &realCommandRunner{},
		metrics:    GetMetrics(),
	}
}

// maxConsecutiveMissing is the number of consecutive DB-missing checks
// before the runner returns a fatal error (issue #22).
const maxConsecutiveMissing = 10

func (l *LitestreamRunner) Start(ctx context.Context) error {
	if l.ConfigPath == "" {
		return fmt.Errorf("litestream config path is required")
	}

	if l.metrics != nil {
		l.metrics.LitestreamUp.Set(1)
	}

	// Monitor DB file existence in background — if deleted at runtime,
	// litestream keeps running but logs errors indefinitely (issue #22).
	// Only trigger exit if the DB was previously seen (distinguishes
	// runtime deletion from first-deploy where app hasn't created it yet,
	// see issue #29).
	dbErrCh := make(chan error, 1)
	if l.DBPath != "" {
		go func() {
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()
			dbWasSeen := false
			consecutive := 0
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if _, err := os.Stat(l.DBPath); os.IsNotExist(err) {
						if !dbWasSeen {
							// First deploy: DB not created yet, keep waiting
							continue
						}
						consecutive++
						log.Printf("[dataangel] WARNING: database file missing: %s (%d/%d)", l.DBPath, consecutive, maxConsecutiveMissing)
						if consecutive >= maxConsecutiveMissing {
							dbErrCh <- fmt.Errorf("CRITICAL: database file %s has been missing for %d consecutive checks — exiting to trigger restore", l.DBPath, consecutive)
							return
						}
					} else {
						dbWasSeen = true
						consecutive = 0
					}
				}
			}
		}()
	}

	// Run litestream, but also watch for DB disappearance
	litestreamDone := make(chan error, 1)
	go func() {
		litestreamDone <- l.runner.Run(ctx, "litestream", "replicate", "-config", l.ConfigPath)
	}()

	var err error
	select {
	case err = <-litestreamDone:
	case err = <-dbErrCh:
	}

	if err != nil && err != context.Canceled && l.metrics != nil {
		l.metrics.LitestreamUp.Set(0)
	}

	return err
}
