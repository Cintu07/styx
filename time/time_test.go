package time_test

import (
	"testing"

	styxtime "github.com/styx-oracle/styx/time"
)

func TestLogicalTimeMonotonic(t *testing.T) {
	ts := styxtime.Zero()

	t1 := ts.Increment()
	t2 := ts.Increment()
	t3 := ts.Increment()

	if !t1.IsBefore(t2) || !t2.IsBefore(t3) {
		t.Error("Logical time must be monotonically increasing")
	}
}

func TestLogicalTimeUpdate(t *testing.T) {
	local := styxtime.LogicalTimestamp(5)
	received := styxtime.LogicalTimestamp(10)

	updated := local.Update(received)

	// Should be max(5, 10) + 1 = 11
	if updated.Value() != 11 {
		t.Errorf("Expected 11, got %d", updated.Value())
	}
}

func TestLogicalTimeUpdateWithOlderMessage(t *testing.T) {
	local := styxtime.LogicalTimestamp(10)
	received := styxtime.LogicalTimestamp(5)

	updated := local.Update(received)

	// Should be max(10, 5) + 1 = 11
	if updated.Value() != 11 {
		t.Errorf("Expected 11, got %d", updated.Value())
	}
}

func TestAgeSince(t *testing.T) {
	event := styxtime.LogicalTimestamp(5)
	now := styxtime.LogicalTimestamp(10)

	age := event.AgeSince(now)
	if age != 5 {
		t.Errorf("Expected age 5, got %d", age)
	}
}

func TestAgeSinceFutureEvent(t *testing.T) {
	future := styxtime.LogicalTimestamp(10)
	now := styxtime.LogicalTimestamp(5)

	age := future.AgeSince(now)
	if age != 0 {
		t.Errorf("Future events should have age 0, got %d", age)
	}
}
