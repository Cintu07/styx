// Package oracle provides the main STYX Oracle API.
//
// The Oracle is the single entry point for querying node liveness.
// It integrates all subsystems: observer, witness, finality, partition.
//
// Key behaviors:
// - Never returns boolean is_alive
// - Returns full belief distribution
// - Can refuse to answer if uncertain
// - Tracks all evidence and reasoning
package oracle
