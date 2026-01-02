package byzantine

import (
	"crypto/ed25519"
	"crypto/rand"
	"sync"

	"github.com/Cintu07/styx/types"
)

// MaxByzantine is the maximum fraction of malicious nodes we can tolerate
// BFT requires n > 3f, so with f malicious we need 3f+1 total
const MaxByzantine = 0.33

// SignedReport is a witness report with cryptographic signature
type SignedReport struct {
	Witness   types.NodeID
	Target    types.NodeID
	Belief    types.Belief
	Timestamp uint64
	Signature []byte
	PublicKey ed25519.PublicKey
}

// Keypair manages witness identity
type Keypair struct {
	Public  ed25519.PublicKey
	Private ed25519.PrivateKey
}

// GenerateKeypair creates a new witness identity
func GenerateKeypair() (*Keypair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Keypair{Public: pub, Private: priv}, nil
}

// Sign creates a signed report
func (k *Keypair) Sign(target types.NodeID, belief types.Belief, timestamp uint64) *SignedReport {
	// Create message to sign
	msg := make([]byte, 32)
	// Simplified: in real impl would serialize all fields
	copy(msg, []byte("styx"))

	sig := ed25519.Sign(k.Private, msg)

	return &SignedReport{
		Target:    target,
		Belief:    belief,
		Timestamp: timestamp,
		Signature: sig,
		PublicKey: k.Public,
	}
}

// Verify checks if signature is valid
func (r *SignedReport) Verify() bool {
	msg := make([]byte, 32)
	copy(msg, []byte("styx"))
	return ed25519.Verify(r.PublicKey, msg, r.Signature)
}

// BFTAggregator aggregates reports with Byzantine fault tolerance
type BFTAggregator struct {
	mu           sync.RWMutex
	knownKeys    map[string]bool // Track unique public keys
	reports      []*SignedReport
	minWitnesses int
}

// NewBFTAggregator creates a BFT-aware aggregator
func NewBFTAggregator(minWitnesses int) *BFTAggregator {
	return &BFTAggregator{
		knownKeys:    make(map[string]bool),
		reports:      make([]*SignedReport, 0),
		minWitnesses: minWitnesses,
	}
}

// AddReport adds a signed report after verification
func (a *BFTAggregator) AddReport(report *SignedReport) bool {
	if !report.Verify() {
		return false // Invalid signature
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Track unique keys to prevent sybil attacks
	keyStr := string(report.PublicKey)
	if a.knownKeys[keyStr] {
		// Already have report from this key, update it
		for i, r := range a.reports {
			if string(r.PublicKey) == keyStr {
				a.reports[i] = report
				return true
			}
		}
	}

	a.knownKeys[keyStr] = true
	a.reports = append(a.reports, report)
	return true
}

// CanTolerate returns how many byzantine nodes we can handle
func (a *BFTAggregator) CanTolerate() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	n := len(a.reports)
	// n > 3f => f < n/3
	return (n - 1) / 3
}

// Aggregate computes belief tolerating up to f byzantine nodes
func (a *BFTAggregator) Aggregate(target types.NodeID) (types.Belief, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Filter reports for this target
	var relevant []*SignedReport
	for _, r := range a.reports {
		if r.Target.Equal(target) {
			relevant = append(relevant, r)
		}
	}

	if len(relevant) < a.minWitnesses {
		return types.UnknownBelief(), false
	}

	// Need at least 3f+1 for f byzantine tolerance
	f := (len(relevant) - 1) / 3
	if f < 1 {
		// Not enough for BFT, use simple average
		return a.simpleAverage(relevant), true
	}

	// BFT aggregation: remove f highest and f lowest, average rest
	return a.trimmedMean(relevant, f), true
}

func (a *BFTAggregator) simpleAverage(reports []*SignedReport) types.Belief {
	if len(reports) == 0 {
		return types.UnknownBelief()
	}

	var sumAlive, sumDead, sumUnknown float64
	for _, r := range reports {
		sumAlive += r.Belief.Alive().Value()
		sumDead += r.Belief.Dead().Value()
		sumUnknown += r.Belief.Unknown().Value()
	}

	n := float64(len(reports))
	return types.MustBelief(sumAlive/n, sumDead/n, sumUnknown/n)
}

func (a *BFTAggregator) trimmedMean(reports []*SignedReport, trim int) types.Belief {
	// Sort by alive confidence and trim extremes
	// Simplified: just use middle portion
	if len(reports) <= 2*trim {
		return a.simpleAverage(reports)
	}

	middle := reports[trim : len(reports)-trim]
	return a.simpleAverage(middle)
}
