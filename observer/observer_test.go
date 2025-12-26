package observer

import (
	"testing"
	"time"

	styxtime "github.com/styx-oracle/styx/time"
	"github.com/styx-oracle/styx/types"
)

// Property 6: Load ≠ failure
// High jitter should reduce timeout evidence weight.
func TestProperty6_JitterDiscount(t *testing.T) {
	jt := NewJitterTracker(10)

	// Simulate high jitter (actual >> expected)
	for i := 0; i < 10; i++ {
		jt.RecordSample(
			100*time.Millisecond,
			500*time.Millisecond, // 5x expected = 400% jitter
		)
	}

	factor := jt.GetJitterFactor()

	if factor > 0.3 {
		t.Errorf("Property 6 violated: high jitter should reduce trust factor, got %f", factor)
	}

	if !jt.IsJittery() {
		t.Error("Property 6: should detect jittery conditions")
	}
}

// Property 6: Low jitter should NOT discount timeouts
func TestProperty6_LowJitterFullTrust(t *testing.T) {
	jt := NewJitterTracker(10)

	// Simulate low jitter (actual ≈ expected)
	for i := 0; i < 10; i++ {
		jt.RecordSample(
			100*time.Millisecond,
			105*time.Millisecond, // 5% jitter
		)
	}

	factor := jt.GetJitterFactor()

	if factor < 0.8 {
		t.Errorf("Property 6: low jitter should maintain high trust, got %f", factor)
	}
}

// Property 15: Silence ≠ death
// Timeout evidence should have capped weight.
func TestProperty15_TimeoutWeightCapped(t *testing.T) {
	source := types.NewNodeID(1)
	target := types.NewNodeID(2)
	ts := styxtime.LogicalTimestamp(1)

	// Create timeout with full jitter trust (worst case for timeout weight)
	ev := NewJitterAwareTimeout(ts, 100, 5000, 1.0, source, target)

	// Weight must be capped
	if ev.Weight > 0.3 {
		t.Errorf("Property 15 violated: timeout weight should be capped at 0.3, got %f", ev.Weight)
	}
}

// Property 15: Jitter should further reduce timeout weight
func TestProperty15_JitterReducesTimeoutWeight(t *testing.T) {
	source := types.NewNodeID(1)
	target := types.NewNodeID(2)
	ts := styxtime.LogicalTimestamp(1)

	// Timeout with low jitter trust
	ev := NewJitterAwareTimeout(ts, 100, 5000, 0.2, source, target)

	// Weight should be even lower due to jitter discount
	if ev.Weight > 0.1 {
		t.Errorf("Property 15: jittery timeout should have very low weight, got %f", ev.Weight)
	}
}

func TestEntropyConsistent(t *testing.T) {
	re := NewResponseEntropy(10)

	// Add consistent latencies
	for i := 0; i < 10; i++ {
		re.AddSample(100 * time.Millisecond)
	}

	entropy := re.Entropy()

	if entropy > 0.1 {
		t.Errorf("Consistent responses should have low entropy, got %f", entropy)
	}

	if re.ConfidenceFactor() < 0.9 {
		t.Errorf("Consistent responses should have high confidence factor")
	}
}

func TestEntropyErratic(t *testing.T) {
	re := NewResponseEntropy(10)

	// Add highly variable latencies
	latencies := []time.Duration{
		10 * time.Millisecond,
		500 * time.Millisecond,
		50 * time.Millisecond,
		800 * time.Millisecond,
		100 * time.Millisecond,
	}

	for _, lat := range latencies {
		re.AddSample(lat)
	}

	entropy := re.Entropy()

	if entropy < 0.3 {
		t.Errorf("Erratic responses should have higher entropy, got %f", entropy)
	}
}

func TestProberWithSimulatedProbes(t *testing.T) {
	observer := types.NewNodeID(1)
	target := types.NewNodeID(2)

	prober := NewProber(observer, 1*time.Second)

	// Simulate successful probe
	prober.SetProbeFunc(func(tgt types.NodeID) ProbeResult {
		return ProbeResult{
			Target:    tgt,
			Success:   true,
			Latency:   50 * time.Millisecond,
			Timestamp: styxtime.LogicalTimestamp(1),
		}
	})

	belief, err := prober.Probe(target)
	if err != nil {
		t.Fatalf("Probe failed: %v", err)
	}

	// Should have positive alive confidence
	if belief.Alive().Value() <= 0 {
		t.Error("Successful probe should increase alive confidence")
	}

	// Should NOT be certain (single probe is not enough)
	if belief.IsCertainAlive() {
		t.Error("Single probe should not create certainty")
	}
}

func TestProberWithTimeouts(t *testing.T) {
	observer := types.NewNodeID(1)
	target := types.NewNodeID(2)

	prober := NewProber(observer, 100*time.Millisecond)

	// Simulate timeout
	prober.SetProbeFunc(func(tgt types.NodeID) ProbeResult {
		time.Sleep(10 * time.Millisecond) // Simulate some delay
		return ProbeResult{
			Target:  tgt,
			Success: false,
			Latency: 110 * time.Millisecond,
		}
	})

	// Multiple timeouts
	var belief types.Belief
	for i := 0; i < 10; i++ {
		var err error
		belief, err = prober.Probe(target)
		if err != nil {
			t.Fatalf("Probe failed: %v", err)
		}
	}

	// Should have some dead confidence
	if belief.Dead().Value() <= 0 {
		t.Error("Timeouts should increase dead confidence")
	}

	// Property 15: Must NOT be certain dead from timeouts alone
	if belief.IsCertainDead() {
		t.Error("Property 15 violated: timeouts alone must NEVER trigger certain death")
	}
}
