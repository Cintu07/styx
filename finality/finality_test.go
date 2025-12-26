package finality

import (
	"testing"

	"github.com/styx-oracle/styx/types"
	"github.com/styx-oracle/styx/witness"
)

func makeWitnesses(n int) (*witness.Registry, []witness.WitnessReport, []types.NodeID) {
	reg := witness.NewRegistry()
	reports := make([]witness.WitnessReport, n)
	ids := make([]types.NodeID, n)

	for i := 0; i < n; i++ {
		id := types.NewNodeID(uint64(i + 1))
		ids[i] = id
		reg.Register(id)
		reports[i] = witness.WitnessReport{
			Witness: id,
			Belief:  types.MustBelief(0.05, 0.90, 0.05), // high dead confidence
		}
	}
	return reg, reports, ids
}

// P13: False death is forbidden
// No execution path may declare death without overwhelming evidence
func TestProperty13_FalseDeathForbidden(t *testing.T) {
	reg := witness.NewRegistry()
	engine := NewEngine(reg)

	target := types.NewNodeID(99)

	// Try with low confidence
	lowBelief := types.MustBelief(0.3, 0.5, 0.2)
	err := engine.DeclareDeath(target, lowBelief, []witness.WitnessReport{}, true)

	if err == nil {
		t.Error("P13 violated: death declared with insufficient evidence")
	}
}

// P13: Require multiple witnesses
func TestProperty13_RequireMultipleWitnesses(t *testing.T) {
	reg := witness.NewRegistry()
	reg.Register(types.NewNodeID(1))
	engine := NewEngine(reg)

	target := types.NewNodeID(99)

	// High confidence but only 1 witness
	highBelief := types.MustBelief(0.05, 0.90, 0.05)
	reports := []witness.WitnessReport{
		{Witness: types.NewNodeID(1), Belief: highBelief},
	}

	err := engine.DeclareDeath(target, highBelief, reports, true)

	if err == nil {
		t.Error("P13 violated: death declared with only 1 witness")
	}
}

// P14: Finality is irreversible
// Dead nodes never transition to non-dead
func TestProperty14_FinalityIrreversible(t *testing.T) {
	reg, reports, _ := makeWitnesses(5)
	engine := NewEngine(reg)

	target := types.NewNodeID(99)
	highBelief := types.MustBelief(0.02, 0.95, 0.03)

	// Declare death with proper evidence
	err := engine.DeclareDeath(target, highBelief, reports, true)
	if err != nil {
		t.Fatalf("Failed to declare death: %v", err)
	}

	// P14: Attempt resurrection must fail
	err = engine.AttemptResurrection(target)
	if err != ErrResurrection {
		t.Error("P14 violated: resurrection should be forbidden")
	}

	// Must still be dead
	if !engine.IsDead(target) {
		t.Error("P14 violated: dead node is not dead anymore")
	}
}

// P15: Silence alone cannot trigger finality
func TestProperty15_SilenceAloneCannotTriggerFinality(t *testing.T) {
	reg, reports, _ := makeWitnesses(5)
	engine := NewEngine(reg)

	target := types.NewNodeID(99)
	highBelief := types.MustBelief(0.02, 0.95, 0.03)

	// Try to declare death with ONLY timeout evidence (silence)
	err := engine.DeclareDeath(target, highBelief, reports, false) // hasNonTimeoutEvidence = false

	if err != ErrSilenceOnly {
		t.Errorf("P15 violated: death should not be declared from silence alone, got %v", err)
	}
}

func TestDeathDeclarationSuccess(t *testing.T) {
	reg, reports, _ := makeWitnesses(5)
	engine := NewEngine(reg)

	target := types.NewNodeID(99)
	highBelief := types.MustBelief(0.02, 0.95, 0.03)

	// Proper death declaration
	err := engine.DeclareDeath(target, highBelief, reports, true)
	if err != nil {
		t.Fatalf("Should succeed with proper evidence: %v", err)
	}

	if !engine.IsDead(target) {
		t.Error("Node should be dead after declaration")
	}

	record := engine.GetDeathRecord(target)
	if record == nil {
		t.Error("Should have death record")
	}
}

func TestAlreadyDead(t *testing.T) {
	reg, reports, _ := makeWitnesses(5)
	engine := NewEngine(reg)

	target := types.NewNodeID(99)
	highBelief := types.MustBelief(0.02, 0.95, 0.03)

	// First declaration
	engine.DeclareDeath(target, highBelief, reports, true)

	// Second attempt
	err := engine.DeclareDeath(target, highBelief, reports, true)
	if err != ErrAlreadyDead {
		t.Error("Should return ErrAlreadyDead for double declaration")
	}
}
