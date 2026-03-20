package sidecar

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	once           sync.Once
	sidecarMetrics *SidecarMetrics
)

// SidecarMetrics holds all Prometheus metrics for the sidecar daemon.
// Metrics are registered in the default Prometheus registry so they are
// served by the readiness server's /metrics endpoint (phase.go).
type SidecarMetrics struct {
	LitestreamUp    prometheus.Gauge
	RcloneUp        prometheus.Gauge
	SyncDuration    prometheus.Histogram
	SyncsTotal      prometheus.Counter
	SyncsFailed     prometheus.Counter
	YAMLValidations prometheus.Counter
	YAMLCacheHits   prometheus.Counter
	SidecarUptime   prometheus.Gauge
	startTime       time.Time
}

// GetMetrics returns the singleton SidecarMetrics instance
func GetMetrics() *SidecarMetrics {
	once.Do(func() {
		sidecarMetrics = &SidecarMetrics{
			LitestreamUp: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "dataguard_litestream_up",
				Help: "Litestream replication status (1=up, 0=down)",
			}),
			RcloneUp: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "dataguard_rclone_up",
				Help: "Rclone sync status (1=up, 0=down)",
			}),
			SyncDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
				Name:    "dataguard_rclone_sync_duration_seconds",
				Help:    "Duration of rclone sync operations",
				Buckets: prometheus.DefBuckets,
			}),
			SyncsTotal: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "dataguard_rclone_syncs_total",
				Help: "Total number of rclone sync operations",
			}),
			SyncsFailed: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "dataguard_rclone_syncs_failed_total",
				Help: "Total number of failed rclone sync operations",
			}),
			YAMLValidations: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "dataguard_yaml_validations_total",
				Help: "Total number of YAML validation operations",
			}),
			YAMLCacheHits: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "dataguard_yaml_cache_hits_total",
				Help: "Total number of YAML cache hits",
			}),
			SidecarUptime: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "dataguard_sidecar_uptime_seconds",
				Help: "Sidecar daemon uptime in seconds",
			}),
			startTime: time.Now(),
		}

		// Register all metrics in the default Prometheus registry
		prometheus.MustRegister(
			sidecarMetrics.LitestreamUp,
			sidecarMetrics.RcloneUp,
			sidecarMetrics.SyncDuration,
			sidecarMetrics.SyncsTotal,
			sidecarMetrics.SyncsFailed,
			sidecarMetrics.YAMLValidations,
			sidecarMetrics.YAMLCacheHits,
			sidecarMetrics.SidecarUptime,
		)

		// Start uptime updater
		go sidecarMetrics.updateUptime()
	})

	return sidecarMetrics
}

// updateUptime updates the uptime metric every 5 seconds
func (m *SidecarMetrics) updateUptime() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		uptime := time.Since(m.startTime).Seconds()
		m.SidecarUptime.Set(uptime)
	}
}

