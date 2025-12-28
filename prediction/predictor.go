package prediction

import (
	"math"
	"sync"
	"time"
)

// Predictor uses historical data to predict failures
type Predictor struct {
	mu      sync.RWMutex
	history map[uint64]*NodeHistory
	window  time.Duration
}

// NodeHistory tracks historical behavior of a node
type NodeHistory struct {
	NodeID         uint64
	ResponseTimes  []time.Duration
	FailureEvents  []time.Time
	LastSeen       time.Time
	AvgResponse    time.Duration
	StdDevResponse time.Duration
}

// Prediction contains failure prediction results
type Prediction struct {
	NodeID             uint64
	FailureProbability float64
	TimeToFailure      time.Duration
	Confidence         float64
	Reason             string
}

// NewPredictor creates a new ML predictor
func NewPredictor(window time.Duration) *Predictor {
	return &Predictor{
		history: make(map[uint64]*NodeHistory),
		window:  window,
	}
}

// RecordResponse records a successful response
func (p *Predictor) RecordResponse(nodeID uint64, responseTime time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	h := p.getOrCreateHistory(nodeID)
	h.ResponseTimes = append(h.ResponseTimes, responseTime)
	h.LastSeen = time.Now()

	// Keep last 1000 samples
	if len(h.ResponseTimes) > 1000 {
		h.ResponseTimes = h.ResponseTimes[len(h.ResponseTimes)-1000:]
	}

	// Update statistics
	p.updateStats(h)
}

// RecordFailure records a failure event
func (p *Predictor) RecordFailure(nodeID uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	h := p.getOrCreateHistory(nodeID)
	h.FailureEvents = append(h.FailureEvents, time.Now())

	// Keep last 100 failures
	if len(h.FailureEvents) > 100 {
		h.FailureEvents = h.FailureEvents[len(h.FailureEvents)-100:]
	}
}

// Predict returns failure prediction for a node
func (p *Predictor) Predict(nodeID uint64) *Prediction {
	p.mu.RLock()
	defer p.mu.RUnlock()

	h, exists := p.history[nodeID]
	if !exists {
		return &Prediction{
			NodeID:             nodeID,
			FailureProbability: 0.5,
			Confidence:         0.0,
			Reason:             "no history available",
		}
	}

	pred := &Prediction{NodeID: nodeID}

	// Factor 1: Time since last seen
	timeSinceLastSeen := time.Since(h.LastSeen)
	if timeSinceLastSeen > p.window {
		pred.FailureProbability = 0.8
		pred.Reason = "not seen recently"
		pred.Confidence = 0.7
		return pred
	}

	// Factor 2: Response time anomaly detection
	if len(h.ResponseTimes) > 10 {
		recentAvg := p.recentAverage(h.ResponseTimes, 10)
		if recentAvg > h.AvgResponse*2 {
			pred.FailureProbability = 0.6
			pred.Reason = "response times degrading"
			pred.Confidence = 0.6
			return pred
		}
	}

	// Factor 3: Failure frequency
	recentFailures := p.countRecentFailures(h, time.Hour)
	if recentFailures > 3 {
		pred.FailureProbability = 0.7
		pred.Reason = "frequent recent failures"
		pred.Confidence = 0.65
		pred.TimeToFailure = time.Minute * 5 // Estimated
		return pred
	}

	// Factor 4: Pattern detection (simplified)
	if p.detectDegradationPattern(h) {
		pred.FailureProbability = 0.5
		pred.Reason = "degradation pattern detected"
		pred.Confidence = 0.5
		pred.TimeToFailure = time.Minute * 15
		return pred
	}

	// All good
	pred.FailureProbability = 0.1
	pred.Reason = "node appears healthy"
	pred.Confidence = 0.8
	return pred
}

func (p *Predictor) getOrCreateHistory(nodeID uint64) *NodeHistory {
	if h, exists := p.history[nodeID]; exists {
		return h
	}
	h := &NodeHistory{
		NodeID:        nodeID,
		ResponseTimes: make([]time.Duration, 0),
		FailureEvents: make([]time.Time, 0),
	}
	p.history[nodeID] = h
	return h
}

func (p *Predictor) updateStats(h *NodeHistory) {
	if len(h.ResponseTimes) == 0 {
		return
	}

	// Calculate average
	var sum time.Duration
	for _, rt := range h.ResponseTimes {
		sum += rt
	}
	h.AvgResponse = sum / time.Duration(len(h.ResponseTimes))

	// Calculate standard deviation
	var variance float64
	avg := float64(h.AvgResponse)
	for _, rt := range h.ResponseTimes {
		diff := float64(rt) - avg
		variance += diff * diff
	}
	variance /= float64(len(h.ResponseTimes))
	h.StdDevResponse = time.Duration(math.Sqrt(variance))
}

func (p *Predictor) recentAverage(times []time.Duration, n int) time.Duration {
	if len(times) < n {
		n = len(times)
	}
	recent := times[len(times)-n:]
	var sum time.Duration
	for _, t := range recent {
		sum += t
	}
	return sum / time.Duration(n)
}

func (p *Predictor) countRecentFailures(h *NodeHistory, window time.Duration) int {
	cutoff := time.Now().Add(-window)
	count := 0
	for _, t := range h.FailureEvents {
		if t.After(cutoff) {
			count++
		}
	}
	return count
}

func (p *Predictor) detectDegradationPattern(h *NodeHistory) bool {
	if len(h.ResponseTimes) < 20 {
		return false
	}

	// Check if recent responses are consistently slower than older ones
	oldAvg := p.recentAverage(h.ResponseTimes[:len(h.ResponseTimes)/2], 10)
	newAvg := p.recentAverage(h.ResponseTimes, 10)

	return newAvg > oldAvg*1.5
}
