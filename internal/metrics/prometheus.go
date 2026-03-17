package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics contient toutes les métriques Prometheus
type Metrics struct {
	BackupsTotal     prometheus.Counter
	BackupsSuccess   prometheus.Counter
	BackupsFailed    prometheus.Counter
	RestoresTotal    prometheus.Counter
	RestoresSuccess  prometheus.Counter
	RestoresFailed   prometheus.Counter
	BackupDuration   prometheus.Histogram
	LockAcquisitions prometheus.Counter
	LockContentions  prometheus.Counter
}

var (
	once    sync.Once
	metrics *Metrics
)

// GetMetrics retourne l'instance unique des métriques
func GetMetrics() *Metrics {
	once.Do(func() {
		metrics = &Metrics{
			BackupsTotal: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dataguard_backups_total",
				Help: "Total number of backup operations",
			}),
			BackupsSuccess: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dataguard_backups_success_total",
				Help: "Number of successful backup operations",
			}),
			BackupsFailed: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dataguard_backups_failed_total",
				Help: "Number of failed backup operations",
			}),
			RestoresTotal: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dataguard_restores_total",
				Help: "Total number of restore operations",
			}),
			RestoresSuccess: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dataguard_restores_success_total",
				Help: "Number of successful restore operations",
			}),
			RestoresFailed: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dataguard_restores_failed_total",
				Help: "Number of failed restore operations",
			}),
			BackupDuration: promauto.NewHistogram(prometheus.HistogramOpts{
				Name:    "dataguard_backup_duration_seconds",
				Help:    "Duration of backup operations in seconds",
				Buckets: prometheus.DefBuckets,
			}),
			LockAcquisitions: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dataguard_lock_acquisitions_total",
				Help: "Total number of lock acquisitions",
			}),
			LockContentions: promauto.NewCounter(prometheus.CounterOpts{
				Name: "dataguard_lock_contentions_total",
				Help: "Number of lock contentions detected",
			}),
		}
	})
	return metrics
}

// RecordBackupSuccess enregistre un backup réussi
func (m *Metrics) RecordBackupSuccess() {
	m.BackupsTotal.Inc()
	m.BackupsSuccess.Inc()
}

// RecordBackupFailure enregistre un backup échoué
func (m *Metrics) RecordBackupFailure() {
	m.BackupsTotal.Inc()
	m.BackupsFailed.Inc()
}

// RecordBackupDuration enregistre la durée d'un backup
func (m *Metrics) RecordBackupDuration(duration float64) {
	m.BackupDuration.Observe(duration)
}

// RecordRestoreSuccess enregistre une restauration réussie
func (m *Metrics) RecordRestoreSuccess() {
	m.RestoresTotal.Inc()
	m.RestoresSuccess.Inc()
}

// RecordRestoreFailure enregistre une restauration échouée
func (m *Metrics) RecordRestoreFailure() {
	m.RestoresTotal.Inc()
	m.RestoresFailed.Inc()
}

// RecordLockAcquisition enregistre une acquisition de lock
func (m *Metrics) RecordLockAcquisition() {
	m.LockAcquisitions.Inc()
}

// RecordLockContention enregistre une contention de lock
func (m *Metrics) RecordLockContention() {
	m.LockContentions.Inc()
}

// Reset réinitialise les métriques (pour les tests)
func (m *Metrics) Reset() {
	// Note: Prometheus ne supporte pas le reset directement
	// Cette méthode est principalement pour les tests
}
