package types_test

import (
	"testing"

	"github.com/styx-oracle/styx/types"
)

// Property 1: Identity uniqueness
// Two different NodeIDs must not be equal unless explicitly created the same.
func TestProperty1_IdentityUniqueness(t *testing.T) {
	id1 := types.NewNodeID(1)
	id2 := types.NewNodeID(2)

	if id1.Equal(id2) {
		t.Error("Property 1 violated: different base IDs should not be equal")
	}

	// Same base, different generation
	id3 := types.WithGeneration(1, 0)
	id4 := types.WithGeneration(1, 1)

	if id3.Equal(id4) {
		t.Error("Property 1 violated: different generations should not be equal")
	}
}

// Property 3: Restart ≠ resurrection
// A restarted process must appear as a new node (via Rebirth).
func TestProperty3_RestartNotResurrection(t *testing.T) {
	original := types.NewNodeID(42)
	restarted := original.Rebirth()

	if original.Equal(restarted) {
		t.Error("Property 3 violated: reborn node must have different identity")
	}

	if !restarted.IsRebirthOf(original) {
		t.Error("Property 3 violated: reborn node must trace back to original")
	}

	if restarted.Generation != original.Generation+1 {
		t.Error("Property 3 violated: generation must increment on rebirth")
	}
}

// Property 7: Belief is never binary
// Confidence values must never be exactly 0 or 1 without explicit construction.
func TestProperty7_BeliefNeverBinary(t *testing.T) {
	// UnknownBelief is explicitly constructed, so 0/0/1 is allowed
	unknown := types.UnknownBelief()
	if !unknown.Unknown().IsOne() {
		t.Error("UnknownBelief should have unknown=1")
	}

	// Regular beliefs should have constrained values
	belief, err := types.NewBelief(0.5, 0.3, 0.2)
	if err != nil {
		t.Fatalf("Failed to create belief: %v", err)
	}

	// None of the regular values should be exactly 0 or 1
	if belief.Alive().IsOne() || belief.Dead().IsOne() {
		t.Error("Property 7 violated: regular belief should not have certainty=1")
	}
}

// Property 8: Unknown is always allowed
// The system must never force unknown to zero.
func TestProperty8_UnknownAlwaysAllowed(t *testing.T) {
	// Try to create belief with zero unknown
	belief, err := types.NewBelief(0.5, 0.5, 0.0)
	if err != nil {
		t.Fatalf("Should allow zero unknown in construction: %v", err)
	}

	// But this is explicitly allowed - the property is about the system
	// never FORCING zero. Manual construction is allowed.
	if belief.Unknown().Value() != 0.0 {
		t.Error("Explicit zero unknown should be accepted")
	}

	// UnknownBelief must always be available
	unknown := types.UnknownBelief()
	if !unknown.Unknown().IsOne() {
		t.Error("UnknownBelief must have full uncertainty")
	}
}

// Property 18: Confidence sums to 1
func TestProperty18_ConfidenceSumsToOne(t *testing.T) {
	// Valid belief
	belief, err := types.NewBelief(0.5, 0.3, 0.2)
	if err != nil {
		t.Fatalf("Failed to create belief: %v", err)
	}

	sum := belief.Alive().Value() + belief.Dead().Value() + belief.Unknown().Value()
	if diff := sum - 1.0; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("Property 18 violated: sum=%f, expected 1.0", sum)
	}

	// Invalid belief should be rejected
	_, err = types.NewBelief(0.5, 0.5, 0.5)
	if err == nil {
		t.Error("Property 18 violated: sum != 1.0 should be rejected")
	}
}

func TestConfidenceBounds(t *testing.T) {
	// Valid values
	_, err := types.NewConfidence(0.0)
	if err != nil {
		t.Errorf("0.0 should be valid: %v", err)
	}

	_, err = types.NewConfidence(1.0)
	if err != nil {
		t.Errorf("1.0 should be valid: %v", err)
	}

	_, err = types.NewConfidence(0.5)
	if err != nil {
		t.Errorf("0.5 should be valid: %v", err)
	}

	// Invalid values
	_, err = types.NewConfidence(-0.1)
	if err == nil {
		t.Error("Negative values should be rejected")
	}

	_, err = types.NewConfidence(1.1)
	if err == nil {
		t.Error("Values > 1.0 should be rejected")
	}
}

func TestBeliefDominant(t *testing.T) {
	tests := []struct {
		name     string
		alive    float64
		dead     float64
		unknown  float64
		expected types.BeliefState
	}{
		{"mostly alive", 0.7, 0.1, 0.2, types.StateAlive},
		{"mostly dead", 0.1, 0.7, 0.2, types.StateDead},
		{"ambiguous", 0.35, 0.35, 0.3, types.StateUnknown},
		{"pure unknown", 0.0, 0.0, 1.0, types.StateUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			belief, err := types.NewBelief(tt.alive, tt.dead, tt.unknown)
			if err != nil {
				t.Fatalf("Failed to create belief: %v", err)
			}
			if belief.Dominant() != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, belief.Dominant())
			}
		})
	}
}
