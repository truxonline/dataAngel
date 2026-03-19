package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charchess/dataAngel/internal/sidecar"
)

// Config holds the unified configuration for both restore and backup phases
type Config struct {
	// S3 configuration
	Bucket     string
	S3Endpoint string

	// Data paths
	SqlitePaths []string
	FsPaths     []string
	YAMLPaths   []string

	// Backup daemon configuration
	RcloneInterval  time.Duration
	ShutdownTimeout time.Duration

	// Metrics configuration
	MetricsEnabled bool
	MetricsPort    int

	// Logging
	FullLogs bool
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (Config, error) {
	bucket := os.Getenv("DATA_GUARD_BUCKET")
	if bucket == "" {
		return Config{}, fmt.Errorf("DATA_GUARD_BUCKET is required")
	}

	sqlitePathsStr := os.Getenv("DATA_GUARD_SQLITE_PATHS")
	fsPathsStr := os.Getenv("DATA_GUARD_FS_PATHS")
	yamlPathsStr := os.Getenv("DATA_GUARD_YAML_PATHS")

	// At least one of sqlite or fs paths must be set
	if sqlitePathsStr == "" && fsPathsStr == "" {
		return Config{}, fmt.Errorf("at least one of DATA_GUARD_SQLITE_PATHS or DATA_GUARD_FS_PATHS must be set")
	}

	// Parse paths
	var sqlitePaths []string
	if sqlitePathsStr != "" {
		sqlitePaths = strings.Split(sqlitePathsStr, ",")
	}

	var fsPaths []string
	if fsPathsStr != "" {
		fsPaths = strings.Split(fsPathsStr, ",")
	}

	var yamlPaths []string
	if yamlPathsStr != "" {
		yamlPaths = strings.Split(yamlPathsStr, ",")
	}

	// Parse rclone interval (default: 60s)
	rcloneIntervalStr := os.Getenv("DATA_GUARD_RCLONE_INTERVAL")
	if rcloneIntervalStr == "" {
		rcloneIntervalStr = "60s"
	}
	rcloneInterval, err := time.ParseDuration(rcloneIntervalStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid DATA_GUARD_RCLONE_INTERVAL: %w", err)
	}

	// Parse shutdown timeout (default: 15s)
	shutdownTimeoutStr := os.Getenv("DATA_GUARD_SHUTDOWN_TIMEOUT")
	if shutdownTimeoutStr == "" {
		shutdownTimeoutStr = "15s"
	}
	shutdownTimeout, err := time.ParseDuration(shutdownTimeoutStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid DATA_GUARD_SHUTDOWN_TIMEOUT: %w", err)
	}

	// Parse metrics enabled (default: true)
	metricsEnabledStr := os.Getenv("DATA_GUARD_METRICS_ENABLED")
	metricsEnabled := true // default
	if metricsEnabledStr != "" {
		metricsEnabled, err = strconv.ParseBool(metricsEnabledStr)
		if err != nil {
			return Config{}, fmt.Errorf("invalid DATA_GUARD_METRICS_ENABLED: %w", err)
		}
	}

	// Parse metrics port (default: 9090)
	metricsPortStr := os.Getenv("DATA_GUARD_METRICS_PORT")
	if metricsPortStr == "" {
		metricsPortStr = "9090"
	}
	metricsPort, err := strconv.Atoi(metricsPortStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid DATA_GUARD_METRICS_PORT: %w", err)
	}

	// Parse full logs (default: false)
	fullLogsStr := os.Getenv("DATA_GUARD_FULL_LOGS")
	fullLogs := false
	if fullLogsStr != "" {
		fullLogs, err = strconv.ParseBool(fullLogsStr)
		if err != nil {
			return Config{}, fmt.Errorf("invalid DATA_GUARD_FULL_LOGS: %w", err)
		}
	}

	return Config{
		Bucket:          bucket,
		S3Endpoint:      os.Getenv("DATA_GUARD_S3_ENDPOINT"),
		SqlitePaths:     sqlitePaths,
		FsPaths:         fsPaths,
		YAMLPaths:       yamlPaths,
		RcloneInterval:  rcloneInterval,
		ShutdownTimeout: shutdownTimeout,
		MetricsEnabled:  metricsEnabled,
		MetricsPort:     metricsPort,
		FullLogs:        fullLogs,
	}, nil
}

// ToSidecarConfig converts to internal/sidecar.Config for the backup phase
func (c Config) ToSidecarConfig() sidecar.Config {
	return sidecar.Config{
		Bucket:          c.Bucket,
		S3Endpoint:      c.S3Endpoint,
		SqlitePaths:     c.SqlitePaths,
		FsPaths:         c.FsPaths,
		YAMLPaths:       c.YAMLPaths,
		RcloneInterval:  c.RcloneInterval,
		ShutdownTimeout: c.ShutdownTimeout,
		MetricsEnabled:  c.MetricsEnabled,
		MetricsPort:     c.MetricsPort,
	}
}
