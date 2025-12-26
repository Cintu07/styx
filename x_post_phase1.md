# STYX Phase 1 — X Post Content

## Post 1: Introduction Thread 🧵

```
🩸 STYX — building a membership oracle that refuses to lie

Most distributed systems answer "is this node alive?" with:
- timeouts
- heartbeats  
- gossip
- boolean flags

All of those LIE under load, partitions, and reality.

STYX does not.

Thread 🧵 👇
```

---

## Post 2: The Core Problem

```
The lie every distributed system tells:

"We know who is alive."

No. You don't.

- Heartbeats lie (GC pause ≠ death)
- Timeouts lie (network partition ≠ death)  
- Gossip lies (correlated failures)
- Clocks lie (NTP jumps)

STYX forces systems to admit uncertainty.
```

---

## Post 3: Phase 1 Complete

```
✅ STYX Phase 1 COMPLETE

Foundation layer implemented:

→ NodeID with generation (restart ≠ resurrection)
→ Belief = probability distribution (alive/dead/unknown)
→ Logical timestamps (no wall clocks)
→ Evidence with weights + decay
→ Conflict widens uncertainty

10 properties verified ✓

[Attach screenshot of tests passing]
```

---

## Post 4: The Properties

```
Properties that MUST hold in STYX:

1. Identity uniqueness ✓
3. Restart ≠ resurrection ✓
4. No evidence → no conclusion ✓
6. Load ≠ failure ✓
7. Belief is NEVER binary ✓
9. Conflict WIDENS belief ✓
15. Silence ≠ death ✓

If ANY break, the system is invalid.
```

---

## Post 5: The API Contract

```
STYX API shape:

{
  "alive_confidence": 0.61,
  "dead_confidence": 0.19,  
  "unknown": 0.20,
  "evidence": [...]
}

There is NO:
  isAlive(node) -> true/false

If your system needs booleans, STYX won't help you.

That's intentional.
```

---

## Post 6: Link

```
Phase 1 code is live:

github.com/Cintu07/styx

Go implementation.
Property-based tests.
No lies.

Next: Phase 2 — Single Observer probing

🩸
```

---

## Manual Test to Run (for screenshot)

```bash
cd c:\Users\kolag\Desktop\styx
go test ./... -v
```

Expected output: All tests PASS

---

## Suggested Images to Attach

1. GitHub repo screenshot (captured)
2. Terminal showing `go test ./... -v` with all PASS
3. Code snippet showing Belief struct with the 3 confidence values
