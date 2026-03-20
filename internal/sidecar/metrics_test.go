package sidecar

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestGetMetrics(t *testing.T) {
	t.Run("should return singleton instance", func(t *testing.T) {
		// ACT
		m1 := GetMetrics()
		m2 := GetMetrics()

		// ASSERT
		if m1 != m2 {
			t.Error("GetMetrics should return same instance")
		}
	})

	t.Run("should initialize all sidecar metrics", func(t *testing.T) {
		// ACT
		m := GetMetrics()

		// ASSERT
		if m.LitestreamUp == nil {
			t.Error("LitestreamUp should be initialized")
		}
		if m.RcloneUp == nil {
			t.Error("RcloneUp should be initialized")
		}
		if m.SyncDuration == nil {
			t.Error("SyncDuration should be initialized")
		}
		if m.SyncsTotal == nil {
			t.Error("SyncsTotal should be initialized")
		}
		if m.SyncsFailed == nil {
			t.Error("SyncsFailed should be initialized")
		}
		if m.YAMLValidations == nil {
			t.Error("YAMLValidations should be initialized")
		}
		if m.YAMLCacheHits == nil {
			t.Error("YAMLCacheHits should be initialized")
		}
		if m.SidecarUptime == nil {
			t.Error("SidecarUptime should be initialized")
		}
	})

	t.Run("should record metric values", func(t *testing.T) {
		// ARRANGE
		m := GetMetrics()

		// ACT
		m.LitestreamUp.Set(1)
		m.SyncsTotal.Inc()
		m.SyncDuration.Observe(1.5)

		// ASSERT — no panic, basic functionality works
	})
}

func TestMetricsInDefaultRegistry(t *testing.T) {
	t.Run("should serve sidecar metrics via default registry", func(t *testing.T) {
		// ARRANGE — ensure metrics are initialized
		GetMetrics()
		handler := promhttp.Handler() // default registry
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()

		// ACT
		handler.ServeHTTP(w, req)

		// ASSERT
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		body := w.Body.String()
		if !strings.Contains(body, "dataguard_litestream_up") {
			t.Error("Default registry should contain dataguard_litestream_up metric")
		}
		if !strings.Contains(body, "dataguard_rclone_up") {
			t.Error("Default registry should contain dataguard_rclone_up metric")
		}
		if !strings.Contains(body, "dataguard_rclone_syncs_failed_total") {
			t.Error("Default registry should contain dataguard_rclone_syncs_failed_total metric")
		}
	})
}
