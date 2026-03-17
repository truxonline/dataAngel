package sidecar

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestSetupSignalHandler(t *testing.T) {
	t.Run("should create cancelable context", func(t *testing.T) {
		// ACT
		ctx, cancel := SetupSignalHandler()
		defer cancel()

		// ASSERT
		if ctx == nil {
			t.Error("Expected non-nil context")
		}

		select {
		case <-ctx.Done():
			t.Error("Context should not be canceled yet")
		default:
			// OK
		}
	})

	t.Run("should cancel context on SIGTERM", func(t *testing.T) {
		// ARRANGE
		ctx, cancel := SetupSignalHandler()
		defer cancel()

		// ACT
		// Send SIGTERM to self
		proc, _ := os.FindProcess(os.Getpid())
		proc.Signal(syscall.SIGTERM)

		// Wait for context cancellation (with timeout)
		select {
		case <-ctx.Done():
			// OK — context was canceled
		case <-time.After(1 * time.Second):
			t.Error("Context should be canceled after SIGTERM")
		}
	})

	t.Run("should cancel context on SIGINT", func(t *testing.T) {
		// ARRANGE
		ctx, cancel := SetupSignalHandler()
		defer cancel()

		// ACT
		proc, _ := os.FindProcess(os.Getpid())
		proc.Signal(syscall.SIGINT)

		// Wait for context cancellation
		select {
		case <-ctx.Done():
			// OK
		case <-time.After(1 * time.Second):
			t.Error("Context should be canceled after SIGINT")
		}
	})
}

func TestWaitForShutdown(t *testing.T) {
	t.Run("should return when context is canceled", func(t *testing.T) {
		// ARRANGE
		ctx, cancel := context.WithCancel(context.Background())

		// ACT
		done := make(chan struct{})
		go func() {
			WaitForShutdown(ctx)
			close(done)
		}()

		cancel()

		// ASSERT
		select {
		case <-done:
			// OK
		case <-time.After(100 * time.Millisecond):
			t.Error("WaitForShutdown should return quickly after context cancel")
		}
	})

	t.Run("should block until context is canceled", func(t *testing.T) {
		// ARRANGE
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// ACT
		done := make(chan struct{})
		go func() {
			WaitForShutdown(ctx)
			close(done)
		}()

		// Wait a bit
		time.Sleep(50 * time.Millisecond)

		// ASSERT — should still be blocked
		select {
		case <-done:
			t.Error("WaitForShutdown should not return before context cancel")
		default:
			// OK — still waiting
		}

		// Cleanup
		cancel()
		<-done
	})
}
