package types

import (
	"errors"
	"fmt"
	"math"
)

// Confidence errors
var (
	ErrConfidenceBelowMinimum = errors.New("confidence value below minimum 0.0")
	ErrConfidenceAboveMaximum = errors.New("confidence value above maximum 1.0")
	ErrConfidenceNaN          = errors.New("confidence value cannot be NaN")
)

// ConfidenceEpsilon is the tolerance for floating-point comparisons.
const ConfidenceEpsilon = 1e-10

// Confidence represents a value in the range [0.0, 1.0].
//
// This type enforces bounds at construction time, ensuring that
// all operations on Confidence values remain within valid range.
type Confidence struct {
	value float64
}

// NewConfidence creates a new Confidence from a raw value.
// Returns an error if the value is outside [0.0, 1.0] or is NaN.
func NewConfidence(value float64) (Confidence, error) {
	if math.IsNaN(value) {
		return Confidence{}, ErrConfidenceNaN
	}
	if value < 0.0 {
		return Confidence{}, fmt.Errorf("%w: %f", ErrConfidenceBelowMinimum, value)
	}
	if value > 1.0 {
		return Confidence{}, fmt.Errorf("%w: %f", ErrConfidenceAboveMaximum, value)
	}
	return Confidence{value: value}, nil
}

// MustConfidence creates a Confidence or panics if invalid.
// Use only when you're certain the value is valid.
func MustConfidence(value float64) Confidence {
	c, err := NewConfidence(value)
	if err != nil {
		panic(err)
	}
	return c
}

// ClampedConfidence creates a Confidence, clamping to valid range.
// NaN is clamped to 0.0.
func ClampedConfidence(value float64) Confidence {
	if math.IsNaN(value) {
		return Confidence{value: 0.0}
	}
	if value < 0.0 {
		return Confidence{value: 0.0}
	}
	if value > 1.0 {
		return Confidence{value: 1.0}
	}
	return Confidence{value: value}
}

// ConfidenceZero returns minimum confidence (complete uncertainty).
func ConfidenceZero() Confidence {
	return Confidence{value: 0.0}
}

// ConfidenceOne returns maximum confidence (absolute certainty).
func ConfidenceOne() Confidence {
	return Confidence{value: 1.0}
}

// Value returns the raw confidence value.
func (c Confidence) Value() float64 {
	return c.value
}

// IsZero checks if this confidence is effectively zero.
func (c Confidence) IsZero() bool {
	return c.value < ConfidenceEpsilon
}

// IsOne checks if this confidence is effectively one.
func (c Confidence) IsOne() bool {
	return (1.0 - c.value) < ConfidenceEpsilon
}

// Equal checks if two confidences are equal within tolerance.
func (c Confidence) Equal(other Confidence) bool {
	return math.Abs(c.value-other.value) < ConfidenceEpsilon
}

// Less checks if this confidence is less than another.
func (c Confidence) Less(other Confidence) bool {
	if c.Equal(other) {
		return false
	}
	return c.value < other.value
}

// String returns a human-readable representation.
func (c Confidence) String() string {
	return fmt.Sprintf("%.2f%%", c.value*100.0)
}
