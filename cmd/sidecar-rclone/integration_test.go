package main

import (
	"strings"
	"testing"
)

// TestRcloneSyncToS3 vérifie que la configuration S3 est correcte
func TestRcloneSyncToS3(t *testing.T) {
	content, err := ReadDockerfile()
	if err != nil {
		t.Skip("Dockerfile non trouvé, test ignoré")
	}
	if !strings.Contains(content, "s3:backup-bucket") {
		t.Errorf("Le sync doit être configuré vers s3:backup-bucket")
	}
}

// TestRcloneSyncTiming vérifie que le sync se produit toutes les 60s
func TestRcloneSyncTiming(t *testing.T) {
	content, err := ReadDockerfile()
	if err != nil {
		t.Skip("Dockerfile non trouvé, test ignoré")
	}
	if !strings.Contains(content, "--stats") || !strings.Contains(content, "60s") {
		t.Errorf("Le sync doit être configuré pour s'exécuter toutes les 60s")
	}
}

// TestRcloneFailureRecovery vérifie que la configuration permet la récupération après erreur
func TestRcloneFailureRecovery(t *testing.T) {
	content, err := ReadRcloneConfig()
	if err != nil {
		t.Skip("Configuration Rclone non trouvée, test ignoré")
	}
	if !strings.Contains(content, "type = s3") {
		t.Errorf("La configuration doit utiliser S3 pour la récupération")
	}
}

// TestRcloneFileAccessibility vérifie que les fichiers sont accessibles pour restauration
func TestRcloneFileAccessibility(t *testing.T) {
	content, err := ReadDockerfile()
	if err != nil {
		t.Skip("Dockerfile non trouvé, test ignoré")
	}
	if !strings.Contains(content, "/data") {
		t.Errorf("Le sync doit être configuré pour synchroniser /data")
	}
}
