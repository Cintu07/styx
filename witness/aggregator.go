package witness

import (
	"math"

	"github.com/styx-oracle/styx/types"
)

// WitnessReport is a belief report from a single witness
type WitnessReport struct {
	Witness types.NodeID
	Target  types.NodeID
	Belief  types.Belief
	Trust   TrustScore
}

// Aggregator combines multiple witness reports into a single belief
// Implements:
// - P10: Disagreement is preserved
// - P11: Correlated witnesses weaken confidence
type Aggregator struct {
	registry *Registry
}

// NewAggregator creates an aggregator with a witness registry
func NewAggregator(registry *Registry) *Aggregator {
	return &Aggregator{registry: registry}
}

// AggregateResult contains the combined belief and disagreement info
type AggregateResult struct {
	Belief       types.Belief
	Disagreement float64 // 0 = all agree, 1 = max disagreement
	WitnessCount int
	Reports      []WitnessReport
}

// Aggregate combines multiple witness reports
// P10: Disagreement preserved - we track it, dont hide it
// P11: Correlated witnesses (similar reports) reduce confidence
func (a *Aggregator) Aggregate(reports []WitnessReport) AggregateResult {
	if len(reports) == 0 {
		return AggregateResult{
			Belief: types.UnknownBelief(),
		}
	}

	if len(reports) == 1 {
		return AggregateResult{
			Belief:       reports[0].Belief,
			Disagreement: 0,
			WitnessCount: 1,
			Reports:      reports,
		}
	}

	// Calculate weighted average of beliefs
	var totalWeight float64
	var aliveSum, deadSum, unknownSum float64

	for _, r := range reports {
		trust := float64(a.registry.GetTrust(r.Witness))
		totalWeight += trust

		aliveSum += r.Belief.Alive().Value() * trust
		deadSum += r.Belief.Dead().Value() * trust
		unknownSum += r.Belief.Unknown().Value() * trust
	}

	if totalWeight < 0.001 {
		return AggregateResult{
			Belief:       types.UnknownBelief(),
			WitnessCount: len(reports),
			Reports:      reports,
		}
	}

	avgAlive := aliveSum / totalWeight
	avgDead := deadSum / totalWeight
	avgUnknown := unknownSum / totalWeight

	// P10: Calculate disagreement (variance across witnesses)
	disagreement := a.calculateDisagreement(reports, avgAlive, avgDead)

	// P11: Correlated witnesses reduce confidence
	// If witnesses are too similar, increase unknown
	correlation := a.detectCorrelation(reports)
	if correlation > 0.9 {
		// Too correlated - reduce confidence
		factor := 0.7
		avgAlive *= factor
		avgDead *= factor
		avgUnknown = 1.0 - avgAlive - avgDead
	}

	// P10: High disagreement increases unknown
	if disagreement > 0.3 {
		// Significant disagreement - widen uncertainty
		reduction := disagreement * 0.5
		avgAlive *= (1 - reduction)
		avgDead *= (1 - reduction)
		avgUnknown = 1.0 - avgAlive - avgDead
	}

	// Ensure valid belief
	if avgUnknown < 0.05 {
		avgUnknown = 0.05
		excess := 0.05 - (1.0 - avgAlive - avgDead)
		avgAlive -= excess / 2
		avgDead -= excess / 2
	}

	belief, err := types.NewBelief(avgAlive, avgDead, avgUnknown)
	if err != nil {
		belief = types.UnknownBelief()
	}

	return AggregateResult{
		Belief:       belief,
		Disagreement: disagreement,
		WitnessCount: len(reports),
		Reports:      reports,
	}
}

// calculateDisagreement measures variance in witness opinions
// P10: We track this, not hide it
func (a *Aggregator) calculateDisagreement(reports []WitnessReport, avgAlive, avgDead float64) float64 {
	if len(reports) < 2 {
		return 0
	}

	var variance float64
	for _, r := range reports {
		diffAlive := r.Belief.Alive().Value() - avgAlive
		diffDead := r.Belief.Dead().Value() - avgDead
		variance += diffAlive*diffAlive + diffDead*diffDead
	}
	variance /= float64(len(reports))

	// Normalize to [0,1]
	return math.Min(math.Sqrt(variance), 1.0)
}

// detectCorrelation checks if witnesses are too similar
// P11: Correlated witnesses weaken confidence
func (a *Aggregator) detectCorrelation(reports []WitnessReport) float64 {
	if len(reports) < 2 {
		return 0
	}

	// Check how similar all reports are
	first := reports[0].Belief
	var totalDiff float64

	for i := 1; i < len(reports); i++ {
		b := reports[i].Belief
		diffAlive := math.Abs(first.Alive().Value() - b.Alive().Value())
		diffDead := math.Abs(first.Dead().Value() - b.Dead().Value())
		totalDiff += diffAlive + diffDead
	}

	avgDiff := totalDiff / float64(len(reports)-1)

	// Low difference = high correlation
	return 1.0 - math.Min(avgDiff*2, 1.0)
}
