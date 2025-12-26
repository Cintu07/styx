package metrics

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Metrics tracks STYX operational metrics
type Metrics struct {
	mu sync.RWMutex

	// Counters
	QueriesTotal       int64
	ReportsTotal       int64
	RefusalsTotal      int64
	DeathsTotal        int64
	PartitionsDetected int64

	// Gauges
	WitnessCount   int
	ActiveNodes    int
	CurrentUnknown float64

	// Histograms (simplified as averages)
	QueryLatencySum   time.Duration
	QueryLatencyCount int64
}

// Global metrics instance
var Default = &Metrics{}

// RecordQuery records a query
func (m *Metrics) RecordQuery(latency time.Duration, refused bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.QueriesTotal++
	m.QueryLatencySum += latency
	m.QueryLatencyCount++

	if refused {
		m.RefusalsTotal++
	}
}

// RecordReport records a witness report
func (m *Metrics) RecordReport() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ReportsTotal++
}

// RecordDeath records a death declaration
func (m *Metrics) RecordDeath() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DeathsTotal++
}

// RecordPartition records a partition detection
func (m *Metrics) RecordPartition() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PartitionsDetected++
}

// SetWitnessCount sets current witness count
func (m *Metrics) SetWitnessCount(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.WitnessCount = count
}

// Handler returns Prometheus-compatible metrics endpoint
func (m *Metrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		defer m.mu.RUnlock()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		// Counters
		writeMetric(w, "styx_queries_total", "counter", "Total queries processed", m.QueriesTotal)
		writeMetric(w, "styx_reports_total", "counter", "Total witness reports received", m.ReportsTotal)
		writeMetric(w, "styx_refusals_total", "counter", "Total query refusals", m.RefusalsTotal)
		writeMetric(w, "styx_deaths_total", "counter", "Total death declarations", m.DeathsTotal)
		writeMetric(w, "styx_partitions_detected_total", "counter", "Total partitions detected", m.PartitionsDetected)

		// Gauges
		writeMetric(w, "styx_witnesses", "gauge", "Current witness count", int64(m.WitnessCount))
		writeMetric(w, "styx_active_nodes", "gauge", "Current active nodes", int64(m.ActiveNodes))

		// Query latency
		if m.QueryLatencyCount > 0 {
			avgMs := float64(m.QueryLatencySum.Milliseconds()) / float64(m.QueryLatencyCount)
			w.Write([]byte("# HELP styx_query_latency_avg_ms Average query latency in milliseconds\n"))
			w.Write([]byte("# TYPE styx_query_latency_avg_ms gauge\n"))
			w.Write([]byte("styx_query_latency_avg_ms " + formatFloat(avgMs) + "\n"))
		}
	}
}

func writeMetric(w http.ResponseWriter, name, mtype, help string, value int64) {
	w.Write([]byte("# HELP " + name + " " + help + "\n"))
	w.Write([]byte("# TYPE " + name + " " + mtype + "\n"))
	w.Write([]byte(name + " " + strconv.FormatInt(value, 10) + "\n"))
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 3, 64)
}
