package bolts

import (
	"context"
	"math"
	"math/rand"
	"testing"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

func TestReferenceSolversMatchDijkstraOnRandomNonNegativeGraphs(t *testing.T) {
	for seed := int64(1); seed <= 40; seed++ {
		rng := rand.New(rand.NewSource(seed))
		n := 8 + rng.Intn(24)
		directed := seed%2 == 0
		g := core.NewAdjacencyGraph(n, directed)
		for i := 0; i < n; i++ {
			g.SetPosition(core.NodeID(i), core.Point{X: rng.Float64() * 100, Y: rng.Float64() * 100})
		}
		for i := 0; i < n-1; i++ {
			_ = g.AddEdge(core.NodeID(i), core.NodeID(i+1), 0.1+rng.Float64()*9.9)
		}
		for i := 0; i < n*4; i++ {
			u, v := rng.Intn(n), rng.Intn(n)
			if u != v {
				_ = g.AddEdge(core.NodeID(u), core.NodeID(v), 0.1+rng.Float64()*9.9)
			}
		}
		for q := 0; q < 12; q++ {
			s, z := core.NodeID(rng.Intn(n)), core.NodeID(rng.Intn(n))
			req := core.RouteRequest{Source: s, Target: z, Workers: 1}
			oracle := (Dijkstra{}).Solve(context.Background(), g, req, core.WorkBudget{}, bearing.NullObserver{})
			for _, solver := range []Solver{AStar{}, BidirectionalDijkstra{}} {
				got := solver.Solve(context.Background(), g, req, core.WorkBudget{}, bearing.NullObserver{})
				if got.Found != oracle.Found {
					t.Fatalf("seed=%d query=%d solver=%s found=%v oracle=%v", seed, q, solver.Name(), got.Found, oracle.Found)
				}
				if got.Found && math.Abs(got.Distance-oracle.Distance) > 1e-9 {
					t.Fatalf("seed=%d query=%d solver=%s distance=%g oracle=%g", seed, q, solver.Name(), got.Distance, oracle.Distance)
				}
			}
		}
	}
}

func TestReferenceSolverTerminationContracts(t *testing.T) {
	g := core.NewAdjacencyGraph(3, true)
	_ = g.AddEdge(0, 1, 1)
	req := core.RouteRequest{Source: 0, Target: 2, Workers: 1}
	for _, solver := range []Solver{Dijkstra{}, AStar{}, BidirectionalDijkstra{}} {
		got := solver.Solve(context.Background(), g, req, core.WorkBudget{}, bearing.NullObserver{})
		if got.Found || !got.SearchCompleted || got.TerminationStatus != core.TerminationUnreachable || got.ErrorCode != core.ErrNoPath {
			t.Fatalf("solver=%s invalid unreachable contract: %+v", solver.Name(), got)
		}
	}
}

func TestLocalExecutorProducesUnreachableEvidence(t *testing.T) {
	g := core.NewAdjacencyGraph(3, true)
	_ = g.AddEdge(0, 1, 1)
	out, err := (LocalExecutor{}).Execute(context.Background(), g, core.HandoffRequest{
		ID: "unreachable", Purpose: core.ProveUnreachable,
		Inputs:       []core.Checkpoint{{Node: 0}, {Node: 2}},
		Region:       core.Region{Nodes: []core.NodeID{0, 1, 2}},
		HypothesisID: "h", Budget: core.WorkBudget{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Evidence) != 1 || out.Evidence[0].Proof != core.ProofUnreachable {
		t.Fatalf("missing unreachable evidence: %+v", out)
	}
}
