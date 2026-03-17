package metrics

import (
	"testing"
)

func TestAlertOnBackupFailure(t *testing.T) {
	metrics := GetMetrics()
	metrics.RecordBackupFailure()
}

func TestAlertOnRestorePerformed(t *testing.T) {
	metrics := GetMetrics()
	metrics.RecordRestoreSuccess()
}
