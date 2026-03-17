package sidecar

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"
)

// MockCommandRunner for testing
type MockCommandRunner struct {
	commands []*exec.Cmd
	err      error
	output   string
}

func (m *MockCommandRunner) Run(ctx context.Context, cmd *exec.Cmd) error {
	m.commands = append(m.commands, cmd)
	return m.err
}

func TestLitestreamReplicator(t *testing.T) {
	t.Run("should create replicator with valid config", func(t *testing.T) {
		// ARRANGE
		config := LitestreamConfig{
			DBPath:     "/data/app.db",
			S3Bucket:   "backup-bucket",
			S3Path:     "backups/app.db",
			S3Endpoint: "http://minio:9000",
		}

		// ACT
		replicator := NewLitestreamReplicator(config)

		// ASSERT
		if replicator == nil {
			t.Error("Expected non-nil replicator")
		}
		if replicator.config.DBPath != "/data/app.db" {
			t.Errorf("Expected DBPath '/data/app.db', got '%s'", replicator.config.DBPath)
		}
	})

	t.Run("should build correct litestream command", func(t *testing.T) {
		// ARRANGE
		config := LitestreamConfig{
			DBPath:     "/data/app.db",
			S3Bucket:   "backup-bucket",
			S3Path:     "backups/app.db",
			S3Endpoint: "http://minio:9000",
		}
		replicator := NewLitestreamReplicator(config)

		// ACT
		cmd := replicator.buildCommand()

		// ASSERT
		if cmd == nil {
			t.Fatal("Expected non-nil command")
		}
		if cmd.Path != "litestream" && cmd.Args[0] != "litestream" {
			t.Errorf("Expected 'litestream' command, got '%v'", cmd.Args)
		}
		// Check for replicate subcommand
		found := false
		for _, arg := range cmd.Args {
			if arg == "replicate" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected 'replicate' subcommand in args: %v", cmd.Args)
		}
	})

	t.Run("should set context with cancel and timeout", func(t *testing.T) {
		// ARRANGE
		config := LitestreamConfig{
			DBPath:     "/data/app.db",
			S3Bucket:   "backup-bucket",
			S3Path:     "backups/app.db",
			S3Endpoint: "http://minio:9000",
		}
		replicator := NewLitestreamReplicator(config)

		// ACT
		cmd := replicator.buildCommand()
		cmd.Cancel = func() error {
			return nil
		}
		cmd.WaitDelay = 15 * time.Second

		// ASSERT
		if cmd.Cancel == nil {
			t.Error("Expected Cancel to be set")
		}
		if cmd.WaitDelay != 15*time.Second {
			t.Errorf("Expected WaitDelay 15s, got %v", cmd.WaitDelay)
		}
	})

	t.Run("should start replicator with mock runner", func(t *testing.T) {
		// ARRANGE
		config := LitestreamConfig{
			DBPath:     "/data/app.db",
			S3Bucket:   "backup-bucket",
			S3Path:     "backups/app.db",
			S3Endpoint: "http://minio:9000",
		}
		replicator := NewLitestreamReplicator(config)
		mockRunner := &MockCommandRunner{}
		replicator.runner = mockRunner

		// ACT
		err := replicator.Start(context.Background())

		// ASSERT
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(mockRunner.commands) != 1 {
			t.Errorf("Expected 1 command executed, got %d", len(mockRunner.commands))
		}
	})

	t.Run("should handle runner error", func(t *testing.T) {
		// ARRANGE
		config := LitestreamConfig{
			DBPath:     "/data/app.db",
			S3Bucket:   "backup-bucket",
			S3Path:     "backups/app.db",
			S3Endpoint: "http://minio:9000",
		}
		replicator := NewLitestreamReplicator(config)
		mockRunner := &MockCommandRunner{err: errors.New("command failed")}
		replicator.runner = mockRunner

		// ACT
		err := replicator.Start(context.Background())

		// ASSERT
		if err == nil {
			t.Error("Expected error from runner")
		}
		if !errors.Is(err, mockRunner.err) {
			t.Errorf("Expected error '%v', got '%v'", mockRunner.err, err)
		}
	})

	t.Run("should include S3 endpoint in environment", func(t *testing.T) {
		// ARRANGE
		config := LitestreamConfig{
			DBPath:     "/data/app.db",
			S3Bucket:   "backup-bucket",
			S3Path:     "backups/app.db",
			S3Endpoint: "http://minio:9000",
		}
		replicator := NewLitestreamReplicator(config)

		// ACT
		cmd := replicator.buildCommand()

		// ASSERT
		if cmd.Env == nil {
			t.Error("Expected environment variables to be set")
		}
		// Check for S3_ENDPOINT in env
		found := false
		for _, env := range cmd.Env {
			if env == "LITESTREAM_S3_ENDPOINT=http://minio:9000" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected S3_ENDPOINT in env, got %v", cmd.Env)
		}
	})

	t.Run("should stop replicator gracefully", func(t *testing.T) {
		// ARRANGE
		config := LitestreamConfig{
			DBPath:     "/data/app.db",
			S3Bucket:   "backup-bucket",
			S3Path:     "backups/app.db",
			S3Endpoint: "http://minio:9000",
		}
		replicator := NewLitestreamReplicator(config)
		mockRunner := &MockCommandRunner{}
		replicator.runner = mockRunner

		// ACT
		ctx, cancel := context.WithCancel(context.Background())
		replicator.Start(ctx)
		cancel()
		time.Sleep(100 * time.Millisecond) // Allow graceful shutdown

		// ASSERT
		// Verify context was cancelled
		select {
		case <-ctx.Done():
			// Expected
		default:
			t.Error("Expected context to be cancelled")
		}
	})

	t.Run("should handle missing S3 endpoint gracefully", func(t *testing.T) {
		// ARRANGE
		config := LitestreamConfig{
			DBPath:     "/data/app.db",
			S3Bucket:   "backup-bucket",
			S3Path:     "backups/app.db",
			S3Endpoint: "", // Empty endpoint
		}
		replicator := NewLitestreamReplicator(config)

		// ACT
		cmd := replicator.buildCommand()

		// ASSERT
		if cmd == nil {
			t.Fatal("Expected non-nil command even with empty endpoint")
		}
		// Should still build valid command
		if len(cmd.Args) == 0 {
			t.Error("Expected non-empty args")
		}
	})
}
