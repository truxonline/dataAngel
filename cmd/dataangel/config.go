package main

import (
	"fmt"
	"os"
	"path/filepath"
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

	// Lock configuration
	DeploymentName     string
	LockTTL            time.Duration
	LockEnabled        bool

	// Restore configuration
	RestoreTimeout time.Duration

	// Backup daemon configuration
	RcloneInterval  time.Duration
	ShutdownTimeout time.Duration

	// Metrics configuration
	MetricsEnabled bool
	MetricsPort    int

	// Rclone configuration
	ExcludePatterns    []string
	SyncTimeout        time.Duration
	LockAcquireTimeout time.Duration
	RcloneDelay        time.Duration
	RcloneTransfers    int
	RcloneCheckers     int
	RcloneBwlimit      string

	// Logging
	FullLogs bool
}

// detectPrefixCollisions checks for basename collisions in paths (issue #32).
func detectPrefixCollisions(sqlitePaths, fsPaths []string) error {
	seen := make(map[string]string)
	for _, p := range sqlitePaths {
		base := filepath.Base(p)
		if existing, ok := seen[base]; ok {
			return fmt.Errorf("S3 prefix collision: %q and %q both map to prefix %q", existing, p, base)
		}
		seen[base] = p
	}
	seen = make(map[string]string)
	for _, p := range fsPaths {
		base := filepath.Base(p)
		if existing, ok := seen[base]; ok {
			return fmt.Errorf("S3 prefix collision: %q and %q both map to prefix %q", existing, p, base)
		}
		seen[base] = p
	}
	return nil
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

	// Parse restore timeout (default: 30m)
	restoreTimeoutStr := os.Getenv("DATA_GUARD_RESTORE_TIMEOUT")
	if restoreTimeoutStr == "" {
		restoreTimeoutStr = "30m"
	}
	restoreTimeout, err := time.ParseDuration(restoreTimeoutStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid DATA_GUARD_RESTORE_TIMEOUT: %w", err)
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

	// Parse deployment name (required for distributed lock)
	deploymentName := os.Getenv("DATA_GUARD_DEPLOYMENT_NAME")
	if deploymentName == "" {
		return Config{}, fmt.Errorf("DATA_GUARD_DEPLOYMENT_NAME is required")
	}

	// Parse lock TTL (default: 60s)
	lockTTLStr := os.Getenv("DATA_GUARD_LOCK_TTL")
	if lockTTLStr == "" {
		lockTTLStr = "60s"
	}
	lockTTL, err := time.ParseDuration(lockTTLStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid DATA_GUARD_LOCK_TTL: %w", err)
	}

	// Parse lock enabled (default: true)
	lockEnabled := true
	if s := os.Getenv("DATA_GUARD_LOCK_ENABLED"); s != "" {
		lockEnabled, err = strconv.ParseBool(s)
		if err != nil {
			return Config{}, fmt.Errorf("invalid DATA_GUARD_LOCK_ENABLED: %w", err)
		}
	}

	// Parse exclude patterns (default: "*.db*,.*.db-litestream/**") (#31)
	excludePatternsStr := os.Getenv("DATA_GUARD_EXCLUDE_PATTERNS")
	var excludePatterns []string
	if excludePatternsStr != "" {
		excludePatterns = strings.Split(excludePatternsStr, ",")
	} else {
		excludePatterns = []string{"*.db*", ".*.db-litestream/**", "*.dataangel-clean"}
	}

	// Parse sync timeout (default: 3m) (#36)
	syncTimeoutStr := os.Getenv("DATA_GUARD_SYNC_TIMEOUT")
	if syncTimeoutStr == "" {
		syncTimeoutStr = "3m"
	}
	syncTimeout, err := time.ParseDuration(syncTimeoutStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid DATA_GUARD_SYNC_TIMEOUT: %w", err)
	}

	// Parse lock acquire timeout (default: 5m) (#36)
	lockAcquireTimeoutStr := os.Getenv("DATA_GUARD_LOCK_ACQUIRE_TIMEOUT")
	if lockAcquireTimeoutStr == "" {
		lockAcquireTimeoutStr = "5m"
	}
	lockAcquireTimeout, err := time.ParseDuration(lockAcquireTimeoutStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid DATA_GUARD_LOCK_ACQUIRE_TIMEOUT: %w", err)
	}

	// Parse rclone delay (default: 30s) (#36)
	rcloneDelayStr := os.Getenv("DATA_GUARD_RCLONE_DELAY")
	if rcloneDelayStr == "" {
		rcloneDelayStr = "30s"
	}
	rcloneDelay, err := time.ParseDuration(rcloneDelayStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid DATA_GUARD_RCLONE_DELAY: %w", err)
	}

	// Parse rclone transfers (default: 1) (#36)
	rcloneTransfers := 1
	if s := os.Getenv("DATA_GUARD_RCLONE_TRANSFERS"); s != "" {
		if v, e := strconv.Atoi(s); e == nil && v > 0 {
			rcloneTransfers = v
		}
	}

	// Parse rclone checkers (default: 2) (#36)
	rcloneCheckers := 2
	if s := os.Getenv("DATA_GUARD_RCLONE_CHECKERS"); s != "" {
		if v, e := strconv.Atoi(s); e == nil && v > 0 {
			rcloneCheckers = v
		}
	}

	// Parse rclone bandwidth limit (default: "" = unlimited) (#36)
	rcloneBwlimit := os.Getenv("DATA_GUARD_RCLONE_BWLIMIT")

	// Detect S3 prefix collisions (#32)
	if err := detectPrefixCollisions(sqlitePaths, fsPaths); err != nil {
		return Config{}, err
	}

	return Config{
		Bucket:             bucket,
		S3Endpoint:         os.Getenv("DATA_GUARD_S3_ENDPOINT"),
		SqlitePaths:        sqlitePaths,
		FsPaths:            fsPaths,
		YAMLPaths:          yamlPaths,
		RestoreTimeout:     restoreTimeout,
		DeploymentName:     deploymentName,
		LockTTL:            lockTTL,
		LockEnabled:        lockEnabled,
		LockAcquireTimeout: lockAcquireTimeout,
		RcloneInterval:     rcloneInterval,
		RcloneDelay:        rcloneDelay,
		SyncTimeout:        syncTimeout,
		RcloneTransfers:    rcloneTransfers,
		RcloneCheckers:     rcloneCheckers,
		RcloneBwlimit:      rcloneBwlimit,
		ExcludePatterns:    excludePatterns,
		ShutdownTimeout:    shutdownTimeout,
		MetricsEnabled:     metricsEnabled,
		MetricsPort:        metricsPort,
		FullLogs:           fullLogs,
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
		RcloneDelay:     c.RcloneDelay,
		SyncTimeout:     c.SyncTimeout,
		RcloneTransfers: c.RcloneTransfers,
		RcloneCheckers:  c.RcloneCheckers,
		RcloneBwlimit:   c.RcloneBwlimit,
		ExcludePatterns: c.ExcludePatterns,
		ShutdownTimeout: c.ShutdownTimeout,
		MetricsEnabled:  c.MetricsEnabled,
		MetricsPort:     c.MetricsPort,
	}
}
