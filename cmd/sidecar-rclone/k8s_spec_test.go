package main

import (
	"strings"
	"testing"
)

// TestK8sSidecarContainer vérifie que le sidecar container est correctement défini
func TestK8sSidecarContainer(t *testing.T) {
	spec := GenerateK8sSidecarSpec("rclone:latest", "/config", "/data", 60)
	if !strings.Contains(spec, "name: rclone-sidecar") {
		t.Errorf("Le sidecar container doit avoir un nom")
	}
	if !strings.Contains(spec, "image: rclone:latest") {
		t.Errorf("Le sidecar container doit avoir une image")
	}
}

// TestK8sVolumeMounts vérifie que les volumes sont montés en lecture seule
func TestK8sVolumeMounts(t *testing.T) {
	spec := GenerateK8sSidecarSpec("rclone:latest", "/config", "/data", 60)
	if !strings.Contains(spec, "mountPath: /config") {
		t.Errorf("Le volume config doit être monté")
	}
	if !strings.Contains(spec, "readOnly: true") {
		t.Errorf("Les volumes doivent être en lecture seule")
	}
}

// TestK8sResourceLimits vérifie que les limites de ressources sont configurées
func TestK8sResourceLimits(t *testing.T) {
	spec := GenerateK8sSidecarSpec("rclone:latest", "/config", "/data", 60)
	if !strings.Contains(spec, "memory:") {
		t.Errorf("Les limites de mémoire doivent être configurées")
	}
	if !strings.Contains(spec, "cpu:") {
		t.Errorf("Les limites de CPU doivent être configurées")
	}
}

// TestK8sSyncInterval vérifie que l'intervalle de sync est configuré
func TestK8sSyncInterval(t *testing.T) {
	spec := GenerateK8sSidecarSpec("rclone:latest", "/config", "/data", 60)
	if !strings.Contains(spec, "--stats") || !strings.Contains(spec, "60s") {
		t.Errorf("L'intervalle de sync doit être configuré à 60s")
	}
}
