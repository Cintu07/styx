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

## packages

| package | what it does |
|---------|--------------|
| types | nodeid confidence belief |
| time | logical timestamps |
| evidence | evidence types and aggregation |
| state | local belief state machine |
| observer | single observer probing with jitter |
| witness | multi witness aggregation and trust |
| finality | irreversible death declaration |
| partition | network split detection |
| oracle | main api that ties it all together |

## api shape

```go
result := oracle.Query(targetNode)

// result contains:
// - Belief (alive/dead/unknown distribution)
// - Refused (bool - true if oracle cant answer honestly)
// - RefusalReason (string)
// - Dead (bool - true if finality declared)
// - WitnessCount
// - Disagreement
// - PartitionState
// - Evidence (list of reasons)
```

there is no isAlive(node) returning true or false.

## implementation status

all phases done:
- phase 1: foundation (types beliefs evidence)
- phase 2: single observer (jitter tracking)
- phase 3: multi witness (trust decay disagreement)
- phase 4: finality engine (irreversible death)
- phase 5: partition awareness (split reality)
- phase 6: oracle api (main interface)

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
| 10 | disagreement preserved | done |
| 11 | correlated witnesses weaken confidence | done |
| 12 | witness trust decays | done |
| 13 | false death forbidden | done |
| 14 | finality irreversible | done |
| 15 | silence not equals death | done |
| 18 | confidence sums to 1 | done |

## run tests

```bash
go test ./... -v
```

## license

MIT
