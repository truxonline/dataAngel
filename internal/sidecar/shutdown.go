package sidecar

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// SetupSignalHandler creates a context that is canceled on SIGTERM or SIGINT.
// Returns the context and a cancel function for cleanup.
func SetupSignalHandler() (context.Context, context.CancelFunc) {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
	)

	return ctx, cancel
}

// WaitForShutdown blocks until the provided context is canceled.
// Use this in the main daemon loop to wait for shutdown signal.
func WaitForShutdown(ctx context.Context) {
	<-ctx.Done()
}
