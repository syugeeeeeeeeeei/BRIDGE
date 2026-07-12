package core

import (
	"math"
	"testing"
)

func TestGraphAndPathDistance(t *testing.T) {
	g := NewAdjacencyGraph(3, false)
	_ = g.AddEdge(0, 1, 2)
	_ = g.AddEdge(1, 2, 3)
	if d := PathDistance(g, []NodeID{0, 1, 2}); math.Abs(d-5) > 1e-12 {
		t.Fatalf("distance=%v", d)
	}
}
func TestRejectNegativeWeight(t *testing.T) {
	g := NewAdjacencyGraph(2, false)
	if g.AddEdge(0, 1, -1) == nil {
		t.Fatal("negative weight accepted")
	}
}

func TestAdjacencyOrderIsCanonical(t *testing.T) {
	a := NewAdjacencyGraph(4, true)
	_ = a.AddEdge(0, 3, 2)
	_ = a.AddEdge(0, 1, 5)
	_ = a.AddEdge(0, 2, 1)
	_ = a.AddEdge(0, 1, 1)

	got := a.EdgesFrom(0)
	want := []Edge{{To: 1, Weight: 1}, {To: 1, Weight: 5}, {To: 2, Weight: 1}, {To: 3, Weight: 2}}
	if len(got) != len(want) {
		t.Fatalf("len=%d want=%d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("edge[%d]=%+v want=%+v", i, got[i], want[i])
		}
	}
}

func TestWorkMetricsInvariant(t *testing.T) {
	w := WorkMetrics{WorkerCount: 1}
	for _, k := range []string{"select", "expand", "evaluate", "relax", "enqueue", "reject", "terminate"} {
		w.AddAction(k)
		w.LogicalSteps++
		w.ScheduledSteps++
	}
	if !w.Valid() {
		t.Fatalf("invalid work metrics: %+v", w)
	}
}
