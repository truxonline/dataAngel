package restore

import "log"

// ShouldSkip determines if the restore process should be skipped based on the decision.
func ShouldSkip(decision RestoreDecision) bool {
	return decision == DecisionSkip
}

// LogSkipReason logs the reason for skipping the restore.
func LogSkipReason(decision RestoreDecision) {
	switch decision {
	case DecisionSkip:
		log.Println("Skipping restore: local data is up to date")
	case DecisionRestore:
		log.Println("Restore needed: local data is outdated or missing")
	case DecisionCorrupted:
		log.Println("Restore needed: local data is corrupted")
	default:
		log.Printf("Unknown decision: %v", decision)
	}
}
