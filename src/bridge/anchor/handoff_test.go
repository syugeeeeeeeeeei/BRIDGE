package anchor

import (
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"testing"
)

func TestApplyHandoffResumesSession(t *testing.T) {
	g := core.NewAdjacencyGraph(3, false)
	_ = g.AddEdge(0, 1, 1)
	_ = g.AddEdge(1, 2, 1)
	s, err := NewSession(g, core.RouteRequest{Source: 0, Target: 2, Workers: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	r := core.HandoffResult{RequestID: "x", Path: []core.NodeID{0, 1, 2}, Distance: 2, Found: true, ResumeCheckpoints: []core.Checkpoint{{Node: 2, Cost: 2, HypothesisID: "h0"}}, Evidence: []core.Evidence{{ID: "e", Solver: "bolts", Scope: core.Region{Nodes: []core.NodeID{0, 1, 2}}, GeneratedWork: 1, Proof: core.ProofExact, Value: 2}}}
	if err := s.ApplyHandoff(r); err != nil {
		t.Fatal(err)
	}
	if !s.Result().Found {
		t.Fatal("handoff candidate not imported")
	}
}
