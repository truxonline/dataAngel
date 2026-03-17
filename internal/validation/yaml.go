package validation

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ValidateYAMLParse vérifie que le fichier YAML peut être parsé
func ValidateYAMLParse(filePath string) (bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("impossible de lire le fichier: %w", err)
	}

	var content interface{}
	err = yaml.Unmarshal(data, &content)
	if err != nil {
		return false, fmt.Errorf("impossible de parser le YAML: %w", err)
	}

	return true, nil
}

// ValidateYAMLStructure vérifie la structure du YAML
func ValidateYAMLStructure(filePath string, requiredKeys []string) (bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("impossible de lire le fichier: %w", err)
	}

	var content map[string]interface{}
	err = yaml.Unmarshal(data, &content)
	if err != nil {
		return false, fmt.Errorf("impossible de parser le YAML: %w", err)
	}

	for _, key := range requiredKeys {
		if _, exists := content[key]; !exists {
			return false, fmt.Errorf("clé requise manquante: %s", key)
		}
	}

	return true, nil
}
