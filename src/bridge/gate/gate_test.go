package gate

import (
	"context"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"math"
	"testing"
)

func graph() *core.AdjacencyGraph {
	g := core.NewAdjacencyGraph(4, false)
	_ = g.AddEdge(0, 1, 1)
	_ = g.AddEdge(1, 3, 1)
	_ = g.AddEdge(0, 2, 5)
	_ = g.AddEdge(2, 3, 1)
	return g
}
func TestExact(t *testing.T) {
	res, err := New(nil).Route(context.Background(), graph(), core.RouteRequest{Source: 0, Target: 3, Mode: core.ModeExact, Workers: 1})
	if err != nil || !res.Found || !res.Exact || math.Abs(res.Distance-2) > 1e-12 {
		t.Fatalf("%+v %v", res, err)
	}
}
func TestBudget(t *testing.T) {
	b := uint64(1)
	res, err := New(nil).Route(context.Background(), graph(), core.RouteRequest{Source: 0, Target: 3, Mode: core.ModeExact, Workers: 1, WorkBudget: &b})
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalWork() > b {
		t.Fatalf("budget exceeded: %d", res.WorkExpandedNodes)
	}
}
