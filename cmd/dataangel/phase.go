package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Phase represents the current execution phase
type Phase string

const (
	PhaseRestore Phase = "restore"
	PhaseBackup  Phase = "backup"
)

type PhaseManager struct {
	mu             sync.RWMutex
	currentPhase   Phase
	lockAcquired   bool
	metricsPort    int
	metricsEnabled bool

	phaseGauge      *prometheus.GaugeVec
	restoreDuration prometheus.Gauge
}

// NewPhaseManager creates a new phase manager
func NewPhaseManager(metricsPort int, metricsEnabled bool) *PhaseManager {
	pm := &PhaseManager{
		currentPhase:   PhaseRestore,
		metricsPort:    metricsPort,
		metricsEnabled: metricsEnabled,
	}

	if metricsEnabled {
		// Register phase gauge (0 = restore, 1 = backup)
		pm.phaseGauge = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "dataangel_phase",
				Help: "Current phase of dataangel (0=restore, 1=backup)",
			},
			[]string{"phase"},
		)
		prometheus.MustRegister(pm.phaseGauge)

		// Register restore duration gauge
		pm.restoreDuration = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "dataangel_restore_duration_seconds",
				Help: "Duration of the restore phase in seconds",
			},
		)
		prometheus.MustRegister(pm.restoreDuration)

		// Initialize metrics
		pm.phaseGauge.WithLabelValues("restore").Set(1)
		pm.phaseGauge.WithLabelValues("backup").Set(0)
	}

	return pm
}

// SetPhase updates the current phase and metrics
func (pm *PhaseManager) SetPhase(phase Phase) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.currentPhase = phase

	if pm.metricsEnabled {
		switch phase {
		case PhaseRestore:
			pm.phaseGauge.WithLabelValues("restore").Set(1)
			pm.phaseGauge.WithLabelValues("backup").Set(0)
		case PhaseBackup:
			pm.phaseGauge.WithLabelValues("restore").Set(0)
			pm.phaseGauge.WithLabelValues("backup").Set(1)
		}
	}

	log.Printf("[dataangel] Phase transition: %s", phase)
}

// GetPhase returns the current phase (thread-safe)
func (pm *PhaseManager) GetPhase() Phase {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.currentPhase
}

func (pm *PhaseManager) RecordRestoreDuration(duration time.Duration) {
	if pm.metricsEnabled {
		pm.restoreDuration.Set(duration.Seconds())
	}
}

func (pm *PhaseManager) SetLockAcquired(acquired bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.lockAcquired = acquired
}

// StartReadinessServer starts the HTTP server for readiness probe and metrics
func (pm *PhaseManager) StartReadinessServer() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		pm.mu.RLock()
		phase := pm.currentPhase
		lockAcquired := pm.lockAcquired
		pm.mu.RUnlock()

		if phase == PhaseRestore {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("restore in progress\n"))
			return
		}

		if phase == PhaseBackup && !lockAcquired {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("waiting for lock acquisition\n"))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok\n"))
	})

	// Metrics endpoint (only if enabled)
	if pm.metricsEnabled {
		mux.Handle("/metrics", promhttp.Handler())
	}

	addr := fmt.Sprintf(":%d", pm.metricsPort)
	log.Printf("[dataangel] Starting readiness/metrics server on %s", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return server.ListenAndServe()
}
