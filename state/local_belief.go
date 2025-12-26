// Package state provides local state management for observers.
package state

import (
	"fmt"

	"github.com/styx-oracle/styx/evidence"
	styxtime "github.com/styx-oracle/styx/time"
	"github.com/styx-oracle/styx/types"
)

// LocalBelief represents what a single observer believes about a target node.
type LocalBelief struct {
	target      types.NodeID
	belief      types.Belief
	evidence    *evidence.EvidenceSet
	lastUpdated styxtime.LogicalTimestamp
}

// NewLocalBelief creates a new LocalBelief for a target node.
// Starts with pure uncertainty and no evidence.
func NewLocalBelief(target types.NodeID) *LocalBelief {
	return &LocalBelief{
		target:      target,
		belief:      types.UnknownBelief(),
		evidence:    evidence.NewEvidenceSet(),
		lastUpdated: styxtime.Zero(),
	}
}

// Target returns the target node.
func (lb *LocalBelief) Target() types.NodeID {
	return lb.target
}

// Belief returns the current belief distribution.
func (lb *LocalBelief) Belief() types.Belief {
	return lb.belief
}

// Evidence returns the evidence set.
func (lb *LocalBelief) Evidence() *evidence.EvidenceSet {
	return lb.evidence
}

// LastUpdated returns when this belief was last updated.
func (lb *LocalBelief) LastUpdated() styxtime.LogicalTimestamp {
	return lb.lastUpdated
}

// RecordEvidence adds new evidence and recomputes the belief.
func (lb *LocalBelief) RecordEvidence(e evidence.Evidence) types.Belief {
	if e.Timestamp > lb.lastUpdated {
		lb.lastUpdated = e.Timestamp
	}
	lb.evidence.Add(e)
	lb.belief = lb.evidence.ComputeBelief(lb.lastUpdated)
	return lb.belief
}

// RecomputeAt recomputes the belief at a given time (for decay).
func (lb *LocalBelief) RecomputeAt(now styxtime.LogicalTimestamp) {
	lb.belief = lb.evidence.ComputeBelief(now)
	lb.lastUpdated = now
}

// IsCertainAlive checks if we're certain the target is alive.
func (lb *LocalBelief) IsCertainAlive() bool {
	return lb.belief.IsCertainAlive()
}

// IsCertainDead checks if we're certain the target is dead.
func (lb *LocalBelief) IsCertainDead() bool {
	return lb.belief.IsCertainDead()
}

// Reasoning returns a summary of why we believe what we believe.
func (lb *LocalBelief) Reasoning() BeliefReasoning {
	return BeliefReasoning{
		Belief:             lb.belief,
		EvidenceCount:      lb.evidence.Len(),
		AliveEvidenceCount: len(lb.evidence.AliveEvidence()),
		DeadEvidenceCount:  len(lb.evidence.DeadEvidence()),
		LatestEvidence:     lb.evidence.LatestTimestamp(),
	}
}

func (lb *LocalBelief) String() string {
	return fmt.Sprintf("LocalBelief(%s → %s, %d evidence)",
		lb.target, lb.belief, lb.evidence.Len())
}

// BeliefReasoning summarizes why we hold a particular belief.
type BeliefReasoning struct {
	Belief             types.Belief
	EvidenceCount      int
	AliveEvidenceCount int
	DeadEvidenceCount  int
	LatestEvidence     styxtime.LogicalTimestamp
}

func (br BeliefReasoning) String() string {
	return fmt.Sprintf("%s (evidence: %d total, %d alive, %d dead)",
		br.Belief, br.EvidenceCount, br.AliveEvidenceCount, br.DeadEvidenceCount)
}
