// Package chaos provides stress tests to break STYX.
//
// These tests simulate adversarial conditions:
// - Byzantine witnesses (liars)
// - Rapid state changes (flapping nodes)
// - Network partitions
// - Timeout storms
// - Correlated failures
// - Resurrection attacks
//
// If STYX survives these, it survives production.
package chaos
