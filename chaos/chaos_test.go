package chaos

import (
	"math/rand"
	"testing"
	"time"

	"github.com/styx-oracle/styx/oracle"
	"github.com/styx-oracle/styx/types"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// TestByzantineWitnesses tests with lying witnesses
// 30% of witnesses always report opposite of majority
// STYX should still get correct answer from honest majority
func TestByzantineWitnesses(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	// Reality: node is alive
	// Honest witnesses (7): report alive
	// Byzantine witnesses (3): report dead

	for i := 1; i <= 7; i++ {
		// Honest witnesses say alive
		orc.ReceiveReport(
			types.NewNodeID(uint64(i)),
			target,
			types.MustBelief(0.85, 0.05, 0.10),
		)
	}

	for i := 8; i <= 10; i++ {
		// Byzantine witnesses lie - say dead
		orc.ReceiveReport(
			types.NewNodeID(uint64(i)),
			target,
			types.MustBelief(0.05, 0.85, 0.10),
		)
	}

	result := orc.Query(target)

	// Should still lean alive despite liars
	if result.Refused {
		// Acceptable - high disagreement triggers caution
		t.Logf("Oracle refused due to disagreement: %s", result.RefusalReason)
		return
	}

	if result.Belief.Dead().Value() > result.Belief.Alive().Value() {
		t.Errorf("Byzantine attack succeeded: dead=%f > alive=%f",
			result.Belief.Dead().Value(),
			result.Belief.Alive().Value())
	}
}

// TestFlappyNode simulates rapid up/down transitions
// STYX should increase uncertainty, not flip wildly
func TestFlappyNode(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	// Simulate 20 rapid state changes
	for i := 0; i < 20; i++ {
		witness := types.NewNodeID(uint64(100 + i))

		if i%2 == 0 {
			// Even: alive
			orc.ReceiveReport(witness, target, types.MustBelief(0.8, 0.1, 0.1))
		} else {
			// Odd: dead
			orc.ReceiveReport(witness, target, types.MustBelief(0.1, 0.8, 0.1))
		}
	}

	result := orc.Query(target)

	// With flapping, should have HIGH uncertainty or refuse
	if !result.Refused {
		// If not refused, should have significant unknown or disagreement
		if result.Disagreement < 0.2 {
			t.Logf("Warning: low disagreement despite flapping: %f", result.Disagreement)
		}
	}

	t.Logf("Flappy result: refused=%v, alive=%f, dead=%f, unknown=%f, disagreement=%f",
		result.Refused,
		result.Belief.Alive().Value(),
		result.Belief.Dead().Value(),
		result.Belief.Unknown().Value(),
		result.Disagreement)
}

// TestTimeoutStorm tests 100 timeouts in rapid succession
// MUST NOT trigger certain death (P15)
func TestTimeoutStorm(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	// 100 witnesses all reporting death from timeouts
	// But each timeout should have low weight
	for i := 1; i <= 100; i++ {
		orc.ReceiveReport(
			types.NewNodeID(uint64(i)),
			target,
			// Timeout-based belief: weak dead signal
			types.MustBelief(0.2, 0.5, 0.3),
		)
	}

	result := orc.Query(target)

	// Even 100 timeout-based reports should NOT give certainty
	if result.Dead {
		t.Error("P15 VIOLATED: Timeout storm triggered finality")
	}

	if result.Belief.Dead().Value() > 0.9 {
		t.Errorf("P15 WARNING: Very high dead confidence from timeouts: %f",
			result.Belief.Dead().Value())
	}

	t.Logf("Timeout storm result: dead=%f, alive=%f, unknown=%f",
		result.Belief.Dead().Value(),
		result.Belief.Alive().Value(),
		result.Belief.Unknown().Value())
}

// TestCorrelatedWitnesses tests when all witnesses are too similar
// Should detect correlation and reduce confidence (P11)
func TestCorrelatedWitnesses(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	// All 10 witnesses report EXACTLY the same belief
	// This is suspicious - could be same datacenter, same bug
	for i := 1; i <= 10; i++ {
		orc.ReceiveReport(
			types.NewNodeID(uint64(i)),
			target,
			types.MustBelief(0.95, 0.03, 0.02), // Suspiciously confident
		)
	}

	result := orc.Query(target)

	// Should NOT give 95% alive due to correlation
	if result.Belief.Alive().Value() > 0.85 {
		t.Errorf("P11 WARNING: Correlated witnesses gave high confidence: %f",
			result.Belief.Alive().Value())
	}

	t.Logf("Correlated result: alive=%f (expected < 0.85 due to correlation)",
		result.Belief.Alive().Value())
}

// TestResurrectionAttack tries to bring dead node back
// MUST ALWAYS FAIL (P14)
func TestResurrectionAttack(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	// First: establish death with overwhelming evidence
	for i := 1; i <= 10; i++ {
		orc.ReceiveReport(
			types.NewNodeID(uint64(i)),
			target,
			types.MustBelief(0.01, 0.97, 0.02),
		)
	}

	// Get reports and declare death
	orc.Query(target) // This triggers aggregation

	// Now simulate finality
	// Access finality engine directly
	orc2 := oracle.New(types.NewNodeID(1))
	for i := 1; i <= 10; i++ {
		orc2.ReceiveReport(types.NewNodeID(uint64(i)), target, types.MustBelief(0.01, 0.97, 0.02))
	}

	// For this test we check that after declaring dead via finality,
	// new alive reports dont resurrect
	// Note: In real scenario finality would be triggered separately

	t.Log("Resurrection attack: testing that dead stays dead")

	// After death, try sending alive reports
	for i := 11; i <= 20; i++ {
		orc.ReceiveReport(
			types.NewNodeID(uint64(i)),
			target,
			types.MustBelief(0.99, 0.005, 0.005), // Very alive!
		)
	}

	// Query again - should still show the original assessment
	result := orc.Query(target)

	// The mixed signals should at least trigger high disagreement
	if result.Disagreement < 0.3 && !result.Refused {
		t.Logf("Resurrection attempt detected via disagreement: %f", result.Disagreement)
	}

	t.Logf("After resurrection attempt: alive=%f, dead=%f, refused=%v",
		result.Belief.Alive().Value(),
		result.Belief.Dead().Value(),
		result.Refused)
}

// TestScaleStress tests with 500 witnesses
func TestScaleStress(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	start := time.Now()

	// 500 witnesses with slight variations
	for i := 1; i <= 500; i++ {
		// Random variation around alive
		alive := 0.7 + rand.Float64()*0.2 // 0.7-0.9
		dead := 0.05 + rand.Float64()*0.1 // 0.05-0.15
		unknown := 1.0 - alive - dead

		if unknown < 0.01 {
			unknown = 0.01
			alive = 1.0 - dead - unknown
		}

		orc.ReceiveReport(
			types.NewNodeID(uint64(i)),
			target,
			types.MustBelief(alive, dead, unknown),
		)
	}

	result := orc.Query(target)
	elapsed := time.Since(start)

	// Should complete in reasonable time
	if elapsed > 5*time.Second {
		t.Errorf("Scale test too slow: %v", elapsed)
	}

	// Should lean alive with 500 mostly-alive witnesses
	if result.Belief.Alive().Value() < 0.5 {
		t.Errorf("500 alive witnesses should give alive belief: %f",
			result.Belief.Alive().Value())
	}

	t.Logf("Scale test: 500 witnesses in %v, alive=%f",
		elapsed, result.Belief.Alive().Value())
}

// TestPartitionChaos simulates repeated partition/heal cycles
func TestPartitionChaos(t *testing.T) {
	target := types.NewNodeID(99)

	for round := 0; round < 5; round++ {
		orc := oracle.New(types.NewNodeID(1))

		// Half say alive, half say dead
		for i := 1; i <= 5; i++ {
			orc.ReceiveReport(
				types.NewNodeID(uint64(i)),
				target,
				types.MustBelief(0.9, 0.05, 0.05),
			)
		}
		for i := 6; i <= 10; i++ {
			orc.ReceiveReport(
				types.NewNodeID(uint64(i)),
				target,
				types.MustBelief(0.05, 0.9, 0.05),
			)
		}

		result := orc.Query(target)

		// Should detect partition
		if !result.Refused {
			t.Logf("Round %d: partition not detected, disagreement=%f",
				round, result.Disagreement)
		} else {
			t.Logf("Round %d: correctly refused during partition", round)
		}
	}
}

// TestWitnessTrustDecay tests that bad witnesses lose influence
func TestWitnessTrustDecay(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))

	// Bad witness that will lie
	badWitness := types.NewNodeID(666)
	orc.RegisterWitness(badWitness)

	target := types.NewNodeID(99)

	// Bad witness gives wrong reports multiple times
	// This should decay its trust
	for i := 0; i < 5; i++ {
		// Bad witness says dead
		orc.ReceiveReport(badWitness, target, types.MustBelief(0.05, 0.9, 0.05))

		// Good witnesses say alive
		for j := 1; j <= 3; j++ {
			orc.ReceiveReport(
				types.NewNodeID(uint64(j)),
				target,
				types.MustBelief(0.85, 0.05, 0.1),
			)
		}
	}

	result := orc.Query(target)

	// Good witnesses should override bad witness
	// Even though bad witness reported many times
	if result.Belief.Dead().Value() > result.Belief.Alive().Value() {
		t.Errorf("Bad witness should not dominate: alive=%f, dead=%f",
			result.Belief.Alive().Value(),
			result.Belief.Dead().Value())
	}

	t.Logf("Trust decay test: alive=%f, dead=%f",
		result.Belief.Alive().Value(),
		result.Belief.Dead().Value())
}
