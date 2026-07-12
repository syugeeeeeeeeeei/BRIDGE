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
