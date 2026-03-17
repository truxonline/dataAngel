package cli

import (
	"testing"
)

func TestVerifyCommand_Parsing(t *testing.T) {
	args := []string{"verify", "--bucket", "myapp"}
	cmd, err := ParseCommand(args)
	if err != nil {
		t.Errorf("La commande devrait être parsée sans erreur: %v", err)
	}
	if cmd.Bucket != "myapp" {
		t.Errorf("Le bucket devrait être 'myapp', got '%s'", cmd.Bucket)
	}
}

func TestVerifyCommand_MissingBucket(t *testing.T) {
	args := []string{"verify"}
	_, err := ParseCommand(args)
	if err == nil {
		t.Errorf("Une commande sans bucket devrait échouer")
	}
}

func TestFormatBackupStatus_ValidBackups(t *testing.T) {
	status := "Healthy"
	result := FormatBackupStatus(status)
	if result != "Statut: Healthy" {
		t.Errorf("Le formatage devrait être 'Statut: Healthy', got '%s'", result)
	}
}

func TestFormatBackupStatus_NoBackups(t *testing.T) {
	status := "No backups found"
	result := FormatBackupStatus(status)
	if result != "Aucun backup trouvé" {
		t.Errorf("Le formatage devrait être 'Aucun backup trouvé', got '%s'", result)
	}
}
