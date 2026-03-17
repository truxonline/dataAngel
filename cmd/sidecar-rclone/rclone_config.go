package main

import (
	"fmt"
	"io/ioutil"
	"strings"
)

// ReadRcloneConfig lit le contenu du fichier de configuration Rclone
func ReadRcloneConfig() (string, error) {
	content, err := ioutil.ReadFile("rclone.conf")
	if err != nil {
		return "", fmt.Errorf("impossible de lire la configuration Rclone: %w", err)
	}
	return string(content), nil
}

// ValidateRcloneConfig vérifie que la configuration Rclone est valide
func ValidateRcloneConfig(content string) (bool, string) {
	if !strings.Contains(content, "[") || !strings.Contains(content, "]") {
		return false, "La configuration Rclone doit contenir une section"
	}
	if !strings.Contains(content, "type = s3") {
		return false, "La configuration Rclone doit utiliser le backend S3"
	}
	if !strings.Contains(content, "provider = AWS") {
		return false, "La configuration Rclone doit spécifier le provider AWS"
	}
	return true, ""
}
