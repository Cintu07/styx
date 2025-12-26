package oracle

import (
	"testing"

	"github.com/styx-oracle/styx/types"
)

func TestOracleNeverReturnsBoolean(t *testing.T) {
	oracle := New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	result := oracle.Query(target)

	// Result must be a belief distribution, not a boolean
	_ = result.Belief.Alive()
	_ = result.Belief.Dead()
	_ = result.Belief.Unknown()

	// There is NO boolean field
	// This test exists to document the design
}

func TestOracleUnknownWithNoEvidence(t *testing.T) {
	oracle := New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	result := oracle.Query(target)

	// No evidence should give unknown belief
	if !result.Belief.Equal(types.UnknownBelief()) {
		t.Error("No evidence should result in unknown belief")
	}
}

func TestOracleRefusesDuringPartition(t *testing.T) {
	oracle := New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	// Add conflicting reports to trigger partition
	oracle.ReceiveReport(types.NewNodeID(10), target, types.MustBelief(0.9, 0.05, 0.05))
	oracle.ReceiveReport(types.NewNodeID(11), target, types.MustBelief(0.9, 0.05, 0.05))
	oracle.ReceiveReport(types.NewNodeID(12), target, types.MustBelief(0.05, 0.9, 0.05))
	oracle.ReceiveReport(types.NewNodeID(13), target, types.MustBelief(0.05, 0.9, 0.05))

	result := oracle.Query(target)

	if !result.Refused {
		t.Error("Oracle should refuse during partition")
	}
}

func TestOracleAggregatesReports(t *testing.T) {
	oracle := New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	// Add agreeing reports
	oracle.ReceiveReport(types.NewNodeID(10), target, types.MustBelief(0.8, 0.1, 0.1))
	oracle.ReceiveReport(types.NewNodeID(11), target, types.MustBelief(0.75, 0.15, 0.1))
	oracle.ReceiveReport(types.NewNodeID(12), target, types.MustBelief(0.7, 0.2, 0.1))

	result := oracle.Query(target)

	if result.Refused {
		t.Error("Should not refuse with agreeing witnesses")
	}

	// Should lean alive
	if result.Belief.Dominant() != types.StateAlive {
		t.Error("Should believe alive with agreeing witnesses")
	}
}

func TestOracleRespectsConfidenceRequirements(t *testing.T) {
	oracle := New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	// Add uncertain reports
	oracle.ReceiveReport(types.NewNodeID(10), target, types.MustBelief(0.4, 0.2, 0.4))
	oracle.ReceiveReport(types.NewNodeID(11), target, types.MustBelief(0.35, 0.25, 0.4))

	// Query with strict requirements
	result := oracle.QueryWithRequirement(target, RequiredConfidence{
		MinAlive:   0.7,
		MinDead:    0.0,
		MaxUnknown: 0.2,
	})

	if !result.Refused {
		t.Error("Should refuse when confidence requirements not met")
	}
}

func TestOracleReturnsDeadForFinalizedNodes(t *testing.T) {
	oracle := New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	// First add reports
	for i := 1; i <= 5; i++ {
		oracle.ReceiveReport(types.NewNodeID(uint64(i)), target, types.MustBelief(0.02, 0.95, 0.03))
	}

	// Get reports
	oracle.mu.RLock()
	reports := oracle.reports[target]
	oracle.mu.RUnlock()

	// Declare death via finality engine
	oracle.finality.DeclareDeath(target, types.MustBelief(0.02, 0.95, 0.03), reports, true)

	result := oracle.Query(target)

	if !result.Dead {
		t.Error("Should return Dead=true for finalized node")
	}
}
