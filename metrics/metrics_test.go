package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsEndpoint(t *testing.T) {
	m := &Metrics{}

	// Record some data
	m.RecordQuery(100, false)
	m.RecordQuery(50, true)
	m.RecordReport()
	m.RecordDeath()
	m.RecordPartition()
	m.SetWitnessCount(5)

	handler := m.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Check all metrics are present
	checks := []string{
		"styx_queries_total 2",
		"styx_reports_total 1",
		"styx_refusals_total 1",
		"styx_deaths_total 1",
		"styx_partitions_detected_total 1",
		"styx_witnesses 5",
	}

	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Errorf("expected metric %q in output", check)
		}
	}
}

func TestMetricsContentType(t *testing.T) {
	m := &Metrics{}
	handler := m.Handler()

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("expected text/plain content type, got %s", contentType)
	}
}
