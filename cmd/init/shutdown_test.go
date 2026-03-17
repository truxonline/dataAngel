package main

import (
	"context"
	"testing"
	"time"
)

func TestShutdownSignalHandling(t *testing.T) {
	handler := NewShutdownHandler(1 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	flushFunc := func() error {
		return nil
	}

	err := handler.HandleShutdown(ctx, flushFunc)
	if err == nil {
		t.Errorf("Attendu une erreur de timeout, mais obtenu nil")
	}
}

func TestWALFlushLogic(t *testing.T) {
	err := FlushWAL()
	if err != nil {
		t.Errorf("FlushWAL ne devrait pas retourner d'erreur: %v", err)
	}
}

func TestS3SyncVerification(t *testing.T) {
	err := VerifyS3Sync()
	if err != nil {
		t.Errorf("VerifyS3Sync ne devrait pas retourner d'erreur: %v", err)
	}
}

func TestGracefulShutdownSequence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := GracefulShutdown(ctx, 1*time.Millisecond)

	if err == nil {
		t.Errorf("Attendu une erreur de timeout avec un délai très court")
	}
}
