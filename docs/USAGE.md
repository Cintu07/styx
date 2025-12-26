# STYX Usage Documentation

## Quick Start

### Run Server

```bash
go run cmd/styx-server/main.go
```

Server starts on port 8080.

### Query a Node

```bash
curl "http://localhost:8080/query?target=42"
```

Response:
```json
{
  "target": 42,
  "alive_confidence": 0.0,
  "dead_confidence": 0.0,
  "unknown": 1.0,
  "refused": false,
  "dead": false,
  "witness_count": 0,
  "disagreement": 0,
  "partition_state": "NO_PARTITION",
  "evidence": ["no witness reports available"]
}
```

### Submit Witness Report

```bash
curl -X POST http://localhost:8080/report \
  -H "Content-Type: application/json" \
  -d '{"witness": 10, "target": 42, "alive": 0.8, "dead": 0.1, "unknown": 0.1}'
```

### Register Witness

```bash
curl -X POST http://localhost:8080/witnesses \
  -H "Content-Type: application/json" \
  -d '{"id": 10}'
```

---

## API Reference

### GET /health

Health check endpoint.

Response: `{"status":"ok","service":"styx"}`

### GET /query?target=ID

Query belief about a node.

Parameters:
- `target` (required): Node ID to query

Response fields:
- `alive_confidence`: Probability node is alive [0,1]
- `dead_confidence`: Probability node is dead [0,1]
- `unknown`: Uncertainty level [0,1]
- `refused`: true if Oracle refused to answer
- `refusal_reason`: Why Oracle refused (if refused)
- `dead`: true if node declared permanently dead
- `witness_count`: Number of witness reports
- `disagreement`: How much witnesses disagree [0,1]
- `partition_state`: NO_PARTITION, SUSPECTED_PARTITION, CONFIRMED_PARTITION
- `evidence`: List of reasoning strings

### POST /report

Submit a witness report.

Body:
```json
{
  "witness": 10,
  "target": 42,
  "alive": 0.8,
  "dead": 0.1,
  "unknown": 0.1
}
```

Rules:
- `alive + dead + unknown` must equal 1.0
- All values must be in [0,1]

### POST /witnesses

Register a new witness.

Body:
```json
{
  "id": 10
}
```

---

## Integration Example

### Go Client

```go
package main

import (
    "encoding/json"
    "net/http"
    "bytes"
)

func main() {
    // Submit report
    report := map[string]interface{}{
        "witness": 10,
        "target":  42,
        "alive":   0.8,
        "dead":    0.1,
        "unknown": 0.1,
    }
    body, _ := json.Marshal(report)
    http.Post("http://localhost:8080/report", "application/json", bytes.NewReader(body))

    // Query node
    resp, _ := http.Get("http://localhost:8080/query?target=42")
    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    
    // Check if Oracle refused
    if result["refused"].(bool) {
        // Handle uncertainty
    }
}
```

### Interpreting Results

1. **No Witnesses**: `unknown = 1.0` - need more data
2. **Refused**: Oracle cant answer honestly - wait for more data or partition to heal
3. **High Disagreement**: Witnesses disagree - possible network issue
4. **Dead = true**: Node permanently dead, irreversible

---

## Running Tests

```bash
# All tests
go test ./... -v

# Chaos tests only
go test ./chaos/... -v

# Benchmarks
go test ./benchmark/... -bench=.
```

---

## Properties Guaranteed

| Property | Guarantee |
|----------|-----------|
| P6 | Load does not equal failure |
| P7 | Belief is never binary |
| P13 | False death forbidden |
| P14 | Death is irreversible |
| P15 | Silence does not equal death |
