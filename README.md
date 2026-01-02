# STYX

distributed membership that doesnt lie

---

## what is this

basically a system that tells you if nodes are alive or dead but it doesnt just say true or false like every other system does

the problem with normal health checks is they lie. if the network is slow they say the node is dead when its actually fine. this causes alot of problems

styx returns probabilities instead. like alive 61%, dead 19%, unknown 20%. if it doesnt know it says unknown instead of guessing wrong

## why i built this

was frustrated with how membership systems handle uncertainty. they pretend to know things they dont know. kept seeing cascading failures because a slow node got marked dead

so i built this. took a few weeks. still some rough edges but it works

## how to use

run the server:
```bash
go run cmd/styx-server/main.go
```

query a node:
```bash
curl "http://localhost:8080/query?target=42"
```

you get back something like:
```json
{
  "alive_confidence": 0.61,
  "dead_confidence": 0.19,
  "unknown": 0.20
}
```

submit witness report:
```bash
curl -X POST http://localhost:8080/report \
  -d '{"witness":10,"target":42,"alive":0.8,"dead":0.1,"unknown":0.1}'
```

## docker

```bash
docker pull pawan126/styx
docker run -p 8080:8080 pawan126/styx
```

or build yourself:
```bash
docker build -t styx .
docker run -p 8080:8080 styx
```

## cli

```bash
go install github.com/Cintu07/styx/cmd/styx@latest
styx query 42
styx health
```

## main ideas

- if unsure, say unknown (dont guess)
- death is permanent, cant be undone
- multiple witnesses needed for death
- partitioned? refuse to answer
- timeout alone is not enough evidence

## whats in here

| folder | what it does |
|--------|--------------|
| types | basic types like nodeid and belief |
| oracle | main api that you use |
| api | http server |
| witness | tracks witness trust |
| finality | handles death declaration |
| partition | detects network splits |
| prediction | tries to predict failures (experimental) |
| byzantine | crypto signatures for bad actors |
| economics | staking system (experimental) |
| k8s | kubernetes manifests |
| dashboard | web ui (basic) |

## run tests

```bash
go test ./... -v
```

## license

MIT - do whatever you want

---

made by pawan
