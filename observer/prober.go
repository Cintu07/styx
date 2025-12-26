package observer

import (
	"fmt"
	"sync"
	"time"

	"github.com/styx-oracle/styx/evidence"
	"github.com/styx-oracle/styx/state"
	styxtime "github.com/styx-oracle/styx/time"
	"github.com/styx-oracle/styx/types"
)

// ProbeResult represents the outcome of a probe.
type ProbeResult struct {
	Target    types.NodeID
	Success   bool
	Latency   time.Duration
	Error     error
	Timestamp styxtime.LogicalTimestamp
}

// ProbeFunc is a function that probes a target node.
// This is injected to allow simulated probing in tests.
type ProbeFunc func(target types.NodeID) ProbeResult

// Prober sends probes and collects responses.
//
// The Prober is responsible for:
// - Sending probes to target nodes
// - Tracking response latency and entropy
// - Accounting for local jitter (Property 6: load ≠ failure)
// - Recording evidence to the observer state
type Prober struct {
	mu           sync.Mutex
	selfID       types.NodeID
	state        *state.ObserverState
	jitter       *JitterTracker
	entropy      map[types.NodeID]*ResponseEntropy
	probeFunc    ProbeFunc
	probeTimeout time.Duration
}

// NewProber creates a new Prober.
func NewProber(selfID types.NodeID, probeTimeout time.Duration) *Prober {
	return &Prober{
		selfID:       selfID,
		state:        state.NewObserverState(selfID),
		jitter:       NewJitterTracker(100),
		entropy:      make(map[types.NodeID]*ResponseEntropy),
		probeTimeout: probeTimeout,
	}
}

// SetProbeFunc sets the function used to probe targets.
// Required before calling Probe().
func (p *Prober) SetProbeFunc(fn ProbeFunc) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.probeFunc = fn
}

// State returns the observer state.
func (p *Prober) State() *state.ObserverState {
	return p.state
}

// JitterTracker returns the jitter tracker.
func (p *Prober) JitterTracker() *JitterTracker {
	return p.jitter
}

// Probe sends a probe to the target and records evidence.
// Returns the updated belief about the target.
func (p *Prober) Probe(target types.NodeID) (types.Belief, error) {
	p.mu.Lock()
	probeFunc := p.probeFunc
	p.mu.Unlock()

	if probeFunc == nil {
		return types.UnknownBelief(), fmt.Errorf("no probe function set")
	}

	// Record expected timing for jitter measurement
	expectedDuration := p.probeTimeout / 2 // Expect response in half the timeout

	// Perform the probe
	start := time.Now()
	result := probeFunc(target)
	actualDuration := time.Since(start)

	// Record jitter sample (local scheduling delay)
	p.jitter.RecordSample(expectedDuration, actualDuration)

	// Get jitter factor to discount timeout evidence
	jitterFactor := p.jitter.GetJitterFactor()

	// Advance logical clock
	ts := p.state.Tick()

	// Record evidence based on result
	var ev evidence.Evidence
	if result.Success {
		// Direct response - strong evidence of liveness
		ev = evidence.NewDirectResponse(
			ts,
			uint64(result.Latency.Milliseconds()),
			p.selfID,
			target,
		)

		// Track entropy for this target
		p.getEntropy(target).AddSample(result.Latency)

		// Adjust weight by entropy confidence
		entropyFactor := p.getEntropy(target).ConfidenceFactor()
		ev.Weight *= entropyFactor
	} else {
		// Timeout - weak evidence, further discounted by jitter
		// Per Property 15: Silence ≠ death
		ev = NewJitterAwareTimeout(
			ts,
			uint64(p.probeTimeout.Milliseconds()),
			uint64(actualDuration.Milliseconds()),
			jitterFactor,
			p.selfID,
			target,
		)
	}

	// Record to observer state
	belief := p.state.RecordEvidence(target, ev)
	return belief, nil
}

// Query returns the current belief about a target.
func (p *Prober) Query(target types.NodeID) state.BeliefQuery {
	return p.state.QueryOrUnknown(target)
}

// getEntropy returns the entropy tracker for a target, creating if needed.
func (p *Prober) getEntropy(target types.NodeID) *ResponseEntropy {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.entropy[target] == nil {
		p.entropy[target] = NewResponseEntropy(50)
	}
	return p.entropy[target]
}

// NewJitterAwareTimeout creates timeout evidence adjusted for jitter.
//
// Per Property 6: Load ≠ failure.
// Per Property 15: Silence ≠ death.
//
// The jitterFactor discounts the evidence weight:
// - jitterFactor=1.0: Full weight (no jitter)
// - jitterFactor=0.0: Zero weight (extreme jitter, ignore timeout)
func NewJitterAwareTimeout(
	ts styxtime.LogicalTimestamp,
	expectedMS, waitedMS uint64,
	jitterFactor float64,
	source, target types.NodeID,
) evidence.Evidence {
	// Create base timeout evidence
	ev := evidence.NewTimeout(ts, expectedMS, waitedMS, source, target)

	// Discount by jitter factor
	// This implements Property 6: local load should not cause false death signals
	ev.Weight *= jitterFactor

	// Cap maximum weight for timeouts (Property 15: silence ≠ death)
	// Even with no jitter, a single timeout is weak evidence
	if ev.Weight > 0.3 {
		ev.Weight = 0.3
	}

	return ev
}
