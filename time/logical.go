// Package time provides logical time representation.
//
// STYX uses logical timestamps (Lamport-style) instead of wall clocks.
// Wall clocks lie. Causality doesn't.
package time

import "fmt"

// LogicalTimestamp is a Lamport-style logical timestamp.
//
// Logical timestamps capture the "happens-before" relationship
// between events without requiring synchronized clocks.
//
// Properties:
//   - If event A happens before event B, then ts(A) < ts(B)
//   - The converse is NOT necessarily true (concurrent events may have any order)
//   - Timestamps are monotonically increasing within a single process
//
// Why Not Wall Clocks?
// Wall clocks lie:
//   - NTP can jump backwards
//   - Virtualization causes clock drift
//   - CPU stalls make timestamps meaningless
//
// Logical time captures what we actually care about: causality.
type LogicalTimestamp uint64

// Zero returns a timestamp at zero (epoch).
func Zero() LogicalTimestamp {
	return LogicalTimestamp(0)
}

// Value returns the raw timestamp value.
func (t LogicalTimestamp) Value() uint64 {
	return uint64(t)
}

// Increment advances the timestamp and returns the new value.
// This should be called on any local event.
func (t *LogicalTimestamp) Increment() LogicalTimestamp {
	*t++
	return *t
}

// Update updates the timestamp based on a received message.
// Lamport's rule: ts = max(local_ts, received_ts) + 1
// This ensures that the timestamp of the receive event is
// greater than both the send and all prior local events.
func (t *LogicalTimestamp) Update(received LogicalTimestamp) LogicalTimestamp {
	if received > *t {
		*t = received
	}
	t.Increment()
	return *t
}

// IsBefore checks if this timestamp is causally before another.
func (t LogicalTimestamp) IsBefore(other LogicalTimestamp) bool {
	return t < other
}

// IsAfter checks if this timestamp is causally after another.
func (t LogicalTimestamp) IsAfter(other LogicalTimestamp) bool {
	return t > other
}

// AgeSince calculates the "age" of an event relative to current time.
// Returns 0 if the event is in the future.
func (t LogicalTimestamp) AgeSince(now LogicalTimestamp) uint64 {
	if now < t {
		return 0
	}
	return uint64(now - t)
}

// String returns a human-readable representation.
func (t LogicalTimestamp) String() string {
	return fmt.Sprintf("@%d", t)
}
