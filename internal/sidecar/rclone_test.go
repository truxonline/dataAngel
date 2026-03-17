package sidecar

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Mock for rclone runner
type mockRcloneRunner struct {
	syncCount int
	err       error
}

func (m *mockRcloneRunner) Run(ctx context.Context, name string, args ...string) error {
	m.syncCount++
	return m.err
}

func TestRcloneRunner(t *testing.T) {
	t.Run("should sync on interval", func(t *testing.T) {
		mock := &mockRcloneRunner{}
		runner := &RcloneRunner{
			Interval:   10 * time.Millisecond,
			FsPaths:    []string{"/data"},
			Bucket:     "test-bucket",
			S3Endpoint: "",
			runner:     mock,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		runner.Start(ctx)

		if mock.syncCount < 2 {
			t.Errorf("Expected at least 2 syncs, got %d", mock.syncCount)
		}
	})

	t.Run("should stop on context cancel", func(t *testing.T) {
		mock := &mockRcloneRunner{}
		runner := &RcloneRunner{
			Interval:   1 * time.Second,
			FsPaths:    []string{"/data"},
			Bucket:     "test-bucket",
			S3Endpoint: "",
			runner:     mock,
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := runner.Start(ctx)
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	})

	t.Run("should continue after sync error", func(t *testing.T) {
		mock := &mockRcloneRunner{err: errors.New("sync failed")}
		runner := &RcloneRunner{
			Interval:   10 * time.Millisecond,
			FsPaths:    []string{"/data"},
			Bucket:     "test-bucket",
			S3Endpoint: "",
			runner:     mock,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		runner.Start(ctx)

		if mock.syncCount < 2 {
			t.Error("Expected rclone to retry after error")
		}
	})

	t.Run("should skip sync when no paths configured", func(t *testing.T) {
		mock := &mockRcloneRunner{}
		runner := &RcloneRunner{
			Interval:   10 * time.Millisecond,
			FsPaths:    []string{},
			Bucket:     "test-bucket",
			S3Endpoint: "",
			runner:     mock,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		runner.Start(ctx)

		if mock.syncCount > 0 {
			t.Error("Expected no syncs when FsPaths is empty")
		}
	})
}
