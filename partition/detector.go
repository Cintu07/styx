package partition

import (
	"sync"

	"github.com/styx-oracle/styx/types"
	"github.com/styx-oracle/styx/witness"
)

// PartitionState represents what we know about network partitions
type PartitionState int

const (
	// NoPartition detected
	NoPartition PartitionState = iota
	// SuspectedPartition based on evidence
	SuspectedPartition
	// ConfirmedPartition with split realities
	ConfirmedPartition
)

func (p PartitionState) String() string {
	switch p {
	case NoPartition:
		return "NO_PARTITION"
	case SuspectedPartition:
		return "SUSPECTED_PARTITION"
	case ConfirmedPartition:
		return "CONFIRMED_PARTITION"
	default:
		return "UNKNOWN"
	}
}

// WitnessGroup is a set of witnesses that can see each other
type WitnessGroup struct {
	Witnesses []types.NodeID
	// What this group believes about targets
	Beliefs map[types.NodeID]types.Belief
}

// SplitReality represents divergent views of the world
type SplitReality struct {
	Groups       []WitnessGroup
	Disagreement float64
	Ambiguous    []types.NodeID // nodes with conflicting status
}

// Detector detects network partitions from witness reports
type Detector struct {
	mu                    sync.RWMutex
	state                 PartitionState
	lastSplit             *SplitReality
	disagreementThreshold float64
}

// NewDetector creates a partition detector
func NewDetector() *Detector {
	return &Detector{
		state:                 NoPartition,
		disagreementThreshold: 0.4,
	}
}

// Analyze checks for partition based on witness reports
// Returns partition state and any split realities detected
func (d *Detector) Analyze(reports []witness.WitnessReport, target types.NodeID) (PartitionState, *SplitReality) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(reports) < 2 {
		d.state = NoPartition
		return NoPartition, nil
	}

	// Check for disagreement patterns
	aliveVotes := 0
	deadVotes := 0
	unknownVotes := 0

	for _, r := range reports {
		switch r.Belief.Dominant() {
		case types.StateAlive:
			aliveVotes++
		case types.StateDead:
			deadVotes++
		case types.StateUnknown:
			unknownVotes++
		}
	}

	total := len(reports)

	// If witnesses strongly disagree, suspect partition
	if aliveVotes > 0 && deadVotes > 0 {
		disagreement := float64(min(aliveVotes, deadVotes)) / float64(total)

		if disagreement > d.disagreementThreshold {
			// Confirmed split - some see alive, some see dead
			d.state = ConfirmedPartition

			split := &SplitReality{
				Disagreement: disagreement,
				Ambiguous:    []types.NodeID{target},
			}

			// Create groups
			aliveGroup := WitnessGroup{
				Witnesses: make([]types.NodeID, 0),
				Beliefs:   make(map[types.NodeID]types.Belief),
			}
			deadGroup := WitnessGroup{
				Witnesses: make([]types.NodeID, 0),
				Beliefs:   make(map[types.NodeID]types.Belief),
			}

			for _, r := range reports {
				if r.Belief.Dominant() == types.StateAlive {
					aliveGroup.Witnesses = append(aliveGroup.Witnesses, r.Witness)
					aliveGroup.Beliefs[target] = r.Belief
				} else if r.Belief.Dominant() == types.StateDead {
					deadGroup.Witnesses = append(deadGroup.Witnesses, r.Witness)
					deadGroup.Beliefs[target] = r.Belief
				}
			}

			split.Groups = []WitnessGroup{aliveGroup, deadGroup}
			d.lastSplit = split

			return ConfirmedPartition, split
		}

		// Some disagreement but not extreme
		d.state = SuspectedPartition
		return SuspectedPartition, nil
	}

	// High unknown votes also suggest partition
	if float64(unknownVotes)/float64(total) > 0.5 {
		d.state = SuspectedPartition
		return SuspectedPartition, nil
	}

	d.state = NoPartition
	return NoPartition, nil
}

// GetState returns current partition state
func (d *Detector) GetState() PartitionState {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.state
}

// GetLastSplit returns the last detected split reality
func (d *Detector) GetLastSplit() *SplitReality {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.lastSplit
}

// ShouldRefuseAnswer returns true if partition makes answering dishonest
// STYX refuses to guess during partitions
func (d *Detector) ShouldRefuseAnswer() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.state == ConfirmedPartition
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
