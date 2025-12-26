package evidence_test

import (
	"testing"

	"github.com/styx-oracle/styx/evidence"
	styxtime "github.com/styx-oracle/styx/time"
	"github.com/styx-oracle/styx/types"
)

func testNodes() (types.NodeID, types.NodeID) {
	return types.NewNodeID(1), types.NewNodeID(2)
}

// Property 4: No evidence implies no conclusion
// Empty evidence set must yield pure uncertainty, not death.
func TestProperty4_NoEvidenceNoConclusion(t *testing.T) {
	es := evidence.NewEvidenceSet()
	belief := es.ComputeBelief(styxtime.Zero())

	if !belief.Equal(types.UnknownBelief()) {
		t.Error("Property 4 violated: empty evidence must yield unknown, not death")
	}
}

// Property 5: Observation is monotonic
// Evidence can only be added, never removed or rewritten.
func TestProperty5_EvidenceMonotonic(t *testing.T) {
	source, target := testNodes()
	es := evidence.NewEvidenceSet()

	e1 := evidence.NewDirectResponse(styxtime.LogicalTimestamp(1), 50, source, target)
	es.Add(e1)
	count1 := es.Len()

	e2 := evidence.NewDirectResponse(styxtime.LogicalTimestamp(2), 100, source, target)
	es.Add(e2)
	count2 := es.Len()

	if count2 <= count1 {
		t.Error("Property 5 violated: evidence count must increase monotonically")
	}

	// Verify all evidence is preserved
	all := es.All()
	if len(all) != 2 {
		t.Errorf("Property 5 violated: expected 2 evidence, got %d", len(all))
	}
}

// Property 6: Load ≠ failure
// Scheduling jitter must not trigger death signal.
func TestProperty6_LoadNotFailure(t *testing.T) {
	source, target := testNodes()
	es := evidence.NewEvidenceSet()

	// Add ONLY jitter evidence
	jitter := evidence.NewSchedulingJitter(styxtime.LogicalTimestamp(1), 5000, source, target)
	es.Add(jitter)

	belief := es.ComputeBelief(styxtime.LogicalTimestamp(1))

	// Jitter alone must NOT suggest death
	if belief.Dominant() == types.StateDead {
		t.Error("Property 6 violated: jitter alone must not indicate death")
	}
}

// Property 9: Conflicting evidence widens belief
// More conflict → more uncertainty, not resolution.
func TestProperty9_ConflictWidensBelief(t *testing.T) {
	source, target := testNodes()

	// Setup 1: Only alive evidence
	es1 := evidence.NewEvidenceSet()
	es1.Add(evidence.NewDirectResponse(styxtime.LogicalTimestamp(1), 50, source, target))
	belief1 := es1.ComputeBelief(styxtime.LogicalTimestamp(1))

	// Setup 2: Alive + Dead evidence (conflict)
	es2 := evidence.NewEvidenceSet()
	es2.Add(evidence.NewDirectResponse(styxtime.LogicalTimestamp(1), 50, source, target))
	es2.Add(evidence.NewTimeout(styxtime.LogicalTimestamp(2), 100, 1000, source, target))
	belief2 := es2.ComputeBelief(styxtime.LogicalTimestamp(2))

	// Conflicting evidence should result in MORE uncertainty
	if belief2.Unknown().Value() < belief1.Unknown().Value() {
		t.Errorf("Property 9 violated: conflict should increase uncertainty (was %f, now %f)",
			belief1.Unknown().Value(), belief2.Unknown().Value())
	}
}

// Property 15: Silence ≠ death
// Timeouts alone cannot trigger death.
func TestProperty15_SilenceNotDeath(t *testing.T) {
	source, target := testNodes()
	es := evidence.NewEvidenceSet()

	// Add multiple timeouts
	for i := uint64(1); i <= 10; i++ {
		timeout := evidence.NewTimeout(styxtime.LogicalTimestamp(i), 100, 10000, source, target)
		es.Add(timeout)
	}

	belief := es.ComputeBelief(styxtime.LogicalTimestamp(10))

	// Even with many timeouts, STYX must NOT declare certain death
	if belief.IsCertainDead() {
		t.Error("Property 15 violated: timeouts alone must never trigger certain death")
	}

	// Death confidence should increase but never reach certainty threshold
	if belief.Dead().Value() >= types.CertaintyThreshold {
		t.Errorf("Property 15 violated: dead confidence %f >= threshold %f from timeouts alone",
			belief.Dead().Value(), types.CertaintyThreshold)
	}
}

func TestDirectResponseIncreasesAlive(t *testing.T) {
	source, target := testNodes()
	es := evidence.NewEvidenceSet()

	ts := styxtime.LogicalTimestamp(1)
	es.Add(evidence.NewDirectResponse(ts, 50, source, target))

	belief := es.ComputeBelief(ts)

	if belief.Alive().Value() <= belief.Dead().Value() {
		t.Error("Direct response should increase alive confidence over dead")
	}
}

func TestTimeoutIncreasesDeadSlightly(t *testing.T) {
	source, target := testNodes()
	es := evidence.NewEvidenceSet()

	ts := styxtime.LogicalTimestamp(1)
	es.Add(evidence.NewTimeout(ts, 100, 5000, source, target))

	belief := es.ComputeBelief(ts)

	if belief.Dead().Value() <= 0 {
		t.Error("Timeout should add some dead confidence")
	}

	// But it should still be uncertain overall
	if belief.Dominant() != types.StateUnknown && belief.Dominant() != types.StateDead {
		t.Logf("Single timeout yields: %s", belief)
	}
}
