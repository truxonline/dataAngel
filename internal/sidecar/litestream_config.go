package sidecar

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LitestreamConfig represents a litestream configuration file
type LitestreamConfig struct {
	DBs []LitestreamDB `yaml:"dbs"`
}

type LitestreamDB struct {
	Path     string              `yaml:"path"`
	Replicas []LitestreamReplica `yaml:"replicas"`
}

type LitestreamReplica struct {
	URL              string `yaml:"url"`
	Endpoint         string `yaml:"endpoint,omitempty"`
	SnapshotInterval string `yaml:"snapshot-interval,omitempty"`
}

// GenerateLitestreamConfig creates a litestream config file for a given SQLite database.
// snapshotInterval caps WAL chain length to bound restore time (e.g. "24h").
func GenerateLitestreamConfig(dbPath, bucket, s3Endpoint, outputPath, snapshotInterval string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	dbName := filepath.Base(dbPath)
	s3URL := fmt.Sprintf("s3://%s/%s", bucket, dbName)

	replica := LitestreamReplica{
		URL:              s3URL,
		SnapshotInterval: snapshotInterval,
	}

	if s3Endpoint != "" {
		replica.Endpoint = s3Endpoint
	}

	config := LitestreamConfig{
		DBs: []LitestreamDB{
			{
				Path: dbPath,
				Replicas: []LitestreamReplica{
					replica,
				},
			},
		},
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal litestream config: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write litestream config: %w", err)
	}

	return nil
}
