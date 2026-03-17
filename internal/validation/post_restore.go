package validation

import (
	"fmt"
	"log"
)

// AlertLevel représente le niveau de sévérité d'une alerte
type AlertLevel int

const (
	AlertLevelInfo AlertLevel = iota
	AlertLevelWarning
	AlertLevelError
	AlertLevelCritical
)

// Alert représente une alerte de validation
type Alert struct {
	Level   AlertLevel
	Message string
	Details string
}

// TriggerAlert déclenche une alerte en fonction du niveau
func TriggerAlert(alert Alert) {
	message := fmt.Sprintf("[%s] %s", alertLevelToString(alert.Level), alert.Message)
	if alert.Details != "" {
		message += fmt.Sprintf(" - Details: %s", alert.Details)
	}

	log.Println(message)
}

// alertLevelToString convertit le niveau d'alerte en chaîne
func alertLevelToString(level AlertLevel) string {
	switch level {
	case AlertLevelInfo:
		return "INFO"
	case AlertLevelWarning:
		return "WARNING"
	case AlertLevelError:
		return "ERROR"
	case AlertLevelCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// ValidateAndAlert exécute la validation et déclenche des alertes si nécessaire
func ValidateAndAlert(dbPath string) (bool, error) {
	// Validation SQLite
	valid, err := ValidateSQLiteIntegrity(dbPath)
	if err != nil {
		alert := Alert{
			Level:   AlertLevelCritical,
			Message: "Échec de la validation SQLite",
			Details: err.Error(),
		}
		TriggerAlert(alert)
		return false, err
	}
	if !valid {
		alert := Alert{
			Level:   AlertLevelError,
			Message: "Intégrité SQLite compromise",
			Details: "La base de données ne passe pas le check d'intégrité",
		}
		TriggerAlert(alert)
		return false, nil
	}

	// Validation WAL
	valid, err = ValidateWALState(dbPath)
	if err != nil {
		alert := Alert{
			Level:   AlertLevelError,
			Message: "Échec de la validation WAL",
			Details: err.Error(),
		}
		TriggerAlert(alert)
		return false, err
	}
	if !valid {
		alert := Alert{
			Level:   AlertLevelWarning,
			Message: "État WAL problématique",
			Details: "Le mode WAL n'est pas activé",
		}
		TriggerAlert(alert)
	}

	return true, nil
}
