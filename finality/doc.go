// Package finality provides the Finality Engine for irreversible death declaration.
//
// Phase 4 Properties:
// - P13: False death is forbidden (no death without overwhelming evidence)
// - P14: Finality is irreversible (dead nodes never transition to non-dead)
// - P15: Silence alone cannot trigger finality
package finality
