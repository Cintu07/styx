package state

import (
	"fmt"

	"github.com/styx-oracle/styx/evidence"
	styxtime "github.com/styx-oracle/styx/time"
	"github.com/styx-oracle/styx/types"
)

// ObserverState is the complete local state of a single observer node.
type ObserverState struct {
	selfID       types.NodeID
	beliefs      map[types.NodeID]*LocalBelief
	logicalClock styxtime.LogicalTimestamp
}

// NewObserverState creates a new observer state.
func NewObserverState(selfID types.NodeID) *ObserverState {
	return &ObserverState{
		selfID:       selfID,
		beliefs:      make(map[types.NodeID]*LocalBelief),
		logicalClock: styxtime.Zero(),
	}
}

// SelfID returns this observer's identity.
func (os *ObserverState) SelfID() types.NodeID {
	return os.selfID
}

// LogicalTime returns the current logical time.
func (os *ObserverState) LogicalTime() styxtime.LogicalTimestamp {
	return os.logicalClock
}

// Tick advances the logical clock.
func (os *ObserverState) Tick() styxtime.LogicalTimestamp {
	return os.logicalClock.Increment()
}

// Receive updates the logical clock based on a received message.
func (os *ObserverState) Receive(receivedTS styxtime.LogicalTimestamp) styxtime.LogicalTimestamp {
	return os.logicalClock.Update(receivedTS)
}

// RecordEvidence records evidence about a target node.
func (os *ObserverState) RecordEvidence(target types.NodeID, e evidence.Evidence) types.Belief {
	lb, ok := os.beliefs[target]
	if !ok {
		lb = NewLocalBelief(target)
		os.beliefs[target] = lb
	}
	return lb.RecordEvidence(e)
}

// Query returns the belief about a specific node.
// Returns nil if we have no information about the node.
func (os *ObserverState) Query(target types.NodeID) *BeliefQuery {
	lb, ok := os.beliefs[target]
	if !ok {
		return nil
	}
	return &BeliefQuery{
		Target:    target,
		Belief:    lb.Belief(),
		Reasoning: lb.Reasoning(),
		Observer:  os.selfID,
		QueryTime: os.logicalClock,
	}
}

// QueryOrUnknown returns a query result, defaulting to unknown if no info exists.
func (os *ObserverState) QueryOrUnknown(target types.NodeID) BeliefQuery {
	if q := os.Query(target); q != nil {
		return *q
	}
	return BeliefQuery{
		Target:    target,
		Belief:    types.UnknownBelief(),
		Reasoning: BeliefReasoning{Belief: types.UnknownBelief()},
		Observer:  os.selfID,
		QueryTime: os.logicalClock,
	}
}

// KnownNodes returns all nodes we have beliefs about.
func (os *ObserverState) KnownNodes() []types.NodeID {
	nodes := make([]types.NodeID, 0, len(os.beliefs))
	for id := range os.beliefs {
		nodes = append(nodes, id)
	}
	return nodes
}

// AliveNodes returns nodes we believe are alive.
func (os *ObserverState) AliveNodes() []types.NodeID {
	nodes := make([]types.NodeID, 0)
	for id, lb := range os.beliefs {
		if lb.Belief().Dominant() == types.StateAlive {
			nodes = append(nodes, id)
		}
	}
	return nodes
}

// DeadNodes returns nodes we believe are dead.
func (os *ObserverState) DeadNodes() []types.NodeID {
	nodes := make([]types.NodeID, 0)
	for id, lb := range os.beliefs {
		if lb.Belief().Dominant() == types.StateDead {
			nodes = append(nodes, id)
		}
	}
	return nodes
}

// UnknownNodes returns nodes whose state is unknown.
func (os *ObserverState) UnknownNodes() []types.NodeID {
	nodes := make([]types.NodeID, 0)
	for id, lb := range os.beliefs {
		if lb.Belief().Dominant() == types.StateUnknown {
			nodes = append(nodes, id)
		}
	}
	return nodes
}

// RecomputeBeliefs recomputes all beliefs at current time (for decay).
func (os *ObserverState) RecomputeBeliefs() {
	for _, lb := range os.beliefs {
		lb.RecomputeAt(os.logicalClock)
	}
}

func (os *ObserverState) String() string {
	return fmt.Sprintf("ObserverState(%s at %s, tracking %d nodes)",
		os.selfID, os.logicalClock, len(os.beliefs))
}

// BeliefQuery is the result of querying a belief.
type BeliefQuery struct {
	Target    types.NodeID
	Belief    types.Belief
	Reasoning BeliefReasoning
	Observer  types.NodeID
	QueryTime styxtime.LogicalTimestamp
}

// IsCertainAlive checks if we're certain the target is alive.
func (bq BeliefQuery) IsCertainAlive() bool {
	return bq.Belief.IsCertainAlive()
}

// IsCertainDead checks if we're certain the target is dead.
func (bq BeliefQuery) IsCertainDead() bool {
	return bq.Belief.IsCertainDead()
}

// Dominant returns the dominant state.
func (bq BeliefQuery) Dominant() types.BeliefState {
	return bq.Belief.Dominant()
}

func (bq BeliefQuery) String() string {
	return fmt.Sprintf("Query about %s by %s at %s: %s",
		bq.Target, bq.Observer, bq.QueryTime, bq.Belief)
}
