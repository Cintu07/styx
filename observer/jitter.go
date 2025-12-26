package observer

import (
	"math"
	"sync"
	"time"
)

// JitterTracker measures local scheduling delays.
//
// Per Property 6: Load ≠ failure.
// High scheduling jitter (GC pauses, CPU stalls) should NOT be
// interpreted as evidence of remote node failure.
//
// JitterTracker discounts timeout evidence when local jitter is high.
type JitterTracker struct {
	mu         sync.RWMutex
	samples    []float64 // jitter ratios: (actual-expected)/expected
	windowSize int
}

// NewJitterTracker creates a new jitter tracker.
func NewJitterTracker(windowSize int) *JitterTracker {
	if windowSize < 1 {
		windowSize = 100
	}
	return &JitterTracker{
		samples:    make([]float64, 0, windowSize),
		windowSize: windowSize,
	}
}

// RecordSample records a scheduling jitter sample.
//
// expected: how long the operation should have taken
// actual: how long it actually took
func (jt *JitterTracker) RecordSample(expected, actual time.Duration) {
	if expected <= 0 {
		return
	}

	ratio := float64(actual-expected) / float64(expected)
	if ratio < 0 {
		ratio = 0 // faster than expected is not jitter
	}

	jt.mu.Lock()
	defer jt.mu.Unlock()

	// Sliding window
	if len(jt.samples) >= jt.windowSize {
		jt.samples = jt.samples[1:]
	}
	jt.samples = append(jt.samples, ratio)
}

// GetJitterFactor returns a factor [0,1] representing
// how much to trust timeout evidence.
//
// Returns:
//   - 1.0: No jitter detected, full trust in timeouts
//   - 0.0: Extreme jitter, DO NOT trust timeouts (Property 6)
func (jt *JitterTracker) GetJitterFactor() float64 {
	jt.mu.RLock()
	defer jt.mu.RUnlock()

	if len(jt.samples) == 0 {
		return 1.0 // No data, assume no jitter
	}

	// Calculate mean and max jitter
	var sum, maxJitter float64
	for _, s := range jt.samples {
		sum += s
		if s > maxJitter {
			maxJitter = s
		}
	}
	mean := sum / float64(len(jt.samples))

	// If mean jitter > 50% or max > 200%, reduce trust significantly
	// This implements Property 6: Load ≠ failure
	if maxJitter > 2.0 {
		return 0.1 // Extreme jitter event detected
	}
	if mean > 0.5 {
		return 0.2 // Sustained high jitter
	}
	if mean > 0.2 {
		return 0.5 // Moderate jitter
	}

	// Low jitter: linear decay from 1.0 to 0.5 as mean goes 0 -> 0.2
	return 1.0 - (mean * 2.5)
}

// IsJittery returns true if significant jitter is detected.
func (jt *JitterTracker) IsJittery() bool {
	return jt.GetJitterFactor() < 0.8
}

// JitterStats returns current jitter statistics.
func (jt *JitterTracker) JitterStats() JitterStats {
	jt.mu.RLock()
	defer jt.mu.RUnlock()

	if len(jt.samples) == 0 {
		return JitterStats{}
	}

	var sum, max float64
	for _, s := range jt.samples {
		sum += s
		if s > max {
			max = s
		}
	}

	return JitterStats{
		SampleCount:  len(jt.samples),
		MeanJitter:   sum / float64(len(jt.samples)),
		MaxJitter:    max,
		JitterFactor: jt.GetJitterFactor(),
	}
}

// JitterStats contains jitter statistics.
type JitterStats struct {
	SampleCount  int
	MeanJitter   float64 // Ratio: (actual-expected)/expected
	MaxJitter    float64
	JitterFactor float64 // Trust factor [0,1]
}

// String returns a human-readable summary.
func (js JitterStats) String() string {
	if js.SampleCount == 0 {
		return "JitterStats(no samples)"
	}
	return "JitterStats(" +
		"samples=" + itoa(js.SampleCount) +
		", mean=" + ftoa(js.MeanJitter*100) + "%" +
		", max=" + ftoa(js.MaxJitter*100) + "%" +
		", trust=" + ftoa(js.JitterFactor*100) + "%)"
}

func itoa(i int) string {
	return string(rune('0' + i%10)) // simplified, just for display
}

func ftoa(f float64) string {
	// Simplified formatting
	return fmtFloat(f)
}

func fmtFloat(f float64) string {
	if math.IsNaN(f) {
		return "NaN"
	}
	if f < 0 {
		return "-" + fmtFloat(-f)
	}
	whole := int(f)
	frac := int((f - float64(whole)) * 100)
	return intToStr(whole) + "." + intToStr(frac)
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToStr(-n)
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}
