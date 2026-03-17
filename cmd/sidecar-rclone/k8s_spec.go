package main

import (
	"fmt"
	"strings"
)

// GenerateK8sSidecarSpec génère la spec Kubernetes pour le sidecar Rclone
func GenerateK8sSidecarSpec(image string, configPath string, dataPath string, syncInterval int) string {
	var sb strings.Builder

	sb.WriteString("- name: rclone-sidecar\n")
	sb.WriteString(fmt.Sprintf("  image: %s\n", image))
	sb.WriteString("  imagePullPolicy: IfNotPresent\n")
	sb.WriteString("  volumeMounts:\n")
	sb.WriteString("  - name: config\n")
	sb.WriteString("    mountPath: /config\n")
	sb.WriteString("    readOnly: true\n")
	sb.WriteString("  - name: data\n")
	sb.WriteString(fmt.Sprintf("    mountPath: %s\n", dataPath))
	sb.WriteString("    readOnly: true\n")
	sb.WriteString("  resources:\n")
	sb.WriteString("    requests:\n")
	sb.WriteString("      memory: \"64Mi\"\n")
	sb.WriteString("      cpu: \"100m\"\n")
	sb.WriteString("    limits:\n")
	sb.WriteString("      memory: \"128Mi\"\n")
	sb.WriteString("      cpu: \"200m\"\n")
	sb.WriteString(fmt.Sprintf("  args: [\"sync\", \"%s\", \"s3:backup-bucket\", \"--config\", \"%s/rclone.conf\", \"--transfers\", \"4\", \"--checkers\", \"8\", \"--stats\", \"%ds\"]\n", dataPath, configPath, syncInterval))

	return sb.String()
}

// ValidateK8sSidecarSpec vérifie que la spec Kubernetes est valide
func ValidateK8sSidecarSpec(spec string) (bool, string) {
	if !strings.Contains(spec, "name: rclone-sidecar") {
		return false, "Le sidecar container doit avoir un nom"
	}
	if !strings.Contains(spec, "image:") {
		return false, "Le sidecar container doit avoir une image"
	}
	if !strings.Contains(spec, "volumeMounts:") {
		return false, "Le sidecar container doit avoir des volumeMounts"
	}
	if !strings.Contains(spec, "resources:") {
		return false, "Le sidecar container doit avoir des ressources"
	}
	if !strings.Contains(spec, "args:") {
		return false, "Le sidecar container doit avoir des args"
	}
	return true, ""
}
