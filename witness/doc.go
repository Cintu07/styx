// Package witness provides multi-observer evidence aggregation.
//
// Phase 3: Multiple independent observers report beliefs.
// Properties:
// - P10: Disagreement is preserved (divergent witnesses not collapsed)
// - P11: Correlated witnesses weaken confidence (identical sources reduce weight)
// - P12: Witness trust decays (bad witnesses lose influence)
package witness
