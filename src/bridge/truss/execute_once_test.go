package truss

import (
	"context"
	"testing"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

func testGraph() *core.AdjacencyGraph {
	g := core.NewAdjacencyGraph(4, false)
	_ = g.AddEdge(0, 1, 1)
	_ = g.AddEdge(1, 3, 1)
	_ = g.AddEdge(0, 2, 5)
	_ = g.AddEdge(2, 3, 1)
	return g
}

func TestExecuteOnceAnchor(t *testing.T) {
	got, err := New(nil).ExecuteOnce(context.Background(), testGraph(), ExecuteOnceRequest{
		TargetID: "anchor",
		Source:   0,
		Target:   3,
		Workers:  1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.TargetID != "anchor" || got.TargetKind != TargetKindAlgorithm || got.ExecutionPath != "execute_once" {
		t.Fatalf("unexpected metadata: %+v", got)
	}
}

func TestExecuteOnceBoltsSolver(t *testing.T) {
	got, err := New(nil).ExecuteOnce(context.Background(), testGraph(), ExecuteOnceRequest{
		TargetID: "dijkstra",
		Source:   0,
		Target:   3,
		Workers:  1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Result.Found || got.TargetKind != TargetKindBoltsSolver || got.Result.SolverName != "dijkstra" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestExecuteOnceRejectsUnknownTarget(t *testing.T) {
	if _, err := New(nil).ExecuteOnce(context.Background(), testGraph(), ExecuteOnceRequest{
		TargetID: "unknown",
		Source:   0,
		Target:   3,
		Workers:  1,
	}); err == nil {
		t.Fatal("expected error")
	}
}

func TestRoutePropagatesSolverNanoseconds(t *testing.T) {
	got, err := New(nil).Route(context.Background(), testGraph(), core.RouteRequest{
		Source:  0,
		Target:  3,
		Mode:    core.ModeBalanced,
		Workers: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	b := got.TimeBreakdown
	if b.SolverNS <= 0 {
		t.Fatalf("solver nanoseconds were not propagated: %+v", b)
	}
	if b.SolverNS != b.AnchorNS+b.BoltsNS {
		t.Fatalf("solver_ns must equal anchor_ns + bolts_ns: %+v", b)
	}
	if b.SolverMS != float64(b.SolverNS)/1_000_000 {
		t.Fatalf("solver_ms must derive from solver_ns: %+v", b)
	}
	if b.TotalNS < b.SolverNS {
		t.Fatalf("total_ns must not be less than solver_ns: %+v", b)
	}
	if got.Telemetry["solver_time_ns"] != b.SolverNS {
		t.Fatalf("telemetry solver_time_ns mismatch: telemetry=%v breakdown=%+v", got.Telemetry["solver_time_ns"], b)
	}
}

func TestExecuteOncePreservesSolverNanoseconds(t *testing.T) {
	got, err := New(nil).ExecuteOnce(context.Background(), testGraph(), ExecuteOnceRequest{
		TargetID: "anchor",
		Source:   0,
		Target:   3,
		Workers:  1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Result.TimeBreakdown.SolverNS <= 0 {
		t.Fatalf("solver nanoseconds missing: %+v", got.Result.TimeBreakdown)
	}
	if got.Result.TimeBreakdown.TotalNS < got.Result.TimeBreakdown.SolverNS {
		t.Fatalf("execute_once total shorter than solver: %+v", got.Result.TimeBreakdown)
	}
}
