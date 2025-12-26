package observer

import (
	"math"
	"sync"
	"time"
)

// ResponseEntropy measures unpredictability of responses.
//
// Consistent responses = higher confidence in liveness.
// Erratic responses = degraded confidence (something is wrong).
type ResponseEntropy struct {
	mu         sync.RWMutex
	latencies  []time.Duration
	windowSize int
}

// NewResponseEntropy creates a new entropy tracker.
func NewResponseEntropy(windowSize int) *ResponseEntropy {
	if windowSize < 1 {
		windowSize = 100
	}
	return &ResponseEntropy{
		latencies:  make([]time.Duration, 0, windowSize),
		windowSize: windowSize,
	}
}

// AddSample records a response latency.
func (re *ResponseEntropy) AddSample(latency time.Duration) {
	re.mu.Lock()
	defer re.mu.Unlock()

	if len(re.latencies) >= re.windowSize {
		re.latencies = re.latencies[1:]
	}
	re.latencies = append(re.latencies, latency)
}

// Entropy returns normalized entropy [0,1].
//
// Returns:
//   - 0.0: Perfectly consistent (all same latency) → high confidence
//   - 1.0: Maximum variance → low confidence
func (re *ResponseEntropy) Entropy() float64 {
	re.mu.RLock()
	defer re.mu.RUnlock()

	n := len(re.latencies)
	if n < 2 {
		return 0.5 // Insufficient data, neutral
	}

	// Calculate coefficient of variation (CV = stddev / mean)
	var sum float64
	for _, lat := range re.latencies {
		sum += float64(lat)
	}
	mean := sum / float64(n)

	if mean == 0 {
		return 0.5 // Edge case
	}

	var variance float64
	for _, lat := range re.latencies {
		diff := float64(lat) - mean
		variance += diff * diff
	}
	variance /= float64(n)
	stddev := math.Sqrt(variance)

	cv := stddev / mean

	// Normalize: CV of 0 → entropy 0, CV of 1+ → entropy 1
	// Cap at 1.0
	entropy := math.Min(cv, 1.0)
	return entropy
}

// ConfidenceFactor returns how much to trust responses.
//
// Low entropy → high trust (1.0)
// High entropy → low trust (0.5 minimum, never 0)
func (re *ResponseEntropy) ConfidenceFactor() float64 {
	entropy := re.Entropy()
	// Map entropy [0,1] to trust [1.0, 0.5]
	return 1.0 - (entropy * 0.5)
}

// IsErratic returns true if responses are highly variable.
func (re *ResponseEntropy) IsErratic() bool {
	return re.Entropy() > 0.5
}

// Stats returns current entropy statistics.
func (re *ResponseEntropy) Stats() EntropyStats {
	re.mu.RLock()
	defer re.mu.RUnlock()

	n := len(re.latencies)
	if n == 0 {
		return EntropyStats{}
	}

	var sum time.Duration
	var min, max time.Duration = time.Hour, 0
	for _, lat := range re.latencies {
		sum += lat
		if lat < min {
			min = lat
		}
		if lat > max {
			max = lat
		}
	}

	return EntropyStats{
		SampleCount: n,
		MeanLatency: sum / time.Duration(n),
		MinLatency:  min,
		MaxLatency:  max,
		Entropy:     re.Entropy(),
	}
}

// EntropyStats contains response entropy statistics.
type EntropyStats struct {
	SampleCount int
	MeanLatency time.Duration
	MinLatency  time.Duration
	MaxLatency  time.Duration
	Entropy     float64
}
