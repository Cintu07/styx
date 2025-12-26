package finality

import (
	"errors"
	"sync"

	"github.com/styx-oracle/styx/types"
	"github.com/styx-oracle/styx/witness"
)

// Errors
var (
	ErrAlreadyDead          = errors.New("node already declared dead")
	ErrInsufficientEvidence = errors.New("insufficient evidence for death declaration")
	ErrSilenceOnly          = errors.New("cannot declare death from silence alone")
	ErrResurrection         = errors.New("cannot resurrect a dead node")
)

// Thresholds for death declaration
const (
	// MinDeadConfidence required to even consider death
	MinDeadConfidence = 0.85
	// MinWitnesses required for death declaration
	MinWitnesses = 3
	// MaxDisagreement allowed for death declaration
	MaxDisagreement = 0.2
	// MinNonTimeoutEvidence percentage required (P15: silence alone cant trigger death)
	MinNonTimeoutEvidence = 0.3
)

// DeathRecord stores finalized death info
type DeathRecord struct {
	NodeID      types.NodeID
	FinalBelief types.Belief
	Witnesses   []types.NodeID
	Reason      string
}

// Engine handles death finality decisions
// Implements P13 P14 P15
type Engine struct {
	mu       sync.RWMutex
	dead     map[types.NodeID]*DeathRecord
	registry *witness.Registry
}

// NewEngine creates a new finality engine
func NewEngine(registry *witness.Registry) *Engine {
	return &Engine{
		dead:     make(map[types.NodeID]*DeathRecord),
		registry: registry,
	}
}

// IsDead checks if a node has been declared dead
// P14: Once dead, always dead
func (e *Engine) IsDead(id types.NodeID) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, exists := e.dead[id]
	return exists
}

// GetDeathRecord returns death record if exists
func (e *Engine) GetDeathRecord(id types.NodeID) *DeathRecord {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if rec, ok := e.dead[id]; ok {
		copy := *rec
		return &copy
	}
	return nil
}

// DeclareDeath attempts to declare a node dead
// Returns error if evidence is insufficient
//
// P13: False death is forbidden - requires overwhelming evidence
// P14: Once declared, irreversible
// P15: Silence alone cannot trigger this
func (e *Engine) DeclareDeath(
	nodeID types.NodeID,
	aggregatedBelief types.Belief,
	witnessReports []witness.WitnessReport,
	hasNonTimeoutEvidence bool,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// P14: Already dead stays dead
	if _, exists := e.dead[nodeID]; exists {
		return ErrAlreadyDead
	}

	// P13: Require overwhelming dead confidence
	if aggregatedBelief.Dead().Value() < MinDeadConfidence {
		return ErrInsufficientEvidence
	}

	// P13: Require multiple witnesses
	if len(witnessReports) < MinWitnesses {
		return ErrInsufficientEvidence
	}

	// P15: Silence alone cannot trigger death
	if !hasNonTimeoutEvidence {
		return ErrSilenceOnly
	}

	// P10: Check disagreement isnt too high
	disagreement := calculateDisagreement(witnessReports)
	if disagreement > MaxDisagreement {
		return ErrInsufficientEvidence
	}

	// All checks passed - declare death
	witnesses := make([]types.NodeID, len(witnessReports))
	for i, r := range witnessReports {
		witnesses[i] = r.Witness
	}

	e.dead[nodeID] = &DeathRecord{
		NodeID:      nodeID,
		FinalBelief: aggregatedBelief,
		Witnesses:   witnesses,
		Reason:      "overwhelming evidence from multiple witnesses",
	}

	return nil
}

// AttemptResurrection tries to bring back a dead node
// P14: This must ALWAYS fail
func (e *Engine) AttemptResurrection(id types.NodeID) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if _, exists := e.dead[id]; exists {
		return ErrResurrection
	}
	return nil // wasnt dead anyway
}

// AllDead returns all dead node IDs
func (e *Engine) AllDead() []types.NodeID {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ids := make([]types.NodeID, 0, len(e.dead))
	for id := range e.dead {
		ids = append(ids, id)
	}
	return ids
}

func calculateDisagreement(reports []witness.WitnessReport) float64 {
	if len(reports) < 2 {
		return 0
	}

	var sumDead float64
	for _, r := range reports {
		sumDead += r.Belief.Dead().Value()
	}
	avgDead := sumDead / float64(len(reports))

	var variance float64
	for _, r := range reports {
		diff := r.Belief.Dead().Value() - avgDead
		variance += diff * diff
	}

	return variance / float64(len(reports))
}
