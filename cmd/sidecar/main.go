package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/charchess/dataAngel/internal/sidecar"
)

func main() {
	// Load configuration from environment
	config, err := sidecar.LoadFromEnv()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create daemon
	daemon := sidecar.NewDaemon(config)

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		cancel()
	}()

	// Start daemon
	log.Println("Starting sidecar daemon...")
	if err := daemon.Start(ctx); err != nil {
		log.Printf("Daemon error: %v", err)
	}

	log.Println("Sidecar daemon stopped")
}
