package main

import (
	"strings"
	"testing"
)

// TestRcloneConfigFormat vérifie que la configuration Rclone est au bon format
func TestRcloneConfigFormat(t *testing.T) {
	content, err := ReadRcloneConfig()
	if err != nil {
		t.Skip("Configuration Rclone non trouvée, test ignoré")
	}
	if !strings.Contains(content, "[") || !strings.Contains(content, "]") {
		t.Errorf("La configuration Rclone doit contenir une section")
	}
}

// TestRcloneS3Backend vérifie que le backend S3 est correctement configuré
func TestRcloneS3Backend(t *testing.T) {
	content, err := ReadRcloneConfig()
	if err != nil {
		t.Skip("Configuration Rclone non trouvée, test ignoré")
	}
	if !strings.Contains(content, "type = s3") {
		t.Errorf("La configuration Rclone doit utiliser le backend S3")
	}
	if !strings.Contains(content, "provider = AWS") {
		t.Errorf("La configuration Rclone doit spécifier le provider AWS")
	}
}

// TestRcloneSyncIntervalConfig vérifie que l'intervalle de sync est configuré dans les args
func TestRcloneSyncIntervalConfig(t *testing.T) {
	// L'intervalle de sync est configuré dans le Dockerfile via --stats
	content, err := ReadDockerfile()
	if err != nil {
		t.Skip("Dockerfile non trouvé, test ignoré")
	}
	if !strings.Contains(content, "60s") {
		t.Errorf("L'intervalle de sync doit être configuré à 60s")
	}
}

// TestRcloneLogging vérifie que le logging est correctement configuré
func TestRcloneLogging(t *testing.T) {
	// Le logging est configuré via l'argument --stats dans le Dockerfile
	content, err := ReadDockerfile()
	if err != nil {
		t.Skip("Dockerfile non trouvé, test ignoré")
	}
	if !strings.Contains(content, "--stats") {
		t.Errorf("Le logging doit être configuré via --stats")
	}
}
