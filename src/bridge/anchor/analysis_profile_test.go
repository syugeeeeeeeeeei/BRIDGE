package anchor

import (
	"math"
	"testing"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

type profileGraph struct {
	core.Graph
	profile core.GraphAnalysisProfile
}

func (g profileGraph) GraphAnalysisProfile() core.GraphAnalysisProfile { return g.profile }

func TestRecommendedWeightUsesPreparedGraphProfile(t *testing.T) {
	base := core.NewAdjacencyGraph(3, true)
	_ = base.SetPosition(0, core.Point{X: 0, Y: 0})
	_ = base.SetPosition(1, core.Point{X: 1, Y: 0})
	_ = base.SetPosition(2, core.Point{X: 2, Y: 0})
	_ = base.AddEdge(0, 1, 1)
	_ = base.AddEdge(1, 2, 1)
	profile := base.PrepareAnalysisProfile()
	wrapped := profileGraph{Graph: panicOnTraversalGraph{Graph: base}, profile: profile}
	got := RecommendedWeight(wrapped, core.ModeBalanced)
	want := RecommendedWeight(base, core.ModeBalanced)
	if got != want {
		t.Fatalf("prepared profile changed weight: got %v want %v", got, want)
	}
}

func TestHeuristicUnitScaleUsesPreparedGraphProfile(t *testing.T) {
	base := core.NewAdjacencyGraph(2, true)
	_ = base.SetPosition(0, core.Point{X: 0, Y: 0})
	_ = base.SetPosition(1, core.Point{X: 2, Y: 0})
	_ = base.AddEdge(0, 1, 3)
	profile := base.PrepareAnalysisProfile()
	wrapped := profileGraph{Graph: panicOnTraversalGraph{Graph: base}, profile: profile}
	got := graphHeuristicUnitScale(wrapped)
	if math.Abs(got-1.5) > 1e-12 {
		t.Fatalf("unexpected heuristic unit scale: %v", got)
	}
}

type panicOnTraversalGraph struct{ core.Graph }

func (g panicOnTraversalGraph) EdgesFrom(core.NodeID) []core.Edge {
	panic("route-time graph traversal must not occur when a prepared profile exists")
}
func (g panicOnTraversalGraph) Position(core.NodeID) (core.Point, bool) {
	panic("route-time position traversal must not occur when a prepared profile exists")
}
