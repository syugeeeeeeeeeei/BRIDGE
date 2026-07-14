package bolts

import (
	"context"
	"math"
	"testing"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

func mismatchedScaleGraph(t *testing.T) *core.AdjacencyGraph {
	t.Helper()
	g := core.NewAdjacencyGraph(5, true)
	positions := []core.Point{{X: 0, Y: 0}, {X: 100, Y: 0}, {X: 200, Y: 0}, {X: 0, Y: 1}, {X: 200, Y: 1}}
	for i, p := range positions {
		if err := g.SetPosition(core.NodeID(i), p); err != nil {
			t.Fatal(err)
		}
	}
	// Optimal path 0->1->2 costs 3; geometrically misleading path 0->3->4->2 costs 6.
	for _, e := range []struct {
		u, v core.NodeID
		w    float64
	}{{0, 1, 1}, {1, 2, 2}, {0, 3, 2}, {3, 4, 2}, {4, 2, 2}} {
		if err := g.AddEdge(e.u, e.v, e.w); err != nil {
			t.Fatal(err)
		}
	}
	return g
}

func TestAStarMatchesDijkstraWhenGeometryAndCostScalesDiffer(t *testing.T) {
	g := mismatchedScaleGraph(t)
	req := core.RouteRequest{Source: 0, Target: 2, Workers: 1}
	d := (Dijkstra{}).Solve(context.Background(), g, req, core.WorkBudget{}, bearing.NullObserver{})
	a := (AStar{}).Solve(context.Background(), g, req, core.WorkBudget{}, bearing.NullObserver{})
	if !d.Found || !a.Found {
		t.Fatalf("expected paths: d=%+v a=%+v", d, a)
	}
	if math.Abs(a.Distance-d.Distance) > 1e-9 {
		t.Fatalf("A* distance=%v, Dijkstra=%v", a.Distance, d.Distance)
	}
}

func TestSourceOnlySeedParity(t *testing.T) {
	g := mismatchedScaleGraph(t)
	req := core.RouteRequest{Source: 0, Target: 2, Workers: 1}
	normal := (WeightedAStar{Weight: 1.12, ID: "wa"}).Solve(context.Background(), g, req, core.WorkBudget{}, bearing.NullObserver{})
	seed := SeedState{Dist: []float64{0, math.Inf(1), math.Inf(1), math.Inf(1), math.Inf(1)}, Frontier: []core.NodeID{0}}
	seeded := (SeededWeightedAStar{Weight: 1.12, Seed: seed, RequireSeed: true}).Solve(context.Background(), g, req, core.WorkBudget{}, bearing.NullObserver{})
	if normal.Found != seeded.Found || math.Abs(normal.Distance-seeded.Distance) > 1e-9 {
		t.Fatalf("result mismatch normal=%+v seeded=%+v", normal, seeded)
	}
	if normal.TotalWork() != seeded.TotalWork() {
		t.Fatalf("work mismatch normal=%d seeded=%d", normal.TotalWork(), seeded.TotalWork())
	}
}
