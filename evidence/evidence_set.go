package evidence

import (
	"math"

	styxtime "github.com/styx-oracle/styx/time"
	"github.com/styx-oracle/styx/types"
)

// DefaultHalfLife for evidence decay (in logical time units).
const DefaultHalfLife uint64 = 100

// EvidenceSet aggregates evidence about a single node.
// Implements Property 5: Evidence is monotonic (append-only).
// Implements Property 9: Conflicting evidence widens belief.
type EvidenceSet struct {
	evidence []Evidence
	halfLife uint64
}

// NewEvidenceSet creates a new, empty evidence set.
func NewEvidenceSet() *EvidenceSet {
	return &EvidenceSet{
		evidence: make([]Evidence, 0),
		halfLife: DefaultHalfLife,
	}
}

// WithHalfLife creates an evidence set with custom decay.
func WithHalfLife(halfLife uint64) *EvidenceSet {
	return &EvidenceSet{
		evidence: make([]Evidence, 0),
		halfLife: halfLife,
	}
}

// Add appends new evidence (monotonic, per Property 5).
func (es *EvidenceSet) Add(e Evidence) {
	es.evidence = append(es.evidence, e)
}

// Len returns the number of evidence records.
func (es *EvidenceSet) Len() int {
	return len(es.evidence)
}

// IsEmpty checks if the evidence set is empty.
func (es *EvidenceSet) IsEmpty() bool {
	return len(es.evidence) == 0
}

// All returns all evidence records.
func (es *EvidenceSet) All() []Evidence {
	return es.evidence
}

// ComputeBelief aggregates all evidence into a belief distribution.
//
// Implements:
//   - Property 7: Belief is never binary (confidence ∈ (0,1))
//   - Property 8: Unknown is always allowed (never forced to zero)
//   - Property 9: Conflicting evidence widens belief (more conflict → more uncertainty)
//   - Property 18: Confidence sums to 1
func (es *EvidenceSet) ComputeBelief(now styxtime.LogicalTimestamp) types.Belief {
	if es.IsEmpty() {
		return types.UnknownBelief() // Property 8: Unknown is always allowed
	}

	var aliveWeight, deadWeight, totalWeight float64

	for _, e := range es.evidence {
		w := e.EffectiveWeight(now, es.halfLife)
		totalWeight += w

		if e.SuggestsAlive() {
			aliveWeight += w
		} else if e.SuggestsDead() {
			deadWeight += w
		}
	}

	if totalWeight < 1e-10 {
		return types.UnknownBelief()
	}

	// Property 7: Never binary - cap certainty
	// Property 8: Always leave room for unknown
	maxCertainty := math.Min(totalWeight/(totalWeight+1.0), 0.90) // Never exceed 90% from evidence alone

	aliveRatio := aliveWeight / totalWeight
	deadRatio := deadWeight / totalWeight

	// Property 9: Conflicting evidence widens belief
	// If both alive and dead evidence exist, increase uncertainty
	conflictFactor := 1.0
	if aliveWeight > 0 && deadWeight > 0 {
		// More balanced conflict = more uncertainty
		balance := math.Min(aliveWeight, deadWeight) / math.Max(aliveWeight, deadWeight)
		conflictFactor = 1.0 - (balance * 0.5) // Reduce certainty when conflicted
	}

	aliveConf := aliveRatio * maxCertainty * conflictFactor
	deadConf := deadRatio * maxCertainty * conflictFactor
	unknownConf := 1.0 - aliveConf - deadConf

	// Property 8: Ensure unknown is never zero
	if unknownConf < 0.05 {
		excess := 0.05 - unknownConf
		aliveConf -= excess / 2
		deadConf -= excess / 2
		unknownConf = 0.05
	}

	belief, err := types.NewBelief(aliveConf, deadConf, unknownConf)
	if err != nil {
		return types.UnknownBelief()
	}
	return belief
}

// ComputeBeliefNow computes belief using the latest evidence timestamp.
func (es *EvidenceSet) ComputeBeliefNow() types.Belief {
	var max styxtime.LogicalTimestamp
	for _, e := range es.evidence {
		if e.Timestamp > max {
			max = e.Timestamp
		}
	}
	return es.ComputeBelief(max)
}

// LatestTimestamp returns the most recent evidence timestamp.
func (es *EvidenceSet) LatestTimestamp() styxtime.LogicalTimestamp {
	var max styxtime.LogicalTimestamp
	for _, e := range es.evidence {
		if e.Timestamp > max {
			max = e.Timestamp
		}
	}
	return max
}

// AliveEvidence returns evidence suggesting the node is alive.
func (es *EvidenceSet) AliveEvidence() []Evidence {
	result := make([]Evidence, 0)
	for _, e := range es.evidence {
		if e.SuggestsAlive() {
			result = append(result, e)
		}
	}
	return result
}

// DeadEvidence returns evidence suggesting the node might be dead.
func (es *EvidenceSet) DeadEvidence() []Evidence {
	result := make([]Evidence, 0)
	for _, e := range es.evidence {
		if e.SuggestsDead() {
			result = append(result, e)
		}
	}
	return result
}
