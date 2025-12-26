package types

import "fmt"

// NodeID uniquely identifies a node in the distributed system.
//
// A NodeID consists of:
//   - A base identifier (typically derived from network address or UUID)
//   - A generation counter (incremented on each "rebirth")
//
// Once STYX declares a node dead with finality, any returning process
// MUST use a new NodeID (with incremented generation). This prevents
// zombie nodes and flapping identities.
type NodeID struct {
	// Base identifier (e.g., hash of address or UUID)
	Base uint64
	// Generation counter - incremented on each identity rebirth
	Generation uint64
}

// NewNodeID creates a new NodeID with the given base identifier.
// The generation starts at 0 for new nodes.
func NewNodeID(base uint64) NodeID {
	return NodeID{Base: base, Generation: 0}
}

// WithGeneration creates a NodeID with a specific generation.
// Used when a node rejoins after being declared dead.
func WithGeneration(base, generation uint64) NodeID {
	return NodeID{Base: base, Generation: generation}
}

// Rebirth creates a new identity for a reborn node.
// This MUST be used when a node returns after being declared dead.
// The generation counter is incremented, making this a distinct identity.
func (n NodeID) Rebirth() NodeID {
	return NodeID{
		Base:       n.Base,
		Generation: n.Generation + 1,
	}
}

// IsRebirthOf checks if this could be a rebirth of another NodeID.
// Returns true if the base matches but this generation is higher.
func (n NodeID) IsRebirthOf(other NodeID) bool {
	return n.Base == other.Base && n.Generation > other.Generation
}

// String returns a human-readable representation.
func (n NodeID) String() string {
	return fmt.Sprintf("%016x.g%d", n.Base, n.Generation)
}

// Equal checks if two NodeIDs are equal.
func (n NodeID) Equal(other NodeID) bool {
	return n.Base == other.Base && n.Generation == other.Generation
}
