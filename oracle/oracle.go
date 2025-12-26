package oracle

import (
	"errors"
	"sync"

	"github.com/styx-oracle/styx/finality"
	"github.com/styx-oracle/styx/partition"
	"github.com/styx-oracle/styx/types"
	"github.com/styx-oracle/styx/witness"
)

// Errors
var (
	ErrRefused = errors.New("oracle refuses to answer due to uncertainty")
	ErrDead    = errors.New("node is dead")
)

// QueryResult is the full response from the Oracle
type QueryResult struct {
	Target         types.NodeID
	Belief         types.Belief
	Refused        bool
	RefusalReason  string
	Dead           bool
	WitnessCount   int
	Disagreement   float64
	PartitionState partition.PartitionState
	Evidence       []string
}

// RequiredConfidence specifies minimum confidence for a query
type RequiredConfidence struct {
	MinAlive   float64
	MinDead    float64
	MaxUnknown float64
}

// DefaultRequirement for queries
var DefaultRequirement = RequiredConfidence{
	MinAlive:   0.0,
	MinDead:    0.0,
	MaxUnknown: 1.0, // accept any uncertainty
}

// StrictRequirement for high confidence queries
var StrictRequirement = RequiredConfidence{
	MinAlive:   0.7,
	MinDead:    0.7,
	MaxUnknown: 0.3,
}

// Oracle is the main STYX interface
type Oracle struct {
	mu         sync.RWMutex
	selfID     types.NodeID
	registry   *witness.Registry
	aggregator *witness.Aggregator
	finality   *finality.Engine
	partition  *partition.Detector
	reports    map[types.NodeID][]witness.WitnessReport
}

// New creates a new Oracle
func New(selfID types.NodeID) *Oracle {
	reg := witness.NewRegistry()
	return &Oracle{
		selfID:     selfID,
		registry:   reg,
		aggregator: witness.NewAggregator(reg),
		finality:   finality.NewEngine(reg),
		partition:  partition.NewDetector(),
		reports:    make(map[types.NodeID][]witness.WitnessReport),
	}
}

// RegisterWitness adds a trusted witness
func (o *Oracle) RegisterWitness(id types.NodeID) {
	o.registry.Register(id)
}

// ReceiveReport records a witness report
func (o *Oracle) ReceiveReport(witnessID, target types.NodeID, belief types.Belief) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.registry.Register(witnessID)
	report := witness.WitnessReport{
		Witness: witnessID,
		Target:  target,
		Belief:  belief,
	}

	if o.reports[target] == nil {
		o.reports[target] = make([]witness.WitnessReport, 0)
	}
	o.reports[target] = append(o.reports[target], report)
}

// Query asks the Oracle about a node
// This is the main API - never returns boolean
func (o *Oracle) Query(target types.NodeID) QueryResult {
	return o.QueryWithRequirement(target, DefaultRequirement)
}

// QueryWithRequirement queries with specific confidence requirements
// If requirements not met, Oracle refuses to answer
func (o *Oracle) QueryWithRequirement(target types.NodeID, req RequiredConfidence) QueryResult {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := QueryResult{
		Target: target,
	}

	// Check if already dead (finality)
	if o.finality.IsDead(target) {
		result.Dead = true
		result.Belief = types.MustBelief(0, 1, 0)
		result.Evidence = append(result.Evidence, "finality: node declared dead")
		return result
	}

	// Get reports for this target
	reports := o.reports[target]
	result.WitnessCount = len(reports)

	if len(reports) == 0 {
		// No evidence - unknown belief
		result.Belief = types.UnknownBelief()
		result.Evidence = append(result.Evidence, "no witness reports available")
		return result
	}

	// Check partition state
	pState, split := o.partition.Analyze(reports, target)
	result.PartitionState = pState

	if pState == partition.ConfirmedPartition {
		result.Refused = true
		result.RefusalReason = "network partition detected - witnesses disagree"
		result.Belief = types.UnknownBelief()
		if split != nil {
			result.Disagreement = split.Disagreement
		}
		result.Evidence = append(result.Evidence, "partition: witnesses split into groups")
		return result
	}

	// Aggregate witness reports
	aggResult := o.aggregator.Aggregate(reports)
	result.Belief = aggResult.Belief
	result.Disagreement = aggResult.Disagreement

	// Check if confidence meets requirements
	if aggResult.Belief.Alive().Value() > 0 && aggResult.Belief.Alive().Value() < req.MinAlive {
		if aggResult.Belief.Dead().Value() > 0 && aggResult.Belief.Dead().Value() < req.MinDead {
			result.Refused = true
			result.RefusalReason = "insufficient confidence to meet requirements"
			result.Evidence = append(result.Evidence, "confidence below threshold")
			return result
		}
	}

	if aggResult.Belief.Unknown().Value() > req.MaxUnknown {
		result.Refused = true
		result.RefusalReason = "uncertainty too high"
		result.Evidence = append(result.Evidence, "unknown exceeds threshold")
		return result
	}

	// Build evidence list
	result.Evidence = append(result.Evidence,
		"aggregated "+itoa(len(reports))+" witness reports",
	)
	if result.Disagreement > 0.1 {
		result.Evidence = append(result.Evidence, "some witness disagreement detected")
	}

	return result
}

// MustQuery panics if Oracle refuses or node is dead
// USE WITH CAUTION - defeats the purpose of STYX
func (o *Oracle) MustQuery(target types.NodeID) types.Belief {
	result := o.Query(target)
	if result.Refused {
		panic(ErrRefused)
	}
	if result.Dead {
		panic(ErrDead)
	}
	return result.Belief
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}
