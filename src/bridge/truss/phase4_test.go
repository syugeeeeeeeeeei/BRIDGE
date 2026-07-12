package truss

import (
	"context"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"testing"
)

func TestDisableCertificationAblation(t *testing.T) {
	g := core.NewAdjacencyGraph(3, false)
	_ = g.AddEdge(0, 1, 1)
	_ = g.AddEdge(1, 2, 1)
	r := core.RouteRequest{Source: 0, Target: 2, Mode: core.ModeQuality, Workers: 1, Ablation: core.AblationOptions{DisableCertification: true}}
	got, err := New(nil).Route(context.Background(), g, r)
	if err != nil {
		t.Fatal(err)
	}
	for _, tr := range got.SolverTrace {
		if tr.Purpose == "certification" {
			t.Fatalf("certification executed: %+v", tr)
		}
	}
}
