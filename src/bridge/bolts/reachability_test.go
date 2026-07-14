package bolts

import (
	"context"
	"testing"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

func TestReachabilityDoesNotClaimWeightedOptimality(t *testing.T) {
	g := core.NewAdjacencyGraph(4, true)
	// BFS discovers 0-1-3 first (cost 101), while the shortest weighted path
	// is 0-2-3 (cost 2). Reachability may return the former but must never
	// claim shortest-path optimality.
	_ = g.AddEdge(0, 1, 1)
	_ = g.AddEdge(1, 3, 100)
	_ = g.AddEdge(0, 2, 1)
	_ = g.AddEdge(2, 3, 1)

	got := (Reachability{}).Solve(context.Background(), g, core.RouteRequest{Source: 0, Target: 3, Workers: 1}, core.WorkBudget{}, bearing.NullObserver{})
	if !got.Found || !got.ReachabilityProven || !got.SearchCompleted {
		t.Fatalf("reachability proof missing: %+v", got)
	}
	if got.Exact {
		t.Fatalf("reachability solver falsely claimed weighted optimality: %+v", got)
	}
	if got.TimeBreakdown.SolverNS <= 0 || got.TimeBreakdown.BoltsNS <= 0 {
		t.Fatalf("solver timing boundary missing: %+v", got.TimeBreakdown)
	}
}

func TestReachabilityBudgetExhaustionIsNotAProof(t *testing.T) {
	g := localGraph()
	budget := uint64(1)
	got := (Reachability{}).Solve(context.Background(), g, core.RouteRequest{Source: 0, Target: 3, Workers: 1}, core.WorkBudget{MaxWork: &budget}, bearing.NullObserver{})
	if got.ReachabilityProven || got.SearchCompleted || got.Exact {
		t.Fatalf("budget exhaustion must not produce a proof: %+v", got)
	}
	if !got.BudgetExhausted || got.ErrorCode != core.ErrBudgetExhausted {
		t.Fatalf("wrong budget termination: %+v", got)
	}
	if got.TimeBreakdown.SolverNS <= 0 {
		t.Fatalf("timing must be recorded on early return: %+v", got.TimeBreakdown)
	}
}

func TestReachabilityUnreachableProof(t *testing.T) {
	g := core.NewAdjacencyGraph(3, true)
	_ = g.AddEdge(0, 1, 1)
	got := (Reachability{}).Solve(context.Background(), g, core.RouteRequest{Source: 0, Target: 2, Workers: 1}, core.WorkBudget{}, bearing.NullObserver{})
	if got.Found || !got.ReachabilityProven || !got.SearchCompleted || got.Exact {
		t.Fatalf("wrong unreachable semantics: %+v", got)
	}
	if got.TerminationStatus != core.TerminationUnreachable || got.ErrorCode != core.ErrNoPath {
		t.Fatalf("wrong unreachable termination: %+v", got)
	}
}
