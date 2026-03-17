package cli

import (
	"fmt"
	"strings"
)

type VerifyCommand struct {
	Bucket string
}

func ParseCommand(args []string) (*VerifyCommand, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("manque d'arguments")
	}

	for i, arg := range args {
		if arg == "--bucket" && i+1 < len(args) {
			return &VerifyCommand{Bucket: args[i+1]}, nil
		}
		if strings.HasPrefix(arg, "--bucket=") {
			bucket := strings.TrimPrefix(arg, "--bucket=")
			return &VerifyCommand{Bucket: bucket}, nil
		}
	}

	return nil, fmt.Errorf("option --bucket manquante")
}

func FormatBackupStatus(status string) string {
	if status == "" || status == "No backups found" {
		return "Aucun backup trouvé"
	}
	return fmt.Sprintf("Statut: %s", status)
}
