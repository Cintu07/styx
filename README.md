# STYX

truthful membership for distributed systems.

---

## what it is

styx is a distributed membership system that refuses to lie.

most systems answer "is this node alive" with timeouts, heartbeats, or gossip. all of those lie under load, partitions, and reality.

styx does not.

- if styx is unsure, it says unknown
- if styx declares death, its irreversible
- if styx cant answer honestly, it refuses to answer

## what it is not

- not a monitoring tool
- not a heartbeat service
- not raft, paxos, or quorum based
- not fast
- not convenient

styx optimizes for truth, not usability.

## how it works

instead of returning `isAlive = true/false`, styx returns a belief distribution:

```json
{
  "alive_confidence": 0.61,
  "dead_confidence": 0.19,
  "unknown": 0.20
}
```

the three values always sum to 1.0.

## quick start

```bash
# run server
go run cmd/styx-server/main.go

# query a node
curl "http://localhost:8080/query?target=42"

# submit witness report  
curl -X POST http://localhost:8080/report \
  -d '{"witness":10,"target":42,"alive":0.8,"dead":0.1,"unknown":0.1}'
```

## packages

| package | purpose |
|---------|---------|
| types | nodeid, confidence, belief |
| time | logical timestamps |
| evidence | evidence types and aggregation |
| observer | single observer with jitter tracking |
| witness | multi witness with trust decay |
| finality | irreversible death declaration |
| partition | network split detection |
| oracle | main api |
| api | http server |

## properties

| property | description |
|----------|-------------|
| P1 | identity uniqueness |
| P3 | restart does not equal resurrection |
| P6 | load does not equal failure |
| P7 | belief is never binary |
| P9 | conflict widens belief |
| P13 | false death forbidden |
| P14 | finality irreversible |
| P15 | silence does not equal death |

## run tests

```bash
go test ./... -v
```

## docker

```bash
docker build -t styx .
docker run -p 8080:8080 styx
```

---

## license

[MIT](LICENSE)

---

made by pawan
