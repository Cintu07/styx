# STYX

**Truthful membership for distributed systems.**

Not logs. Not heartbeats. Not guesses.  
**Honesty.**

---

## What STYX Is

STYX is a distributed membership system that **refuses to lie**.

It answers one question:

> **Who is alive — and how sure are we?**

Most systems answer this with timeouts, heartbeats, gossip, boolean flags.  
**All of those lie** under load, partitions, and reality.

STYX does not.

- If STYX is unsure, it says `"unknown"`
- If STYX declares death, it is **irreversible**
- If STYX cannot answer honestly, it **refuses to answer**

---

## What STYX Is NOT

- ❌ Not a monitoring tool
- ❌ Not a heartbeat service
- ❌ Not Raft / Paxos / quorum-based
- ❌ Not fast
- ❌ Not convenient

STYX optimizes for **truth**, not usability.

---

## Core Principles

1. **Uncertainty is first-class**
2. **False death is worse than delayed death**
3. **Silence is better than lies**
4. **Disagreement is preserved**
5. **Time is not trusted**
6. **Death is irreversible**

---

## API Shape

```json
{
  "node": "X",
  "alive_confidence": 0.61,
  "dead_confidence": 0.19,
  "unknown": 0.20,
  "evidence": [
    "causal message observed via node-Y",
    "network instability detected",
    "observer disagreement present"
  ]
}
```

There is **no** `isAlive(node) -> true/false`.

---

## Implementation Status

### Phase 1: Foundation ✓
- [x] NodeID with generation (restart ≠ resurrection)
- [x] Confidence values with bounds enforcement
- [x] Belief distributions (alive + dead + unknown = 1)
- [x] Logical timestamps (no wall clocks)
- [x] Evidence types with weights
- [x] EvidenceSet with belief computation

### Phase 2: Single Observer (In Progress)
- [ ] Probing mechanism
- [ ] Response entropy measurement

### Phase 3-8: Future
- Witness diversity
- Finality engine
- Refusal mode
- API contract

---

## Properties (Must Hold)

| # | Property | Status |
|---|----------|--------|
| 1 | Identity uniqueness | ✓ |
| 3 | Restart ≠ resurrection | ✓ |
| 4 | No evidence → no conclusion | ✓ |
| 5 | Evidence is monotonic | ✓ |
| 6 | Load ≠ failure | ✓ |
| 7 | Belief is never binary | ✓ |
| 8 | Unknown always allowed | ✓ |
| 9 | Conflict widens belief | ✓ |
| 15 | Silence ≠ death | ✓ |
| 18 | Confidence sums to 1 | ✓ |

---

## Run Tests

```bash
go test ./... -v
```

---

## License

MIT
