package truss

import (
	"context"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"testing"
)

func handoffTestGraph() *core.AdjacencyGraph {
	g := core.NewAdjacencyGraph(80, false)
	for i := 0; i < 79; i++ {
		_ = g.AddEdge(core.NodeID(i), core.NodeID(i+1), 1)
	}
	return g
}

func TestDebugProgressSamplesAndHandoffThresholdSweep(t *testing.T) {
	thresholds := []uint64{64, 128, 256}
	previous := uint64(0)
	for _, threshold := range thresholds {
		budget := uint64(2000)
		got, err := New(nil).Route(context.Background(), handoffTestGraph(), core.RouteRequest{
			Source: 0, Target: 79, Mode: core.ModeBalanced, Workers: 1,
			WorkBudget: &budget, CollectProgressSamples: true, HandoffWorkThreshold: &threshold,
		})
		if err != nil {
			t.Fatal(err)
		}
		if got.BottleneckProfile == nil || len(got.BottleneckProfile.ProgressSamples) == 0 {
			t.Fatalf("threshold=%d: progress samples missing", threshold)
		}
		s := got.BottleneckProfile.ProgressSamples[0]
		if s.Work < 64 || s.RejectRate < 0 || s.RejectRate > 1 {
			t.Fatalf("threshold=%d: incomplete sample: %+v", threshold, s)
		}
		if got.HandoffMetrics == nil || len(got.HandoffMetrics.Records) == 0 {
			t.Fatalf("threshold=%d: handoff record missing", threshold)
		}
		at := got.HandoffMetrics.Records[0].AnchorWorkAtHandoff
		if at < threshold || at > threshold+4 {
			t.Fatalf("threshold=%d: handoff work=%d", threshold, at)
		}
		if previous > 0 && at <= previous {
			t.Fatalf("threshold sweep not monotonic: %d <= %d", at, previous)
		}
		previous = at
	}
}
