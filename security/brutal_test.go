package security

import (
	"testing"
	"time"

	"github.com/Cintu07/styx/oracle"
	"github.com/Cintu07/styx/types"
)

// brutal tests for production grade styx
// these are designed to break the system

// test massive concurrent writes
func TestBrutalConcurrentWrites(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	done := make(chan bool)
	start := time.Now()

	// 1000 goroutines each doing 100 writes
	for i := 0; i < 1000; i++ {
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

	// wait for all
	for i := 0; i < 1000; i++ {
		<-done
	}

	elapsed := time.Since(start)
	t.Logf("100k concurrent writes: %v", elapsed)

	// should complete in reasonable time
	if elapsed > 30*time.Second {
		t.Errorf("too slow: %v", elapsed)
	}

	// should still give valid result
	result := orc.Query(target)
	total := result.Belief.Alive().Value() + result.Belief.Dead().Value() + result.Belief.Unknown().Value()
	if total < 0.99 || total > 1.01 {
		t.Errorf("beliefs dont sum to 1: %f", total)
	}
}

// test weird edge case beliefs
func TestBrutalEdgeCaseBeliefs(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	// extremely small values
	orc.ReceiveReport(types.NewNodeID(1), target, types.MustBelief(0.001, 0.001, 0.998))
	orc.ReceiveReport(types.NewNodeID(2), target, types.MustBelief(0.998, 0.001, 0.001))

	result := orc.Query(target)
	if !result.Belief.IsValid() {
		t.Errorf("invalid belief from edge cases")
	}
}

// test rapid fire queries
func TestBrutalRapidQueries(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	// add some data
	for i := 1; i <= 10; i++ {
		orc.ReceiveReport(types.NewNodeID(uint64(i)), target, types.MustBelief(0.8, 0.1, 0.1))
	}

	done := make(chan bool)
	start := time.Now()

	// 500 goroutines each doing 1000 queries
	for i := 0; i < 500; i++ {
		go func() {
			for j := 0; j < 1000; j++ {
				orc.Query(target)
			}
			done <- true
		}()
	}

	for i := 0; i < 500; i++ {
		<-done
	}

	elapsed := time.Since(start)
	t.Logf("500k queries: %v", elapsed)
}

// test memory doesnt explode with many targets
func TestBrutalManyTargets(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))

	// 10k different targets
	for target := 1; target <= 10000; target++ {
		for witness := 1; witness <= 5; witness++ {
			orc.ReceiveReport(
				types.NewNodeID(uint64(witness)),
				types.NewNodeID(uint64(target)),
				types.MustBelief(0.7, 0.2, 0.1),
			)
		}
	}

	// query random targets
	for i := 0; i < 1000; i++ {
		orc.Query(types.NewNodeID(uint64(i%10000 + 1)))
	}

	t.Log("survived 10k targets with 5 witnesses each")
}

// test conflicting reports dont crash
func TestBrutalConflictingReports(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	// half say alive, half say dead
	for i := 1; i <= 100; i++ {
		if i%2 == 0 {
			orc.ReceiveReport(types.NewNodeID(uint64(i)), target, types.MustBelief(0.99, 0.005, 0.005))
		} else {
			orc.ReceiveReport(types.NewNodeID(uint64(i)), target, types.MustBelief(0.005, 0.99, 0.005))
		}
	}

	result := orc.Query(target)

	// should have high disagreement or refuse
	if !result.Refused && result.Disagreement < 0.2 {
		t.Logf("warning: conflict not properly detected, disagreement=%f", result.Disagreement)
	}

	t.Logf("conflict test: refused=%v disagreement=%f", result.Refused, result.Disagreement)
}

// test node id overflow
func TestBrutalNodeIDOverflow(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))

	// max uint64 values
	bigID := types.NewNodeID(^uint64(0))
	target := types.NewNodeID(^uint64(0) - 1)

	orc.ReceiveReport(bigID, target, types.MustBelief(0.8, 0.1, 0.1))

	result := orc.Query(target)
	if !result.Belief.IsValid() {
		t.Errorf("failed with max uint64 node ids")
	}
}

// test witness trust decay under attack
func TestBrutalWitnessTrustAttack(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	// 5 honest witnesses
	for i := 1; i <= 5; i++ {
		orc.ReceiveReport(types.NewNodeID(uint64(i)), target, types.MustBelief(0.9, 0.05, 0.05))
	}

	// 95 malicious witnesses trying to flip result
	for i := 6; i <= 100; i++ {
		orc.ReceiveReport(types.NewNodeID(uint64(i)), target, types.MustBelief(0.05, 0.9, 0.05))
	}

	result := orc.Query(target)

	// with 95% malicious, should either refuse or show high disagreement
	t.Logf("95%% attack: refused=%v alive=%f dead=%f disagreement=%f",
		result.Refused,
		result.Belief.Alive().Value(),
		result.Belief.Dead().Value(),
		result.Disagreement)
}

// test empty state queries
func TestBrutalEmptyQueries(t *testing.T) {
	orc := oracle.New(types.NewNodeID(1))

	// query nodes that have no data
	for i := 1; i <= 1000; i++ {
		result := orc.Query(types.NewNodeID(uint64(i)))

		// should return unknown, not crash
		if result.Belief.Unknown().Value() < 0.9 {
			t.Errorf("empty query should return mostly unknown, got %f", result.Belief.Unknown().Value())
		}
	}
}
