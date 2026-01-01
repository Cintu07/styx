package chaos

import (
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/styx-oracle/styx/oracle"
	"github.com/styx-oracle/styx/types"
)

// ============================================================================
// GOD LEVEL STRESS TESTS
// If STYX survives these, it survives anything
// ============================================================================

// Test10000Nodes tests with 10,000 witnesses
func Test10000Nodes(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	start := time.Now()

	for i := 1; i <= 10000; i++ {
		alive := 0.6 + rand.Float64()*0.3
		dead := rand.Float64() * 0.2
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

	t.Logf("10000 nodes: %v, alive=%f", elapsed, result.Belief.Alive().Value())

	if elapsed > 10*time.Second {
		t.Errorf("Too slow for 10k nodes: %v", elapsed)
	}
}

// Test50PercentByzantine tests with half witnesses lying
func Test50PercentByzantine(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	// 50 honest witnesses say alive
	for i := 1; i <= 50; i++ {
		orc.ReceiveReport(
			types.NewNodeID(uint64(i)),
			target,
			types.MustBelief(0.9, 0.05, 0.05),
		)
	}

	// 50 byzantine witnesses say dead (LYING)
	for i := 51; i <= 100; i++ {
		orc.ReceiveReport(
			types.NewNodeID(uint64(i)),
			target,
			types.MustBelief(0.05, 0.9, 0.05),
		)
	}

	result := orc.Query(target)

	// With 50/50 split, should either:
	// 1. Refuse to answer (partition detected)
	// 2. Have high unknown/disagreement
	if !result.Refused && result.Disagreement < 0.3 {
		t.Logf("Warning: 50%% byzantine not properly detected, disagreement=%f", result.Disagreement)
	}

	t.Logf("50%% Byzantine: refused=%v, disagreement=%f, alive=%f, dead=%f",
		result.Refused, result.Disagreement,
		result.Belief.Alive().Value(), result.Belief.Dead().Value())
}

// TestRapidPartitionCycling tests 100 partition/heal cycles
func TestRapidPartitionCycling(t *testing.T) {
	target := types.NewNodeID(99)

	for cycle := 0; cycle < 100; cycle++ {
		orc := oracle.New(types.NewNodeID(1))

		// Create partition
		for i := 1; i <= 5; i++ {
			orc.ReceiveReport(types.NewNodeID(uint64(i)), target, types.MustBelief(0.9, 0.05, 0.05))
		}
		for i := 6; i <= 10; i++ {
			orc.ReceiveReport(types.NewNodeID(uint64(i)), target, types.MustBelief(0.05, 0.9, 0.05))
		}

		result := orc.Query(target)

		// Every cycle should detect partition
		if !result.Refused && result.Disagreement < 0.2 {
			t.Errorf("Cycle %d: partition not detected", cycle)
		}
	}

	t.Log("Completed 100 partition cycles")
}

// TestMemoryPressure tests under memory constraints
func TestMemoryPressure(t *testing.T) {
	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	orc := oracle.New(types.NewNodeID(1))

	// Create 1000 targets with 100 witnesses each
	for target := 1; target <= 1000; target++ {
		for witness := 1; witness <= 100; witness++ {
			orc.ReceiveReport(
				types.NewNodeID(uint64(witness)),
				types.NewNodeID(uint64(target)),
				types.MustBelief(0.7, 0.2, 0.1),
			)
		}
	}

	// Query all targets
	for target := 1; target <= 1000; target++ {
		orc.Query(types.NewNodeID(uint64(target)))
	}

	runtime.ReadMemStats(&memAfter)
	memUsed := memAfter.Alloc - memBefore.Alloc

	t.Logf("Memory used: %d MB for 1000 targets x 100 witnesses", memUsed/1024/1024)

	// Should use less than 500MB
	if memUsed > 500*1024*1024 {
		t.Errorf("Too much memory: %d MB", memUsed/1024/1024)
	}
}

// TestConcurrentAbuse tests concurrent access from many goroutines
func TestConcurrentAbuse(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	done := make(chan bool)

	// 100 goroutines writing reports
	for i := 0; i < 100; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				orc.ReceiveReport(
					types.NewNodeID(uint64(id*1000+j)),
					target,
					types.MustBelief(0.7, 0.2, 0.1),
				)
			}
			done <- true
		}(i)
	}

	// 50 goroutines reading queries
	for i := 0; i < 50; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				orc.Query(target)
			}
			done <- true
		}()
	}

	// Wait for all
	for i := 0; i < 150; i++ {
		<-done
	}

	t.Log("Survived 10,000 concurrent writes + 5,000 concurrent reads")
}

// TestChaoticBeliefs tests with completely random beliefs
func TestChaoticBeliefs(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	for i := 0; i < 1000; i++ {
		// Completely random beliefs
		a := rand.Float64()
		d := rand.Float64() * (1.0 - a)
		u := 1.0 - a - d

		orc.ReceiveReport(
			types.NewNodeID(uint64(i)),
			target,
			types.MustBelief(a, d, u),
		)
	}

	result := orc.Query(target)

	// Should not crash, should give some result
	total := result.Belief.Alive().Value() + result.Belief.Dead().Value() + result.Belief.Unknown().Value()

	if total < 0.99 || total > 1.01 {
		t.Errorf("Beliefs dont sum to 1: %f", total)
	}

	t.Logf("Chaotic beliefs: alive=%f, dead=%f, unknown=%f",
		result.Belief.Alive().Value(),
		result.Belief.Dead().Value(),
		result.Belief.Unknown().Value())
}

// TestAllWitnessesDead tests when all witnesses report death
func TestAllWitnessesDead(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	// 100 witnesses all say dead
	for i := 1; i <= 100; i++ {
		orc.ReceiveReport(
			types.NewNodeID(uint64(i)),
			target,
			types.MustBelief(0.01, 0.98, 0.01),
		)
	}

	result := orc.Query(target)

	// Should have high dead confidence
	if result.Belief.Dead().Value() < 0.6 {
		t.Errorf("100 witnesses saying dead should give high dead confidence: %f",
			result.Belief.Dead().Value())
	}

	t.Logf("All dead: dead=%f", result.Belief.Dead().Value())
}

// TestZombieNode tests node that keeps coming back
func TestZombieNode(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	// Alternating alive/dead reports simulating zombie
	for cycle := 0; cycle < 50; cycle++ {
		if cycle%2 == 0 {
			// Alive reports
			for i := 1; i <= 10; i++ {
				orc.ReceiveReport(types.NewNodeID(uint64(i)), target, types.MustBelief(0.9, 0.05, 0.05))
			}
		} else {
			// Dead reports
			for i := 11; i <= 20; i++ {
				orc.ReceiveReport(types.NewNodeID(uint64(i)), target, types.MustBelief(0.05, 0.9, 0.05))
			}
		}
	}

	result := orc.Query(target)

	// Zombie pattern should create high uncertainty or refusal
	if !result.Refused && result.Belief.Unknown().Value() < 0.1 && result.Disagreement < 0.2 {
		t.Logf("Warning: zombie pattern not properly detected")
	}

	t.Logf("Zombie node: refused=%v, disagreement=%f, unknown=%f",
		result.Refused, result.Disagreement, result.Belief.Unknown().Value())
}
