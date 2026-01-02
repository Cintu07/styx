package economics

import (
	"sync"

	"github.com/Cintu07/styx/types"
)

// Stake represents tokens locked by a witness
type Stake struct {
	WitnessID types.NodeID
	Amount    uint64
	LockedAt  uint64 // Logical timestamp
}

// Reward represents earned rewards
type Reward struct {
	WitnessID types.NodeID
	Amount    uint64
	Reason    string
}

// Slash represents a penalty
type Slash struct {
	WitnessID types.NodeID
	Amount    uint64
	Reason    string
}

// Economics manages witness incentives
type Economics struct {
	mu       sync.RWMutex
	stakes   map[uint64]*Stake
	balances map[uint64]uint64
	rewards  []Reward
	slashes  []Slash

	// Config
	MinStake        uint64
	RewardPerReport uint64
	SlashPercent    float64
}

// NewEconomics creates economic incentive system
func NewEconomics() *Economics {
	return &Economics{
		stakes:          make(map[uint64]*Stake),
		balances:        make(map[uint64]uint64),
		rewards:         make([]Reward, 0),
		slashes:         make([]Slash, 0),
		MinStake:        100,
		RewardPerReport: 1,
		SlashPercent:    0.1, // 10% slash for bad reports
	}
}

// Deposit adds tokens to witness balance
func (e *Economics) Deposit(witnessID types.NodeID, amount uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.balances[witnessID.Base] += amount
}

// Stake locks tokens for witness participation
func (e *Economics) Stake(witnessID types.NodeID, amount uint64, timestamp uint64) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if amount < e.MinStake {
		return ErrInsufficientStake
	}

	if e.balances[witnessID.Base] < amount {
		return ErrInsufficientBalance
	}

	e.balances[witnessID.Base] -= amount
	e.stakes[witnessID.Base] = &Stake{
		WitnessID: witnessID,
		Amount:    amount,
		LockedAt:  timestamp,
	}
	return nil
}

// Unstake withdraws staked tokens (after lock period)
func (e *Economics) Unstake(witnessID types.NodeID) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	stake, exists := e.stakes[witnessID.Base]
	if !exists {
		return ErrNotStaked
	}

	// Return stake to balance
	e.balances[witnessID.Base] += stake.Amount
	delete(e.stakes, witnessID.Base)
	return nil
}

// RewardCorrectReport gives reward for accurate report
func (e *Economics) RewardCorrectReport(witnessID types.NodeID) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.balances[witnessID.Base] += e.RewardPerReport
	e.rewards = append(e.rewards, Reward{
		WitnessID: witnessID,
		Amount:    e.RewardPerReport,
		Reason:    "correct report",
	})
}

// SlashWrongReport penalizes incorrect report
func (e *Economics) SlashWrongReport(witnessID types.NodeID) {
	e.mu.Lock()
	defer e.mu.Unlock()

	stake, exists := e.stakes[witnessID.Base]
	if !exists {
		return
	}

	slashAmount := uint64(float64(stake.Amount) * e.SlashPercent)
	if slashAmount > stake.Amount {
		slashAmount = stake.Amount
	}

	stake.Amount -= slashAmount
	e.slashes = append(e.slashes, Slash{
		WitnessID: witnessID,
		Amount:    slashAmount,
		Reason:    "incorrect report",
	})
}

// GetBalance returns witness balance
func (e *Economics) GetBalance(witnessID types.NodeID) uint64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.balances[witnessID.Base]
}

// GetStake returns witness stake
func (e *Economics) GetStake(witnessID types.NodeID) uint64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if stake, exists := e.stakes[witnessID.Base]; exists {
		return stake.Amount
	}
	return 0
}

// IsStaked checks if witness has minimum stake
func (e *Economics) IsStaked(witnessID types.NodeID) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	stake, exists := e.stakes[witnessID.Base]
	return exists && stake.Amount >= e.MinStake
}

// Errors
var (
	ErrInsufficientStake   = &EconomicsError{"insufficient stake amount"}
	ErrInsufficientBalance = &EconomicsError{"insufficient balance"}
	ErrNotStaked           = &EconomicsError{"not staked"}
)

type EconomicsError struct {
	msg string
}

func (e *EconomicsError) Error() string {
	return e.msg
}
