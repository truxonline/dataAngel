package main

import (
	"context"
	"fmt"
	"os"

	"github.com/charchess/dataAngel/internal/restore"
)

func main() {
	// Load configuration from environment
	bucket := os.Getenv("DATA_GUARD_BUCKET")
	if bucket == "" {
		fmt.Fprintln(os.Stderr, "Error: DATA_GUARD_BUCKET environment variable is required")
		os.Exit(2)
	}

	remotePath := os.Getenv("DATA_GUARD_PATH")
	if remotePath == "" {
		fmt.Fprintln(os.Stderr, "Error: DATA_GUARD_PATH environment variable is required")
		os.Exit(2)
	}

	localPath := os.Getenv("DATA_GUARD_LOCAL_PATH")
	if localPath == "" {
		fmt.Fprintln(os.Stderr, "Error: DATA_GUARD_LOCAL_PATH environment variable is required")
		os.Exit(2)
	}

	expectedChecksum := os.Getenv("DATA_GUARD_CHECKSUM")
	if expectedChecksum == "" {
		fmt.Fprintln(os.Stderr, "Error: DATA_GUARD_CHECKSUM environment variable is required")
		os.Exit(2)
	}

	// Create restore configuration
	config := restore.RestoreConfig{
		Bucket:           bucket,
		Key:              remotePath,
		DestPath:         localPath,
		ExpectedChecksum: expectedChecksum,
	}

	// Run the restore pipeline
	ctx := context.Background()
	if err := RunRestore(ctx, config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}
