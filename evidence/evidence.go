// Package evidence provides evidence types and aggregation.
//
// Evidence is the foundation of STYX's belief system. Each piece of
// evidence contributes to the belief distribution about a node's liveness.
package evidence

import (
	"fmt"

	styxtime "github.com/styx-oracle/styx/time"
	"github.com/styx-oracle/styx/types"
)

// EventID is a unique identifier for a causal event.
type EventID uint64

// EvidenceKind represents the type of evidence observed.
type EvidenceKind int

const (
	// KindDirectResponse - received a direct response to a probe.
	// Strong evidence of liveness, but not absolute.
	KindDirectResponse EvidenceKind = iota

	// KindTimeout - no response within timeout period.
	// CRITICAL: Absence of evidence is NOT evidence of absence!
	// Property 4: No evidence implies no conclusion.
	// Property 15: Silence ≠ death.
	KindTimeout

	// KindWitnessReport - another node reports about this node.
	// Indirect evidence. Weight depends on trust.
	KindWitnessReport

	// KindCausalEvent - observed a causal event from the node.
	// Strong proof - node was alive when event was created.
	KindCausalEvent

	// KindSchedulingJitter - abnormal scheduling detected.
	// Property 6: Load ≠ failure. GC pauses must not trigger death.
	KindSchedulingJitter

	// KindNetworkInstability - network issues detected on path.
	KindNetworkInstability
)

func (k EvidenceKind) String() string {
	switch k {
	case KindDirectResponse:
		return "DirectResponse"
	case KindTimeout:
		return "Timeout"
	case KindWitnessReport:
		return "WitnessReport"
	case KindCausalEvent:
		return "CausalEvent"
	case KindSchedulingJitter:
		return "SchedulingJitter"
	case KindNetworkInstability:
		return "NetworkInstability"
	default:
		return "Unknown"
	}
}

// Evidence represents a single piece of evidence about a node's liveness.
type Evidence struct {
	Kind      EvidenceKind
	Timestamp styxtime.LogicalTimestamp
	Weight    float64
	Source    types.NodeID
	Target    types.NodeID
	Details   EvidenceDetails
}

// EvidenceDetails contains kind-specific details.
type EvidenceDetails struct {
	// DirectResponse
	LatencyMS uint64

	// Timeout
	ExpectedMS uint64
	WaitedMS   uint64

	// WitnessReport
	Witness       types.NodeID
	ReportedState types.BeliefState
	WitnessConf   float64

	// CausalEvent
	EventID EventID

	// SchedulingJitter
	ObservedDelayMS uint64

	// NetworkInstability
	PacketLossRate    float64
	LatencyVarianceMS uint64
}

// NewDirectResponse creates evidence of a direct response.
func NewDirectResponse(ts styxtime.LogicalTimestamp, latencyMS uint64, source, target types.NodeID) Evidence {
	weight := 1.0
	if latencyMS >= 100 && latencyMS < 1000 {
		weight = 0.8
	} else if latencyMS >= 1000 {
		weight = 0.6
	}
	return Evidence{
		Kind:      KindDirectResponse,
		Timestamp: ts,
		Weight:    weight,
		Source:    source,
		Target:    target,
		Details:   EvidenceDetails{LatencyMS: latencyMS},
	}
}

// NewTimeout creates evidence of a timeout.
// Per Property 4 and 15: timeouts are WEAK evidence, never proof of death.
func NewTimeout(ts styxtime.LogicalTimestamp, expectedMS, waitedMS uint64, source, target types.NodeID) Evidence {
	ratio := float64(waitedMS) / float64(expectedMS)
	weight := 0.1
	if ratio > 10.0 {
		weight = 0.3 // Still weak - silence ≠ death
	} else if ratio > 3.0 {
		weight = 0.2
	}
	return Evidence{
		Kind:      KindTimeout,
		Timestamp: ts,
		Weight:    weight,
		Source:    source,
		Target:    target,
		Details:   EvidenceDetails{ExpectedMS: expectedMS, WaitedMS: waitedMS},
	}
}

// NewCausalEvent creates evidence of a causal event.
func NewCausalEvent(ts styxtime.LogicalTimestamp, eventID EventID, source, target types.NodeID) Evidence {
	return Evidence{
		Kind:      KindCausalEvent,
		Timestamp: ts,
		Weight:    1.0,
		Source:    source,
		Target:    target,
		Details:   EvidenceDetails{EventID: eventID},
	}
}

// NewSchedulingJitter creates evidence of scheduling jitter.
// Per Property 6: This reduces confidence in OTHER evidence, not proof of death.
func NewSchedulingJitter(ts styxtime.LogicalTimestamp, delayMS uint64, source, target types.NodeID) Evidence {
	weight := 0.2
	if delayMS > 1000 {
		weight = 0.4
	}
	return Evidence{
		Kind:      KindSchedulingJitter,
		Timestamp: ts,
		Weight:    weight,
		Source:    source,
		Target:    target,
		Details:   EvidenceDetails{ObservedDelayMS: delayMS},
	}
}

// SuggestsAlive returns true if this evidence suggests the target is alive.
func (e Evidence) SuggestsAlive() bool {
	return e.Kind == KindDirectResponse || e.Kind == KindCausalEvent
}

// SuggestsDead returns true if this evidence suggests the target MIGHT be dead.
// Note: Per Property 15, this is never conclusive on its own.
func (e Evidence) SuggestsDead() bool {
	return e.Kind == KindTimeout
}

// EffectiveWeight returns weight adjusted for age decay.
func (e Evidence) EffectiveWeight(now styxtime.LogicalTimestamp, halfLife uint64) float64 {
	age := e.Timestamp.AgeSince(now)
	decayFactor := pow(0.5, float64(age)/float64(halfLife))
	return e.Weight * decayFactor
}

func pow(base, exp float64) float64 {
	if exp == 0 {
		return 1
	}
	result := 1.0
	for exp >= 1 {
		result *= base
		exp--
	}
	// Approximate for fractional part
	if exp > 0 {
		result *= 1 + exp*(base-1)
	}
	return result
}

func (e Evidence) String() string {
	return fmt.Sprintf("[%s] %s from %s about %s (w=%.2f)",
		e.Timestamp, e.Kind, e.Source, e.Target, e.Weight)
}
