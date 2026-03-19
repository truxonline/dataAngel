package k8s

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// DataGuardConfig représente la configuration DataGuard extraite des annotations
type DataGuardConfig struct {
	Enabled     bool
	Bucket      string
	S3Endpoint  string
	SqlitePaths []string
	FsPaths     []string
	FullLogs    bool
}

// ParseAnnotations extrait la configuration à partir des annotations Kubernetes
func ParseAnnotations(annotations map[string]string) (*DataGuardConfig, error) {
	config := &DataGuardConfig{}

	// dataangel.io/enabled (requis)
	if enabledStr, ok := annotations["dataangel.io/enabled"]; ok {
		enabled, err := strconv.ParseBool(enabledStr)
		if err != nil {
			return nil, fmt.Errorf("valeur invalide pour dataangel.io/enabled: %w", err)
		}
		config.Enabled = enabled
	}

	// Si DataGuard n'est pas activé, retourner config vide
	if !config.Enabled {
		return config, nil
	}

	// dataangel.io/bucket (requis si activé)
	if bucket, ok := annotations["dataangel.io/bucket"]; ok && bucket != "" {
		config.Bucket = bucket
	} else {
		return nil, fmt.Errorf("annotation dataangel.io/bucket est requise quand dataangel.io/enabled=true")
	}

	// dataangel.io/s3-endpoint (optionnel)
	if endpoint, ok := annotations["dataangel.io/s3-endpoint"]; ok && endpoint != "" {
		config.S3Endpoint = endpoint
	}

	// dataangel.io/sqlite-paths (optionnel)
	if pathsStr, ok := annotations["dataangel.io/sqlite-paths"]; ok && pathsStr != "" {
		config.SqlitePaths = parseCSV(pathsStr)
	}

	// dataangel.io/fs-paths (optionnel)
	if pathsStr, ok := annotations["dataangel.io/fs-paths"]; ok && pathsStr != "" {
		config.FsPaths = parseCSV(pathsStr)
	}

	// dataangel.io/full-logs (optionnel)
	if logsStr, ok := annotations["dataangel.io/full-logs"]; ok && logsStr != "" {
		fullLogs, err := strconv.ParseBool(logsStr)
		if err != nil {
			return nil, fmt.Errorf("valeur invalide pour dataangel.io/full-logs: %w", err)
		}
		config.FullLogs = fullLogs
	}

	return config, nil
}

// ToEnvVars convertit la configuration en variables d'environnement
func (c *DataGuardConfig) ToEnvVars() []string {
	envVars := []string{
		fmt.Sprintf("DATA_GUARD_ENABLED=%v", c.Enabled),
	}

	if c.Bucket != "" {
		envVars = append(envVars, fmt.Sprintf("DATA_GUARD_BUCKET=%s", c.Bucket))
	}

	if c.S3Endpoint != "" {
		envVars = append(envVars, fmt.Sprintf("DATA_GUARD_S3_ENDPOINT=%s", c.S3Endpoint))
	}

	if len(c.SqlitePaths) > 0 {
		envVars = append(envVars, fmt.Sprintf("DATA_GUARD_SQLITE_PATHS=%s", strings.Join(c.SqlitePaths, ",")))
	}

	if len(c.FsPaths) > 0 {
		envVars = append(envVars, fmt.Sprintf("DATA_GUARD_FS_PATHS=%s", strings.Join(c.FsPaths, ",")))
	}

	envVars = append(envVars, fmt.Sprintf("DATA_GUARD_FULL_LOGS=%v", c.FullLogs))

	return envVars
}

// parseCSV parse une chaîne de caractères séparée par des virgules
func parseCSV(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// LoadFromAnnotations charge la configuration depuis les variables d'environnement
// (utilisé par les sidecars pour lire la configuration injectée)
func LoadFromEnv() (*DataGuardConfig, error) {
	config := &DataGuardConfig{}

	enabledStr := os.Getenv("DATA_GUARD_ENABLED")
	if enabledStr == "" {
		config.Enabled = false
		return config, nil
	}

	enabled, err := strconv.ParseBool(enabledStr)
	if err != nil {
		return nil, fmt.Errorf("valeur invalide pour DATA_GUARD_ENABLED: %w", err)
	}
	config.Enabled = enabled

	if !config.Enabled {
		return config, nil
	}

	config.Bucket = os.Getenv("DATA_GUARD_BUCKET")
	if config.Bucket == "" {
		return nil, fmt.Errorf("DATA_GUARD_BUCKET est requis quand DATA_GUARD_ENABLED=true")
	}

	config.S3Endpoint = os.Getenv("DATA_GUARD_S3_ENDPOINT")

	if pathsStr := os.Getenv("DATA_GUARD_SQLITE_PATHS"); pathsStr != "" {
		config.SqlitePaths = parseCSV(pathsStr)
	}

	if pathsStr := os.Getenv("DATA_GUARD_FS_PATHS"); pathsStr != "" {
		config.FsPaths = parseCSV(pathsStr)
	}

	if logsStr := os.Getenv("DATA_GUARD_FULL_LOGS"); logsStr != "" {
		fullLogs, err := strconv.ParseBool(logsStr)
		if err != nil {
			return nil, fmt.Errorf("valeur invalide pour DATA_GUARD_FULL_LOGS: %w", err)
		}
		config.FullLogs = fullLogs
	}

	return config, nil
}
