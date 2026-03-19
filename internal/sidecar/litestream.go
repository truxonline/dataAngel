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
	runner     CommandRunner
	metrics    *SidecarMetrics
}

func NewLitestreamRunner(configPath string) *LitestreamRunner {
	return &LitestreamRunner{
		ConfigPath: configPath,
		runner:     &realCommandRunner{},
		metrics:    GetMetrics(),
	}
}

func (l *LitestreamRunner) Start(ctx context.Context) error {
	if l.ConfigPath == "" {
		return fmt.Errorf("litestream config path is required")
	}

	if l.metrics != nil {
		l.metrics.LitestreamUp.Set(1)
	}

	err := l.runner.Run(ctx, "litestream", "replicate", "-config", l.ConfigPath)

	if err != nil && err != context.Canceled && l.metrics != nil {
		l.metrics.LitestreamUp.Set(0)
	}

	return err
}
