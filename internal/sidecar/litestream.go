package sidecar

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// CommandRunner interface for executing commands (allows mocking)
type CommandRunner interface {
	Run(ctx context.Context, cmd *exec.Cmd) error
}

// DefaultCommandRunner implements CommandRunner using os/exec
type DefaultCommandRunner struct{}

func (d *DefaultCommandRunner) Run(ctx context.Context, cmd *exec.Cmd) error {
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return cmd.Process.Signal(os.Interrupt)
		}
		return nil
	}
	cmd.WaitDelay = 15 * time.Second
	return cmd.Run()
}

// LitestreamConfig holds configuration for litestream replication
type LitestreamConfig struct {
	DBPath     string
	S3Bucket   string
	S3Path     string
	S3Endpoint string
}

// LitestreamReplicator manages litestream replication process
type LitestreamReplicator struct {
	config LitestreamConfig
	runner CommandRunner
}

// NewLitestreamReplicator creates a new litestream replicator
func NewLitestreamReplicator(config LitestreamConfig) *LitestreamReplicator {
	return &LitestreamReplicator{
		config: config,
		runner: &DefaultCommandRunner{},
	}
}

// buildCommand constructs the litestream replicate command
func (lr *LitestreamReplicator) buildCommand() *exec.Cmd {
	cmd := exec.Command("litestream", "replicate", lr.config.DBPath)

	// Set environment variables for S3 configuration
	env := os.Environ()
	env = append(env, fmt.Sprintf("LITESTREAM_S3_BUCKET=%s", lr.config.S3Bucket))
	env = append(env, fmt.Sprintf("LITESTREAM_S3_PATH=%s", lr.config.S3Path))

	if lr.config.S3Endpoint != "" {
		env = append(env, fmt.Sprintf("LITESTREAM_S3_ENDPOINT=%s", lr.config.S3Endpoint))
	}

	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd
}

// Start begins the litestream replication process
func (lr *LitestreamReplicator) Start(ctx context.Context) error {
	cmd := lr.buildCommand()
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return cmd.Process.Signal(os.Interrupt)
		}
		return nil
	}
	cmd.WaitDelay = 15 * time.Second

	return lr.runner.Run(ctx, cmd)
}
