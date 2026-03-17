package metrics

import (
	"testing"
)

func TestMetricsRegistration(t *testing.T) {
	metrics := GetMetrics()
	if metrics == nil {
		t.Errorf("GetMetrics ne devrait pas retourner nil")
	}

	if metrics.BackupsTotal == nil {
		t.Errorf("BackupsTotal devrait être initialisé")
	}

	if metrics.BackupsSuccess == nil {
		t.Errorf("BackupsSuccess devrait être initialisé")
	}
}

func TestMetricsIncrement(t *testing.T) {
	metrics := GetMetrics()

	metrics.RecordBackupSuccess()
	metrics.RecordBackupFailure()
	metrics.RecordRestoreSuccess()
	metrics.RecordRestoreFailure()
	metrics.RecordLockAcquisition()
	metrics.RecordLockContention()
}

func TestMetricsGaugeUpdate(t *testing.T) {
	metrics := GetMetrics()

	metrics.RecordBackupDuration(1.5)
	metrics.RecordBackupDuration(2.0)
}
