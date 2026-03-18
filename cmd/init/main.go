package main

import (
	"context"
	"fmt"
	"os"
	"strings"
)

func main() {
	bucket := os.Getenv("DATA_GUARD_BUCKET")
	if bucket == "" {
		fmt.Fprintln(os.Stderr, "Error: DATA_GUARD_BUCKET environment variable is required")
		os.Exit(2)
	}

	s3Endpoint := os.Getenv("DATA_GUARD_S3_ENDPOINT")
	sqlitePaths := os.Getenv("DATA_GUARD_SQLITE_PATHS")
	fsPaths := os.Getenv("DATA_GUARD_FS_PATHS")

	if sqlitePaths == "" && fsPaths == "" {
		fmt.Fprintln(os.Stderr, "Error: At least one of DATA_GUARD_SQLITE_PATHS or DATA_GUARD_FS_PATHS is required")
		os.Exit(2)
	}

	ctx := context.Background()
	hasError := false

	if sqlitePaths != "" {
		fmt.Println("Restoring SQLite databases...")
		paths := strings.Split(sqlitePaths, ",")
		for _, dbPath := range paths {
			dbPath = strings.TrimSpace(dbPath)
			if err := restoreSQLite(ctx, bucket, s3Endpoint, dbPath); err != nil {
				fmt.Fprintf(os.Stderr, "SQLite restore failed for %s: %v\n", dbPath, err)
				hasError = true
			}
		}
	}

	if fsPaths != "" {
		fmt.Println("Restoring filesystem paths...")
		paths := strings.Split(fsPaths, ",")
		for _, fsPath := range paths {
			fsPath = strings.TrimSpace(fsPath)
			if err := restoreFilesystem(ctx, bucket, s3Endpoint, fsPath); err != nil {
				fmt.Fprintf(os.Stderr, "Filesystem restore failed for %s: %v\n", fsPath, err)
				hasError = true
			}
		}
	}

	if hasError {
		os.Exit(1)
	}

	fmt.Println("Restore completed successfully")
	os.Exit(0)
}
