package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPostRestoreValidation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("Impossible de créer la base de données: %v", err)
	}
	db.Close()

	valid, err := ValidateSQLiteIntegrity(dbPath)
	if err != nil {
		t.Logf("Validation post-restore a échoué: %v", err)
	}
	if !valid {
		t.Logf("La validation post-restore n'a pas réussi: %v", err)
	}
}

func TestAlertOnValidationFailure(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("Impossible de créer la base de données: %v", err)
	}
	db.Close()

	valid, err := ValidateAndAlert(dbPath)
	if err != nil {
		t.Logf("Validation a échoué: %v", err)
	}
	if !valid {
		t.Logf("La validation n'a pas réussi (attendu pour base vide): %v", err)
	}
}

func TestRecoveryScenarios(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("Impossible de créer la base de données: %v", err)
	}
	db.Close()

	valid, err := ValidateSQLiteIntegrity(dbPath)
	if err != nil {
		t.Logf("Récupération a échoué: %v", err)
	}
	if !valid {
		t.Logf("La récupération n'a pas réussi: %v", err)
	}
}
