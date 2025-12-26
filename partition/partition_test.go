package partition

import (
	"testing"

	"github.com/styx-oracle/styx/types"
	"github.com/styx-oracle/styx/witness"
)

// Partition detection tests
// STYX must detect when network is partitioned and not guess

func TestNoPartitionWhenAllAgree(t *testing.T) {
	detector := NewDetector()
	target := types.NewNodeID(99)

	// All witnesses agree node is alive
	reports := []witness.WitnessReport{
		{Witness: types.NewNodeID(1), Belief: types.MustBelief(0.8, 0.1, 0.1)},
		{Witness: types.NewNodeID(2), Belief: types.MustBelief(0.75, 0.15, 0.1)},
		{Witness: types.NewNodeID(3), Belief: types.MustBelief(0.85, 0.05, 0.1)},
	}

	state, split := detector.Analyze(reports, target)

	if state != NoPartition {
		t.Errorf("Should be no partition when witnesses agree, got %s", state)
	}
	if split != nil {
		t.Error("Should have no split reality")
	}
}

func TestSuspectedPartitionWithSomeDisagreement(t *testing.T) {
	detector := NewDetector()
	target := types.NewNodeID(99)

	// Some disagreement but not extreme
	reports := []witness.WitnessReport{
		{Witness: types.NewNodeID(1), Belief: types.MustBelief(0.8, 0.1, 0.1)},
		{Witness: types.NewNodeID(2), Belief: types.MustBelief(0.1, 0.8, 0.1)},
		{Witness: types.NewNodeID(3), Belief: types.MustBelief(0.7, 0.2, 0.1)},
		{Witness: types.NewNodeID(4), Belief: types.MustBelief(0.75, 0.15, 0.1)},
	}

	state, _ := detector.Analyze(reports, target)

	// 1 dead vs 3 alive = 25% disagreement, should be suspected
	if state == NoPartition {
		t.Error("Should detect some partition suspicion with disagreement")
	}
}

func TestConfirmedPartitionWithSplitReality(t *testing.T) {
	detector := NewDetector()
	target := types.NewNodeID(99)

	// Strong split - half see alive, half see dead
	reports := []witness.WitnessReport{
		{Witness: types.NewNodeID(1), Belief: types.MustBelief(0.9, 0.05, 0.05)},
		{Witness: types.NewNodeID(2), Belief: types.MustBelief(0.85, 0.1, 0.05)},
		{Witness: types.NewNodeID(3), Belief: types.MustBelief(0.05, 0.9, 0.05)},
		{Witness: types.NewNodeID(4), Belief: types.MustBelief(0.1, 0.85, 0.05)},
	}

	state, split := detector.Analyze(reports, target)

	if state != ConfirmedPartition {
		t.Errorf("Should confirm partition with 50/50 split, got %s", state)
	}

	if split == nil {
		t.Fatal("Should have split reality info")
	}

	if len(split.Groups) != 2 {
		t.Errorf("Should have 2 witness groups, got %d", len(split.Groups))
	}

	if len(split.Ambiguous) == 0 {
		t.Error("Target should be marked as ambiguous")
	}
}

func TestShouldRefuseAnswerDuringPartition(t *testing.T) {
	detector := NewDetector()
	target := types.NewNodeID(99)

	// Create partition
	reports := []witness.WitnessReport{
		{Witness: types.NewNodeID(1), Belief: types.MustBelief(0.9, 0.05, 0.05)},
		{Witness: types.NewNodeID(2), Belief: types.MustBelief(0.05, 0.9, 0.05)},
		{Witness: types.NewNodeID(3), Belief: types.MustBelief(0.9, 0.05, 0.05)},
		{Witness: types.NewNodeID(4), Belief: types.MustBelief(0.05, 0.9, 0.05)},
	}

	detector.Analyze(reports, target)

	if !detector.ShouldRefuseAnswer() {
		t.Error("Should refuse to answer during confirmed partition")
	}
}

func TestHighUnknownSuggestsPartition(t *testing.T) {
	detector := NewDetector()
	target := types.NewNodeID(99)

	// Most witnesses dont know
	reports := []witness.WitnessReport{
		{Witness: types.NewNodeID(1), Belief: types.MustBelief(0.2, 0.2, 0.6)},
		{Witness: types.NewNodeID(2), Belief: types.MustBelief(0.1, 0.3, 0.6)},
		{Witness: types.NewNodeID(3), Belief: types.MustBelief(0.3, 0.1, 0.6)},
	}

	state, _ := detector.Analyze(reports, target)

	if state == NoPartition {
		t.Error("High unknown should suggest partition issues")
	}
}
