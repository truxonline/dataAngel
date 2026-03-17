package sidecar

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDaemon(t *testing.T) {
	t.Run("should create daemon with valid config", func(t *testing.T) {
		// ARRANGE
		config := Config{
			Bucket:          "test-bucket",
			S3Endpoint:      "http://minio:9000",
			SqlitePaths:     []string{"/data/app.db"},
			FsPaths:         []string{"/config"},
			YAMLPaths:       []string{"/config/*.yaml"},
			RcloneInterval:  60 * time.Second,
			MetricsPort:     9090,
			ShutdownTimeout: 15 * time.Second,
		}

		// ACT
		daemon := NewDaemon(config)

		// ASSERT
		if daemon == nil {
			t.Error("Expected non-nil daemon")
		}
		if daemon.config.Bucket != "test-bucket" {
			t.Errorf("Expected bucket 'test-bucket', got '%s'", daemon.config.Bucket)
		}
	})

	t.Run("should start daemon with errgroup", func(t *testing.T) {
		// ARRANGE
		config := Config{
			Bucket:          "test-bucket",
			S3Endpoint:      "http://minio:9000",
			SqlitePaths:     []string{"/data/app.db"},
			FsPaths:         []string{"/config"},
			YAMLPaths:       []string{"/config/*.yaml"},
			RcloneInterval:  60 * time.Second,
			MetricsPort:     9090,
			ShutdownTimeout: 15 * time.Second,
		}
		daemon := NewDaemon(config)

		// Mock the command runners
		mockRunner := &MockCommandRunner{}
		for _, ls := range daemon.litestream {
			ls.runner = mockRunner
		}
		daemon.rclone.runner = mockRunner

		// ACT
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		err := daemon.Start(ctx)

		// ASSERT
		// Should complete without panic
		if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context deadline/canceled or nil, got %v", err)
		}
	})

	t.Run("should handle graceful shutdown", func(t *testing.T) {
		// ARRANGE
		config := Config{
			Bucket:          "test-bucket",
			S3Endpoint:      "http://minio:9000",
			SqlitePaths:     []string{"/data/app.db"},
			FsPaths:         []string{"/config"},
			YAMLPaths:       []string{"/config/*.yaml"},
			RcloneInterval:  60 * time.Second,
			MetricsPort:     9090,
			ShutdownTimeout: 15 * time.Second,
		}
		daemon := NewDaemon(config)

		// Mock the command runners
		mockRunner := &MockCommandRunner{}
		for _, ls := range daemon.litestream {
			ls.runner = mockRunner
		}
		daemon.rclone.runner = mockRunner

		// ACT
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()
		err := daemon.Start(ctx)

		// ASSERT
		// Should complete gracefully
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context canceled or nil, got %v", err)
		}
	})

	t.Run("should respect shutdown timeout", func(t *testing.T) {
		// ARRANGE
		config := Config{
			Bucket:          "test-bucket",
			S3Endpoint:      "http://minio:9000",
			SqlitePaths:     []string{"/data/app.db"},
			FsPaths:         []string{"/config"},
			YAMLPaths:       []string{"/config/*.yaml"},
			RcloneInterval:  60 * time.Second,
			MetricsPort:     9090,
			ShutdownTimeout: 100 * time.Millisecond,
		}
		daemon := NewDaemon(config)

		// Mock the command runners
		mockRunner := &MockCommandRunner{}
		for _, ls := range daemon.litestream {
			ls.runner = mockRunner
		}
		daemon.rclone.runner = mockRunner

		// ACT
		start := time.Now()
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()
		daemon.Start(ctx)
		elapsed := time.Since(start)

		// ASSERT
		// Should complete within reasonable time (not hang forever)
		if elapsed > 5*time.Second {
			t.Errorf("Expected shutdown within 5s, took %v", elapsed)
		}
	})

	t.Run("should initialize litestream replicator", func(t *testing.T) {
		// ARRANGE
		config := Config{
			Bucket:          "test-bucket",
			S3Endpoint:      "http://minio:9000",
			SqlitePaths:     []string{"/data/app.db"},
			FsPaths:         []string{"/config"},
			YAMLPaths:       []string{"/config/*.yaml"},
			RcloneInterval:  60 * time.Second,
			MetricsPort:     9090,
			ShutdownTimeout: 15 * time.Second,
		}
		daemon := NewDaemon(config)

		// ACT
		if daemon.litestream == nil {
			t.Error("Expected litestream replicator to be initialized")
		}

		// ASSERT
		if len(daemon.litestream) == 0 {
			t.Error("Expected at least one litestream replicator")
		}
	})
}
