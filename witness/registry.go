package witness

import (
	"sync"

	"github.com/styx-oracle/styx/types"
)

// TrustScore represents how much we trust a witness [0,1]
type TrustScore float64

const (
	// MaxTrust is full trust in a witness
	MaxTrust TrustScore = 1.0
	// MinTrust is minimum trust (but never zero - always some weight)
	MinTrust TrustScore = 0.1
	// DefaultTrust for new witnesses
	DefaultTrust TrustScore = 0.8
	// DecayRate per incorrect report
	DecayRate = 0.1
	// RecoveryRate per correct report
	RecoveryRate = 0.05
)

// WitnessRecord tracks a single witness node
type WitnessRecord struct {
	ID             types.NodeID
	Trust          TrustScore
	CorrectReports int
	WrongReports   int
	LastReport     types.Belief
}

// Registry tracks all known witnesses and their trust levels
// Implements P12: Witness trust decays
type Registry struct {
	mu        sync.RWMutex
	witnesses map[types.NodeID]*WitnessRecord
}

// NewRegistry creates empty witness registry
func NewRegistry() *Registry {
	return &Registry{
		witnesses: make(map[types.NodeID]*WitnessRecord),
	}
}

// Register adds a new witness with default trust
func (r *Registry) Register(id types.NodeID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.witnesses[id]; !exists {
		r.witnesses[id] = &WitnessRecord{
			ID:    id,
			Trust: DefaultTrust,
		}
	}
}

// GetTrust returns trust score for a witness
func (r *Registry) GetTrust(id types.NodeID) TrustScore {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if w, ok := r.witnesses[id]; ok {
		return w.Trust
	}
	return DefaultTrust
}

// RecordCorrect marks a witness report as correct
// Trust increases slightly
func (r *Registry) RecordCorrect(id types.NodeID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	w := r.getOrCreate(id)
	w.CorrectReports++
	w.Trust += TrustScore(RecoveryRate)
	if w.Trust > MaxTrust {
		w.Trust = MaxTrust
	}
}

// RecordWrong marks a witness report as wrong
// P12: Trust decays for bad witnesses
func (r *Registry) RecordWrong(id types.NodeID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	w := r.getOrCreate(id)
	w.WrongReports++
	w.Trust -= TrustScore(DecayRate)
	if w.Trust < MinTrust {
		w.Trust = MinTrust
	}
}

// RecordReport stores the latest report from a witness
func (r *Registry) RecordReport(id types.NodeID, belief types.Belief) {
	r.mu.Lock()
	defer r.mu.Unlock()

	w := r.getOrCreate(id)
	w.LastReport = belief
}

// AllWitnesses returns all registered witness IDs
func (r *Registry) AllWitnesses() []types.NodeID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]types.NodeID, 0, len(r.witnesses))
	for id := range r.witnesses {
		ids = append(ids, id)
	}
	return ids
}

// GetRecord returns full witness record
func (r *Registry) GetRecord(id types.NodeID) *WitnessRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if w, ok := r.witnesses[id]; ok {
		copy := *w
		return &copy
	}
	return nil
}

func (r *Registry) getOrCreate(id types.NodeID) *WitnessRecord {
	if w, ok := r.witnesses[id]; ok {
		return w
	}
	w := &WitnessRecord{
		ID:    id,
		Trust: DefaultTrust,
	}
	r.witnesses[id] = w
	return w
}
