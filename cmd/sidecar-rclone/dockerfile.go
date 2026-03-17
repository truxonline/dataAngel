package main

import (
	"fmt"
	"io/ioutil"
	"strings"
)

// ReadDockerfile lit le contenu du Dockerfile
func ReadDockerfile() (string, error) {
	content, err := ioutil.ReadFile("Dockerfile")
	if err != nil {
		return "", fmt.Errorf("impossible de lire le Dockerfile: %w", err)
	}
	return string(content), nil
}

// ValidateDockerfile vérifie que le Dockerfile est valide
func ValidateDockerfile(content string) (bool, string) {
	if !strings.Contains(content, "FROM") {
		return false, "Le Dockerfile doit contenir une instruction FROM"
	}
	if !strings.Contains(content, "rclone") {
		return false, "Le Dockerfile doit utiliser une image rclone"
	}
	if !strings.Contains(content, "WORKDIR") {
		return false, "Le Dockerfile doit définir un WORKDIR"
	}
	if !strings.Contains(content, "ENTRYPOINT") {
		return false, "Le Dockerfile doit définir un ENTRYPOINT"
	}
	return true, ""
}
