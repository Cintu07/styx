# STYX

truthful membership for distributed systems

not logs. not heartbeats. not guesses.
honesty.

## what it is

styx is a distributed membership system that refuses to lie.

it answers one question: who is alive and how sure are we

most systems answer this with timeouts heartbeats gossip and boolean flags. all of those lie under load partitions and reality.

styx does not.

if styx is unsure it says unknown.
if styx declares death its irreversible.
if styx cant answer honestly it refuses to answer.

## what it is not

not a monitoring tool
not a heartbeat service
not raft or paxos or quorum based
not fast
not convenient

styx optimizes for truth not usability.

## core principles

uncertainty is first class
false death is worse then delayed death
silence is better then lies
disagreement is preserved
time is not trusted
death is irreversible

## api shape

```json
{
  "node": "X",
  "alive_confidence": 0.61,
  "dead_confidence": 0.19,
  "unknown": 0.20,
  "evidence": [
    "causal message observed via node Y",
    "network instability detected",
    "observer disagreement present"
  ]
}
```

there is no isAlive(node) returning true or false.

## implementation status

phase 1 foundation done
phase 2 single observer done
phase 3 to 8 coming

## properties that must hold

| num | property | status |
|-----|----------|--------|
| 1 | identity uniqueness | done |
| 3 | restart not equals resurrection | done |
| 4 | no evidence means no conclusion | done |
| 5 | evidence is monotonic | done |
| 6 | load not equals failure | done |
| 7 | belief is never binary | done |
| 8 | unknown always allowed | done |
| 9 | conflict widens belief | done |
| 15 | silence not equals death | done |
| 18 | confidence sums to 1 | done |

## run tests

```bash
go test ./... -v
```

## license

MIT
