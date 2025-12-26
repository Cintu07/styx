// Package observer provides single-observer probing and belief formation.
//
// A single observer can:
// - Probe target nodes
// - Measure response latency and entropy
// - Track local scheduling jitter
// - Form beliefs with uncertainty
//
// Per Property 6: Load ≠ failure.
// Per Property 15: Silence ≠ death.
package observer
