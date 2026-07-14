package bolts

import (
	"context"
	"math"
	"testing"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

func TestSeededWeightedAStarRequiresTransferredFrontier(t *testing.T) {
	g := core.NewAdjacencyGraph(3, false)
	_ = g.AddEdge(0, 1, 1)
	_ = g.AddEdge(1, 2, 1)
	got := (SeededWeightedAStar{Weight: 1.1, RequireSeed: true}).Solve(context.Background(), g, core.RouteRequest{Source: 0, Target: 2, Workers: 1}, core.WorkBudget{}, bearing.NullObserver{})
	if got.Found {
		t.Fatalf("local continuation must not restart from source without a transferred frontier: %+v", got)
	}
	if got.TotalWork() != 0 {
		t.Fatalf("missing seed must not consume search work, got %d", got.TotalWork())
	}
}

func TestSeededWeightedAStarUsesFrontierAndReportsReuse(t *testing.T) {
	g := core.NewAdjacencyGraph(4, false)
	_ = g.AddEdge(0, 1, 1)
	_ = g.AddEdge(1, 2, 1)
	_ = g.AddEdge(2, 3, 1)
	seed := SeedState{
		Dist:     []float64{0, 1, 2, math.Inf(1)},
		Prev:     []core.NodeID{0, 0, 1, 0},
		HasPrev:  []bool{false, true, true, false},
		Frontier: []core.NodeID{2},
	}
	got := (SeededWeightedAStar{Weight: 1.0, Seed: seed, RequireSeed: true}).Solve(context.Background(), g, core.RouteRequest{Source: 0, Target: 3, Workers: 1}, core.WorkBudget{}, bearing.NullObserver{})
	if !got.Found || got.Distance != 3 {
		t.Fatalf("seeded continuation failed: %+v", got)
	}
	if handoffTelemetry, ok := got.Telemetry["seed_expanded_count"].(uint64); !ok || handoffTelemetry == 0 {
		t.Fatalf("seed expansion was not reported: %+v", got.Telemetry)
	}
}
