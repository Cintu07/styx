package witness

import (
	"testing"

	"github.com/styx-oracle/styx/types"
)

// P10: Disagreement is preserved
// Divergent witnesses must not be collapsed
func TestProperty10_DisagreementPreserved(t *testing.T) {
	reg := NewRegistry()
	reg.Register(types.NewNodeID(1))
	reg.Register(types.NewNodeID(2))

	agg := NewAggregator(reg)

	// Two witnesses with opposite beliefs
	reports := []WitnessReport{
		{
			Witness: types.NewNodeID(1),
			Belief:  types.MustBelief(0.8, 0.1, 0.1), // thinks alive
		},
		{
			Witness: types.NewNodeID(2),
			Belief:  types.MustBelief(0.1, 0.8, 0.1), // thinks dead
		},
	}

	result := agg.Aggregate(reports)

	// P10: Disagreement must be tracked not hidden
	if result.Disagreement < 0.3 {
		t.Errorf("P10 violated: disagreement should be high, got %f", result.Disagreement)
	}

	// Result should have high unknown due to disagreement
	if result.Belief.Unknown().Value() < 0.2 {
		t.Errorf("P10: disagreement should increase unknown, got %f", result.Belief.Unknown().Value())
	}
}

// P11: Correlated witnesses weaken confidence
// Identical evidence sources reduce weight
func TestProperty11_CorrelatedWitnessesWeakenConfidence(t *testing.T) {
	reg := NewRegistry()
	for i := 1; i <= 5; i++ {
		reg.Register(types.NewNodeID(uint64(i)))
	}

	agg := NewAggregator(reg)

	// All witnesses say exactly the same thing (too correlated)
	reports := []WitnessReport{}
	for i := 1; i <= 5; i++ {
		reports = append(reports, WitnessReport{
			Witness: types.NewNodeID(uint64(i)),
			Belief:  types.MustBelief(0.9, 0.05, 0.05),
		})
	}

	result := agg.Aggregate(reports)

	// P11: Correlated witnesses should NOT give high confidence
	// Even though all say 90% alive, we should be skeptical
	if result.Belief.Alive().Value() > 0.8 {
		t.Errorf("P11 violated: correlated witnesses should reduce confidence, got alive=%f",
			result.Belief.Alive().Value())
	}
}

// P12: Witness trust decays
// Witnesses that lie lose influence over time
func TestProperty12_WitnessTrustDecays(t *testing.T) {
	reg := NewRegistry()
	badWitness := types.NewNodeID(1)
	reg.Register(badWitness)

	initialTrust := reg.GetTrust(badWitness)

	// Record multiple wrong reports
	for i := 0; i < 5; i++ {
		reg.RecordWrong(badWitness)
	}

	finalTrust := reg.GetTrust(badWitness)

	// P12: Trust must have decayed
	if finalTrust >= initialTrust {
		t.Errorf("P12 violated: trust should decay after wrong reports, was %f now %f",
			initialTrust, finalTrust)
	}

	// Trust should never hit zero (always some weight)
	if finalTrust < MinTrust {
		t.Errorf("P12: trust should not go below MinTrust %f, got %f", MinTrust, finalTrust)
	}
}

// P12: Trust recovers slowly
func TestProperty12_TrustRecovery(t *testing.T) {
	reg := NewRegistry()
	witness := types.NewNodeID(1)
	reg.Register(witness)

	// Damage trust first
	reg.RecordWrong(witness)
	reg.RecordWrong(witness)
	damagedTrust := reg.GetTrust(witness)

	// Now correct reports
	reg.RecordCorrect(witness)
	recoveredTrust := reg.GetTrust(witness)

	// Should have recovered some trust
	if recoveredTrust <= damagedTrust {
		t.Errorf("P12: trust should recover after correct reports, was %f now %f",
			damagedTrust, recoveredTrust)
	}
}

func TestAggregatorSingleWitness(t *testing.T) {
	reg := NewRegistry()
	reg.Register(types.NewNodeID(1))

	agg := NewAggregator(reg)

	reports := []WitnessReport{
		{
			Witness: types.NewNodeID(1),
			Belief:  types.MustBelief(0.7, 0.2, 0.1),
		},
	}

	result := agg.Aggregate(reports)

	// Single witness - just use their belief
	if result.Disagreement != 0 {
		t.Error("Single witness should have zero disagreement")
	}
}

func TestAggregatorEmpty(t *testing.T) {
	reg := NewRegistry()
	agg := NewAggregator(reg)

	result := agg.Aggregate([]WitnessReport{})

	// No witnesses - unknown belief
	if !result.Belief.Equal(types.UnknownBelief()) {
		t.Error("No witnesses should return unknown belief")
	}
}

func TestRegistryBasic(t *testing.T) {
	reg := NewRegistry()
	id := types.NewNodeID(42)

	reg.Register(id)
	trust := reg.GetTrust(id)

	if trust != DefaultTrust {
		t.Errorf("New witness should have default trust %f, got %f", DefaultTrust, trust)
	}

	witnesses := reg.AllWitnesses()
	if len(witnesses) != 1 {
		t.Errorf("Should have 1 witness, got %d", len(witnesses))
	}
}
