package types

import (
	"errors"
	"fmt"
	"math"
)

// BeliefState represents the dominant state of a belief distribution.
type BeliefState int

const (
	// StateUnknown indicates the liveness state is unknown.
	StateUnknown BeliefState = iota
	// StateAlive indicates the node is believed to be alive.
	StateAlive
	// StateDead indicates the node is believed to be dead.
	StateDead
)

func (s BeliefState) String() string {
	switch s {
	case StateAlive:
		return "ALIVE"
	case StateDead:
		return "DEAD"
	default:
		return "UNKNOWN"
	}
}

// Belief errors
var (
	ErrBeliefInvalidSum = errors.New("belief values must sum to 1.0")
)

// CertaintyThreshold is the threshold for considering a belief "certain".
// A node is considered certainly alive/dead only if the
// corresponding confidence exceeds this threshold.
const CertaintyThreshold = 0.95

// DominantMargin is the margin required for a state to be considered dominant.
const DominantMargin = 0.1

// BeliefSumEpsilon is the tolerance for belief sum validation.
const BeliefSumEpsilon = 1e-9

// Belief represents a probability distribution over node liveness.
//
// This represents the probability distribution over three mutually
// exclusive states: ALIVE, DEAD, and UNKNOWN.
//
// Invariant: alive + dead + unknown = 1.0 (within floating-point tolerance)
type Belief struct {
	alive   Confidence
	dead    Confidence
	unknown Confidence
}

// NewBelief creates a new Belief from raw confidence values.
// Returns an error if the values don't sum to 1.0 (within tolerance).
func NewBelief(alive, dead, unknown float64) (Belief, error) {
	sum := alive + dead + unknown
	if math.Abs(sum-1.0) > BeliefSumEpsilon {
		return Belief{}, fmt.Errorf("%w: got %f", ErrBeliefInvalidSum, sum)
	}

	aliveConf, err := NewConfidence(alive)
	if err != nil {
		return Belief{}, err
	}
	deadConf, err := NewConfidence(dead)
	if err != nil {
		return Belief{}, err
	}
	unknownConf, err := NewConfidence(unknown)
	if err != nil {
		return Belief{}, err
	}

	return Belief{
		alive:   aliveConf,
		dead:    deadConf,
		unknown: unknownConf,
	}, nil
}

// MustBelief creates a Belief or panics if invalid.
func MustBelief(alive, dead, unknown float64) Belief {
	b, err := NewBelief(alive, dead, unknown)
	if err != nil {
		panic(err)
	}
	return b
}

// UnknownBelief creates a belief of pure uncertainty.
// This is the initial state before any evidence is gathered.
func UnknownBelief() Belief {
	return Belief{
		alive:   ConfidenceZero(),
		dead:    ConfidenceZero(),
		unknown: ConfidenceOne(),
	}
}

// CertainlyAlive creates a belief of certain liveness.
// Use with caution — this represents absolute certainty.
func CertainlyAlive() Belief {
	return Belief{
		alive:   ConfidenceOne(),
		dead:    ConfidenceZero(),
		unknown: ConfidenceZero(),
	}
}

// CertainlyDead creates a belief of certain death.
// Use with caution — triggers irreversible death semantics.
func CertainlyDead() Belief {
	return Belief{
		alive:   ConfidenceZero(),
		dead:    ConfidenceOne(),
		unknown: ConfidenceZero(),
	}
}

// Alive returns the confidence that the node is alive.
func (b Belief) Alive() Confidence {
	return b.alive
}

// Dead returns the confidence that the node is dead.
func (b Belief) Dead() Confidence {
	return b.dead
}

// Unknown returns the confidence that the state is unknown.
func (b Belief) Unknown() Confidence {
	return b.unknown
}

// IsCertainAlive checks if the node is certainly alive.
// Returns true only if alive confidence exceeds the certainty threshold.
func (b Belief) IsCertainAlive() bool {
	return b.alive.Value() >= CertaintyThreshold
}

// IsCertainDead checks if the node is certainly dead.
// Returns true only if dead confidence exceeds the certainty threshold.
// This triggers irreversible death semantics.
func (b Belief) IsCertainDead() bool {
	return b.dead.Value() >= CertaintyThreshold
}

// Dominant returns the dominant state of the belief.
// Returns the state with the highest confidence.
// If there's no clear winner (difference < margin), returns StateUnknown.
func (b Belief) Dominant() BeliefState {
	alive := b.alive.Value()
	dead := b.dead.Value()
	unknown := b.unknown.Value()

	if alive > dead+DominantMargin && alive > unknown+DominantMargin {
		return StateAlive
	}
	if dead > alive+DominantMargin && dead > unknown+DominantMargin {
		return StateDead
	}
	return StateUnknown
}

// IsValid checks that the belief invariant holds.
// Returns true if alive + dead + unknown ≈ 1.0
func (b Belief) IsValid() bool {
	sum := b.alive.Value() + b.dead.Value() + b.unknown.Value()
	return math.Abs(sum-1.0) < BeliefSumEpsilon
}

// Equal checks if two beliefs are equal.
func (b Belief) Equal(other Belief) bool {
	return b.alive.Equal(other.alive) &&
		b.dead.Equal(other.dead) &&
		b.unknown.Equal(other.unknown)
}

// String returns a human-readable representation.
func (b Belief) String() string {
	return fmt.Sprintf("[A:%.0f%% D:%.0f%% U:%.0f%%] → %s",
		b.alive.Value()*100.0,
		b.dead.Value()*100.0,
		b.unknown.Value()*100.0,
		b.Dominant())
}
