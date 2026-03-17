package sidecar

import (
	"context"
	"errors"
	"testing"
)

// Mock CommandRunner for testing
type mockCommandRunner struct {
	err     error
	started bool
}

func (m *mockCommandRunner) Run(ctx context.Context, name string, args ...string) error {
	m.started = true
	if m.err != nil {
		return m.err
	}
	<-ctx.Done()
	return ctx.Err()
}

func TestLitestreamRunner(t *testing.T) {
	t.Run("should start litestream with config", func(t *testing.T) {
		mock := &mockCommandRunner{}
		runner := &LitestreamRunner{
			ConfigPath: "/tmp/litestream.yml",
			runner:     mock,
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Immediate cancel

		err := runner.Start(ctx)
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
		if !mock.started {
			t.Error("Expected litestream to start")
		}
	})

	t.Run("should return error on missing config", func(t *testing.T) {
		runner := &LitestreamRunner{ConfigPath: ""}
		err := runner.Start(context.Background())
		if err == nil {
			t.Error("Expected error for missing config")
		}
	})

	t.Run("should propagate subprocess errors", func(t *testing.T) {
		mock := &mockCommandRunner{err: errors.New("litestream failed")}
		runner := &LitestreamRunner{
			ConfigPath: "/tmp/litestream.yml",
			runner:     mock,
		}

		err := runner.Start(context.Background())
		if err == nil {
			t.Error("Expected error from subprocess")
		}
	})

	t.Run("should stop on context cancel", func(t *testing.T) {
		mock := &mockCommandRunner{}
		runner := &LitestreamRunner{
			ConfigPath: "/tmp/litestream.yml",
			runner:     mock,
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := runner.Start(ctx)
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	})
}
