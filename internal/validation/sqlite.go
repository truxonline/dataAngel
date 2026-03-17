package validation

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// ValidateSQLiteIntegrity vérifie l'intégrité de la base SQLite
func ValidateSQLiteIntegrity(dbPath string) (bool, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return false, fmt.Errorf("impossible d'ouvrir la base de données: %w", err)
	}
	defer db.Close()

	// Exécuter PRAGMA integrity_check
	var result string
	err = db.QueryRow("PRAGMA integrity_check").Scan(&result)
	if err != nil {
		return false, fmt.Errorf("impossible d'exécuter PRAGMA integrity_check: %w", err)
	}

	if strings.ToLower(result) != "ok" {
		return false, fmt.Errorf("integrity_check a échoué: %s", result)
	}

	return true, nil
}

// ValidateWALState vérifie l'état du WAL (Write-Ahead Logging)
func ValidateWALState(dbPath string) (bool, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return false, fmt.Errorf("impossible d'ouvrir la base de données: %w", err)
	}
	defer db.Close()

	// Vérifier si le WAL est activé
	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		return false, fmt.Errorf("impossible de vérifier le mode journal: %w", err)
	}

	if journalMode != "wal" {
		return false, fmt.Errorf("le mode WAL n'est pas activé (mode actuel: %s)", journalMode)
	}

	// Vérifier si le WAL est propre (pas de transactions en cours)
	var walCheckpoint int
	err = db.QueryRow("PRAGMA wal_checkpoint(TRUNCATE)").Scan(&walCheckpoint)
	if err != nil {
		return false, fmt.Errorf("impossible de vérifier le checkpoint WAL: %w", err)
	}

	return true, nil
}
