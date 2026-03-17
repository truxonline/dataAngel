package sidecar

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds sidecar daemon configuration
type Config struct {
	Bucket          string
	S3Endpoint      string
	SqlitePaths     []string
	FsPaths         []string
	YAMLPaths       []string
	RcloneInterval  time.Duration
	MetricsPort     int
	ShutdownTimeout time.Duration
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() (Config, error) {
	bucket := os.Getenv("DATA_GUARD_BUCKET")
	if bucket == "" {
		return Config{}, fmt.Errorf("DATA_GUARD_BUCKET environment variable is required")
	}

	s3Endpoint := os.Getenv("DATA_GUARD_S3_ENDPOINT")
	sqlitePaths := parseCSV(os.Getenv("DATA_GUARD_SQLITE_PATHS"))
	fsPaths := parseCSV(os.Getenv("DATA_GUARD_FS_PATHS"))
	yamlPaths := parseCSV(os.Getenv("DATA_GUARD_YAML_PATHS"))

	// Parse rclone interval (default 60s)
	rcloneInterval := 60 * time.Second
	if intervalStr := os.Getenv("DATA_GUARD_RCLONE_INTERVAL"); intervalStr != "" {
		seconds, err := strconv.Atoi(intervalStr)
		if err == nil && seconds > 0 {
			rcloneInterval = time.Duration(seconds) * time.Second
		}
	}

	// Parse metrics port (default 9090)
	metricsPort := 9090
	if portStr := os.Getenv("DATA_GUARD_METRICS_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err == nil && port > 0 {
			metricsPort = port
		}
	}

	// Parse shutdown timeout (default 15s for litestream WAL flush)
	shutdownTimeout := 15 * time.Second
	if timeoutStr := os.Getenv("DATA_GUARD_SHUTDOWN_TIMEOUT"); timeoutStr != "" {
		seconds, err := strconv.Atoi(timeoutStr)
		if err == nil && seconds > 0 {
			shutdownTimeout = time.Duration(seconds) * time.Second
		}
	}

	return Config{
		Bucket:          bucket,
		S3Endpoint:      s3Endpoint,
		SqlitePaths:     sqlitePaths,
		FsPaths:         fsPaths,
		YAMLPaths:       yamlPaths,
		RcloneInterval:  rcloneInterval,
		MetricsPort:     metricsPort,
		ShutdownTimeout: shutdownTimeout,
	}, nil
}

// parseCSV splits a comma-separated string and trims whitespace
func parseCSV(input string) []string {
	if input == "" {
		return nil
	}

	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
