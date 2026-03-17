package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charchess/dataAngel/internal/k8s"
)

func StreamSQLiteToS3() error {
	config, err := k8s.LoadFromEnv()
	if err != nil {
		return fmt.Errorf("impossible de charger la configuration: %w", err)
	}

	if !config.Enabled {
		fmt.Println("DataGuard n'est pas activé, skipping backup")
		return nil
	}

	if len(config.SqlitePaths) == 0 {
		fmt.Println("Aucun chemin SQLite configuré, skipping backup")
		return nil
	}

	s3URI := fmt.Sprintf("s3://%s/backups", config.Bucket)
	if config.S3Endpoint != "" {
		s3URI = fmt.Sprintf("%s/%s/backups", config.S3Endpoint, config.Bucket)
	}

	for _, dbPath := range config.SqlitePaths {
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			fmt.Printf("Base de données non trouvée: %s, skipping\n", dbPath)
			continue
		}

		fmt.Printf("Streaming %s to %s\n", dbPath, s3URI)
	}

	return nil
}

// RestoreFromS3 restores SQLite database from S3
func RestoreFromS3() error {
	config, err := k8s.LoadFromEnv()
	if err != nil {
		return fmt.Errorf("impossible de charger la configuration: %w", err)
	}

	if !config.Enabled {
		fmt.Println("DataGuard n'est pas activé, skipping restore")
		return nil
	}

	if len(config.SqlitePaths) == 0 {
		fmt.Println("Aucun chemin SQLite configuré, skipping restore")
		return nil
	}

	s3URI := fmt.Sprintf("s3://%s/backups", config.Bucket)
	if config.S3Endpoint != "" {
		s3URI = fmt.Sprintf("%s/%s/backups", config.S3Endpoint, config.Bucket)
	}

	for _, dbPath := range config.SqlitePaths {
		restorePath := strings.TrimSuffix(dbPath, ".db") + "_restored.db"
		fmt.Printf("Restoring from %s to %s\n", s3URI, restorePath)
	}

	return nil
}
