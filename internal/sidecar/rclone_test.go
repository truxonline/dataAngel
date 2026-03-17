package sidecar

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRcloneSyncer(t *testing.T) {
	t.Run("should create syncer with valid config", func(t *testing.T) {
		// ARRANGE
		config := RcloneConfig{
			Paths:    []string{"/config", "/data"},
			S3Bucket: "backup-bucket",
			Interval: 60 * time.Second,
		}

		// ACT
		syncer := NewRcloneSyncer(config)

		// ASSERT
		if syncer == nil {
			t.Error("Expected non-nil syncer")
		}
		if syncer.config.Interval != 60*time.Second {
			t.Errorf("Expected interval 60s, got %v", syncer.config.Interval)
		}
	})

	t.Run("should validate YAML paths before sync", func(t *testing.T) {
		// ARRANGE
		config := RcloneConfig{
			Paths:    []string{"/config/*.yaml", "/data/*.yml"},
			S3Bucket: "backup-bucket",
			Interval: 60 * time.Second,
		}
		syncer := NewRcloneSyncer(config)

		// ACT
		valid := syncer.validateYAMLPaths()

		// ASSERT
		if !valid {
			t.Error("Expected YAML paths to be valid")
		}
	})

	t.Run("should reject invalid YAML patterns", func(t *testing.T) {
		// ARRANGE
		config := RcloneConfig{
			Paths:    []string{"/config/*.txt", "/data/*.json"},
			S3Bucket: "backup-bucket",
			Interval: 60 * time.Second,
		}
		syncer := NewRcloneSyncer(config)

		// ACT
		valid := syncer.validateYAMLPaths()

		// ASSERT
		if valid {
			t.Error("Expected YAML paths to be invalid")
		}
	})

	t.Run("should build correct rclone sync command", func(t *testing.T) {
		// ARRANGE
		config := RcloneConfig{
			Paths:    []string{"/config"},
			S3Bucket: "backup-bucket",
			Interval: 60 * time.Second,
		}
		syncer := NewRcloneSyncer(config)

		// ACT
		cmd := syncer.buildCommand("/config")

		// ASSERT
		if cmd == nil {
			t.Fatal("Expected non-nil command")
		}
		if cmd.Path != "rclone" && cmd.Args[0] != "rclone" {
			t.Errorf("Expected 'rclone' command, got '%v'", cmd.Args)
		}
		// Check for sync subcommand
		found := false
		for _, arg := range cmd.Args {
			if arg == "sync" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected 'sync' subcommand in args: %v", cmd.Args)
		}
	})

	t.Run("should start sync loop with ticker", func(t *testing.T) {
		// ARRANGE
		config := RcloneConfig{
			Paths:    []string{"/config"},
			S3Bucket: "backup-bucket",
			Interval: 100 * time.Millisecond, // Short interval for testing
		}
		syncer := NewRcloneSyncer(config)
		mockRunner := &MockCommandRunner{}
		syncer.runner = mockRunner

		// ACT
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()
		syncer.StartLoop(ctx)

		// ASSERT
		// Should have executed at least once (initial + one tick)
		if len(mockRunner.commands) < 1 {
			t.Errorf("Expected at least 1 command executed, got %d", len(mockRunner.commands))
		}
	})

	t.Run("should handle runner error gracefully", func(t *testing.T) {
		// ARRANGE
		config := RcloneConfig{
			Paths:    []string{"/config"},
			S3Bucket: "backup-bucket",
			Interval: 100 * time.Millisecond,
		}
		syncer := NewRcloneSyncer(config)
		mockRunner := &MockCommandRunner{err: errors.New("sync failed")}
		syncer.runner = mockRunner

		// ACT
		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()
		syncer.StartLoop(ctx)

		// ASSERT
		// Should continue despite error
		if len(mockRunner.commands) == 0 {
			t.Error("Expected commands to be attempted despite error")
		}
	})

	t.Run("should respect context cancellation", func(t *testing.T) {
		// ARRANGE
		config := RcloneConfig{
			Paths:    []string{"/config"},
			S3Bucket: "backup-bucket",
			Interval: 100 * time.Millisecond,
		}
		syncer := NewRcloneSyncer(config)
		mockRunner := &MockCommandRunner{}
		syncer.runner = mockRunner

		// ACT
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(150 * time.Millisecond)
			cancel()
		}()
		syncer.StartLoop(ctx)

		// ASSERT
		select {
		case <-ctx.Done():
			// Expected
		default:
			t.Error("Expected context to be cancelled")
		}
	})
}
