package cli

import (
	"testing"
)

func TestForceReleaseCommand_Parsing(t *testing.T) {
	args := []string{"force-release", "--lock-id", "lock-123"}
	cmd, err := ParseForceReleaseCommand(args)
	if err != nil {
		t.Errorf("La commande devrait être parsée sans erreur: %v", err)
	}
	if cmd.LockID != "lock-123" {
		t.Errorf("Le lock ID devrait être 'lock-123', got '%s'", cmd.LockID)
	}
}

func TestForceReleaseCommand_MissingLockID(t *testing.T) {
	args := []string{"force-release"}
	_, err := ParseForceReleaseCommand(args)
	if err == nil {
		t.Errorf("Une commande sans lock ID devrait échouer")
	}
}
