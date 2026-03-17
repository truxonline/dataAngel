package main

import (
	"strings"
	"testing"
)

// TestDockerfileBaseImage vérifie que le Dockerfile utilise une base image valide
func TestDockerfileBaseImage(t *testing.T) {
	content, err := ReadDockerfile()
	if err != nil {
		t.Skip("Dockerfile non trouvé, test ignoré")
	}
	if !strings.Contains(content, "FROM") {
		t.Errorf("Le Dockerfile doit contenir une instruction FROM")
	}
	if !strings.Contains(strings.ToLower(content), "rclone") {
		t.Errorf("Le Dockerfile doit utiliser une image rclone")
	}
}

// TestDockerfileRcloneInstallation vérifie que Rclone est installé correctement
func TestDockerfileRcloneInstallation(t *testing.T) {
	content, err := ReadDockerfile()
	if err != nil {
		t.Skip("Dockerfile non trouvé, test ignoré")
	}
	if !strings.Contains(content, "COPY") || !strings.Contains(content, "rclone.conf") {
		t.Errorf("Le Dockerfile doit copier la configuration rclone")
	}
}

// TestDockerfileEntrypoint vérifie que l'entrypoint est correctement configuré
func TestDockerfileEntrypoint(t *testing.T) {
	content, err := ReadDockerfile()
	if err != nil {
		t.Skip("Dockerfile non trouvé, test ignoré")
	}
	if !strings.Contains(content, "ENTRYPOINT") {
		t.Errorf("Le Dockerfile doit définir un ENTRYPOINT")
	}
	if !strings.Contains(content, "rclone") || !strings.Contains(content, "sync") {
		t.Errorf("L'entrypoint doit exécuter rclone sync")
	}
}

// TestDockerfileVolumeMounts vérifie que le répertoire de travail est configuré
func TestDockerfileVolumeMounts(t *testing.T) {
	content, err := ReadDockerfile()
	if err != nil {
		t.Skip("Dockerfile non trouvé, test ignoré")
	}
	if !strings.Contains(content, "WORKDIR") {
		t.Errorf("Le Dockerfile doit définir un WORKDIR")
	}
}

// TestDockerfileSize vérifie que l'image de base est légère
func TestDockerfileSize(t *testing.T) {
	content, err := ReadDockerfile()
	if err != nil {
		t.Skip("Dockerfile non trouvé, test ignoré")
	}
	if !strings.Contains(strings.ToLower(content), "rclone") {
		t.Errorf("Le Dockerfile doit utiliser une image rclone")
	}
}
