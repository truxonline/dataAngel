package sidecar

import (
	"context"
	"fmt"
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

	cmd.Cancel = func() error {
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	cmd.WaitDelay = 15 * time.Second

	return cmd.Run()
}

type LitestreamRunner struct {
	ConfigPath string
	runner     CommandRunner
}

func NewLitestreamRunner(configPath string) *LitestreamRunner {
	return &LitestreamRunner{
		ConfigPath: configPath,
		runner:     &realCommandRunner{},
	}
}

func (l *LitestreamRunner) Start(ctx context.Context) error {
	if l.ConfigPath == "" {
		return fmt.Errorf("litestream config path is required")
	}

	return l.runner.Run(ctx, "litestream", "replicate", "-config", l.ConfigPath)
}
