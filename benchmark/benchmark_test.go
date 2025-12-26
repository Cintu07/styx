package benchmark

import (
	"math/rand"
	"testing"
	"time"

	"github.com/styx-oracle/styx/oracle"
	"github.com/styx-oracle/styx/types"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// BenchmarkSingleQuery measures single query performance
func BenchmarkSingleQuery(b *testing.B) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	// Add some witness reports
	for i := 1; i <= 10; i++ {
		orc.ReceiveReport(
			types.NewNodeID(uint64(i)),
			target,
			types.MustBelief(0.8, 0.1, 0.1),
		)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		orc.Query(target)
	}
}

// BenchmarkReceiveReport measures report ingestion
func BenchmarkReceiveReport(b *testing.B) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		orc.ReceiveReport(
			types.NewNodeID(uint64(i%1000+1)),
			target,
			types.MustBelief(0.8, 0.1, 0.1),
		)
	}
}

// BenchmarkQueryWith100Witnesses measures query with scale
func BenchmarkQueryWith100Witnesses(b *testing.B) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	for i := 1; i <= 100; i++ {
		orc.ReceiveReport(
			types.NewNodeID(uint64(i)),
			target,
			types.MustBelief(0.7+rand.Float64()*0.2, 0.05+rand.Float64()*0.1, 0.05+rand.Float64()*0.1),
		)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		orc.Query(target)
	}
}

// BenchmarkQueryWith1000Witnesses measures query at large scale
func BenchmarkQueryWith1000Witnesses(b *testing.B) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	for i := 1; i <= 1000; i++ {
		alive := 0.7 + rand.Float64()*0.2
		dead := 0.05 + rand.Float64()*0.1
		unknown := 1.0 - alive - dead
		if unknown < 0.01 {
			unknown = 0.01
			alive = 1.0 - dead - unknown
		}
		orc.ReceiveReport(
			types.NewNodeID(uint64(i)),
			target,
			types.MustBelief(alive, dead, unknown),
		)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		orc.Query(target)
	}
}

// BenchmarkPartitionDetection measures partition check overhead
func BenchmarkPartitionDetection(b *testing.B) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	// Create partition scenario
	for i := 1; i <= 5; i++ {
		orc.ReceiveReport(types.NewNodeID(uint64(i)), target, types.MustBelief(0.9, 0.05, 0.05))
	}
	for i := 6; i <= 10; i++ {
		orc.ReceiveReport(types.NewNodeID(uint64(i)), target, types.MustBelief(0.05, 0.9, 0.05))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		orc.Query(target)
	}
}

// BenchmarkBeliefCreation measures core type creation
func BenchmarkBeliefCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		types.MustBelief(0.8, 0.1, 0.1)
	}
}

// BenchmarkParallelQueries measures concurrent access
func BenchmarkParallelQueries(b *testing.B) {
	orc := oracle.New(types.NewNodeID(1))
	target := types.NewNodeID(99)

	for i := 1; i <= 50; i++ {
		orc.ReceiveReport(
			types.NewNodeID(uint64(i)),
			target,
			types.MustBelief(0.8, 0.1, 0.1),
		)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			orc.Query(target)
		}
	})
}
