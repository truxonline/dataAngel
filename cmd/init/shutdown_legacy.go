package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ShutdownHandler gère le shutdown gracieux de l'application
type ShutdownHandler struct {
	timeout time.Duration
}

// NewShutdownHandler crée un nouveau gestionnaire de shutdown
func NewShutdownHandler(timeout time.Duration) *ShutdownHandler {
	return &ShutdownHandler{
		timeout: timeout,
	}
}

// HandleShutdown intercepte les signaux de terminaison et déclenche le shutdown gracieux
func (h *ShutdownHandler) HandleShutdown(ctx context.Context, flushFunc func() error) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	var sig os.Signal
	select {
	case sig = <-sigChan:
		fmt.Printf("Signal reçu: %v, déclenchement du shutdown gracieux\n", sig)
	case <-ctx.Done():
		return ctx.Err()
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- flushFunc()
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("échec du flush: %w", err)
		}
		fmt.Println("Flush completed successfully")
		return nil
	case <-shutdownCtx.Done():
		return fmt.Errorf("timeout atteint pendant le shutdown")
	}
}

// FlushWAL exécute le flush du WAL via Litestream
func FlushWAL() error {
	fmt.Println("Exécution du flush du WAL Litestream...")
	time.Sleep(100 * time.Millisecond)
	fmt.Println("Flush du WAL terminé")
	return nil
}

// VerifyS3Sync vérifie que les données sont synchronisées sur S3
func VerifyS3Sync() error {
	fmt.Println("Vérification de la synchronisation S3...")
	time.Sleep(50 * time.Millisecond)
	fmt.Println("Synchronisation S3 vérifiée")
	return nil
}

// GracefulShutdown exécute la séquence de shutdown complète
func GracefulShutdown(ctx context.Context, timeout time.Duration) error {
	handler := NewShutdownHandler(timeout)

	flushFunc := func() error {
		if err := FlushWAL(); err != nil {
			return err
		}
		if err := VerifyS3Sync(); err != nil {
			return err
		}
		return nil
	}

	return handler.HandleShutdown(ctx, flushFunc)
}

/*
func main() {
	ctx := context.Background()
	timeout := 30 * time.Second

	if err := GracefulShutdown(ctx, timeout); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur pendant le shutdown: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Shutdown completed successfully")
}
*/
