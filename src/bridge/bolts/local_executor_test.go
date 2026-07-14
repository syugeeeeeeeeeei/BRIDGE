package bolts

import (
	"context"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"testing"
)

func localGraph() *core.AdjacencyGraph {
	g := core.NewAdjacencyGraph(4, false)
	_ = g.AddEdge(0, 1, 1)
	_ = g.AddEdge(1, 2, 1)
	_ = g.AddEdge(2, 3, 1)
	return g
}
func TestLocalExecutorCapabilities(t *testing.T) {
	g := localGraph()
	ex := LocalExecutor{}
	for _, p := range []core.HandoffPurpose{core.ConnectCheckpoints, core.RepairSegment, core.CertifyCandidate, core.TightenBound, core.ProveUnreachable} {
		req := core.HandoffRequest{ID: string(p), Purpose: p, Inputs: []core.Checkpoint{{Node: 0}, {Node: 3}}, Region: core.Region{Nodes: []core.NodeID{0, 1, 2, 3}}, Budget: core.WorkBudget{}, HypothesisID: "h0"}
		got, err := ex.Execute(context.Background(), g, req)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if p != core.ProveUnreachable && !got.Found {
			t.Fatalf("%s did not find path", p)
		}
	}
	req := core.HandoffRequest{ID: "escape", Purpose: core.EscapeRegion, Inputs: []core.Checkpoint{{Node: 0}}, Region: core.Region{Nodes: []core.NodeID{0, 1}}, HypothesisID: "h0"}
	got, err := ex.Execute(context.Background(), g, req)
	if err != nil || !got.Found {
		t.Fatalf("escape: %+v %v", got, err)
	}
}
