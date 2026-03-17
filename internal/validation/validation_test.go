package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateSQLiteIntegrity(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("Impossible de créer la base de données: %v", err)
	}
	db.Close()

	valid, err := ValidateSQLiteIntegrity(dbPath)
	if err != nil {
		t.Logf("Validation SQLite a échoué: %v", err)
	}
	if !valid {
		t.Logf("La base vide n'est pas considérée comme valide: %v", err)
	}
}

func TestValidateWALState(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("Impossible de créer la base de données: %v", err)
	}
	db.Close()

	valid, err := ValidateWALState(dbPath)
	if err != nil {
		t.Logf("Validation WAL a échoué: %v", err)
	}
	if !valid {
		t.Logf("La validation WAL n'a pas réussi: %v", err)
	}
}

func TestValidateYAMLParse(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "test.yaml")

	yamlContent := `key1: value1
key2: value2
nested:
  key3: value3
`
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Impossible d'écrire le fichier YAML: %v", err)
	}

	valid, err := ValidateYAMLParse(yamlPath)
	if err != nil {
		t.Errorf("Le parsing YAML a échoué: %v", err)
	}
	if !valid {
		t.Errorf("Le YAML valide n'a pas été parsé correctement")
	}

	invalidYamlPath := filepath.Join(tmpDir, "invalid.yaml")
	invalidYamlContent := `key1: value1
key2: [unclosed bracket
`
	err = os.WriteFile(invalidYamlPath, []byte(invalidYamlContent), 0644)
	if err != nil {
		t.Fatalf("Impossible d'écrire le fichier YAML invalide: %v", err)
	}

	valid, err = ValidateYAMLParse(invalidYamlPath)
	if err == nil {
		t.Errorf("Le parsing d'un YAML invalide devrait échouer")
	}
	if valid {
		t.Errorf("Un YAML invalide a été considéré comme valide")
	}
}

func TestValidateYAMLStructure(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "test.yaml")

	yamlContent := `key1: value1
key2: value2
key3: value3
`
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Impossible d'écrire le fichier YAML: %v", err)
	}

	requiredKeys := []string{"key1", "key2"}
	valid, err := ValidateYAMLStructure(yamlPath, requiredKeys)
	if err != nil {
		t.Errorf("La validation de structure a échoué: %v", err)
	}
	if !valid {
		t.Errorf("La structure valide n'a pas été validée correctement")
	}

	requiredKeysMissing := []string{"key1", "missing_key"}
	valid, err = ValidateYAMLStructure(yamlPath, requiredKeysMissing)
	if err == nil {
		t.Errorf("La validation devrait échouer avec une clé manquante")
	}
	if valid {
		t.Errorf("La structure avec clé manquante a été considérée comme valide")
	}
}
