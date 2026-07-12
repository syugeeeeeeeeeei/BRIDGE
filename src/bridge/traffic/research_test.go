package traffic_test

import (
	"context"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
	"math"
	"testing"
)

func TestLegacyTopologyFamilies(t *testing.T) {
	kinds := []traffic.GridTopology{traffic.TopologyOpen, traffic.TopologyWall, traffic.TopologyUShape, traffic.TopologyCulDeSac, traffic.TopologyDisconnected}
	for _, kind := range kinds {
		g, s, d, err := traffic.TopologyGrid(400, kind, 7, .05)
		if err != nil {
			t.Fatal(err)
		}
		exact, err := gate.New(nil).Route(context.Background(), g, core.RouteRequest{Source: s, Target: d, Mode: core.ModeExact, Workers: 1})
		if err != nil {
			t.Fatal(err)
		}
		if kind == traffic.TopologyDisconnected {
			if exact.Found {
				t.Fatalf("%s unexpectedly connected", kind)
			}
			continue
		}
		if !exact.Found || math.IsInf(exact.Distance, 0) {
			t.Fatalf("%s exact failed", kind)
		}
		budget := uint64(g.NodeCount() * 40)
		got, err := gate.New(nil).Route(context.Background(), g, core.RouteRequest{Source: s, Target: d, Mode: core.ModeBalanced, Workers: 1, WorkBudget: &budget})
		if err != nil {
			t.Fatal(err)
		}
		if !got.Found {
			t.Fatalf("%s balanced failed", kind)
		}
		ratio := got.Distance / exact.Distance
		if ratio > 1.35 {
			t.Fatalf("%s ratio %.4f", kind, ratio)
		}
		if got.TotalWork() > budget {
			t.Fatalf("%s budget violation", kind)
		}
		if !got.Work.Valid() {
			t.Fatalf("%s invalid work metrics: %+v", kind, got.Work)
		}
	}
}

func TestRandomGeometricResearchCase(t *testing.T) {
	g, s, d, err := traffic.RandomGeometric(250, 12, 20260709)
	if err != nil {
		t.Fatal(err)
	}
	exact, _ := gate.New(nil).Route(context.Background(), g, core.RouteRequest{Source: s, Target: d, Mode: core.ModeExact, Workers: 1})
	budget := uint64(g.NodeCount() * 40)
	got, _ := gate.New(nil).Route(context.Background(), g, core.RouteRequest{Source: s, Target: d, Mode: core.ModeBalanced, Workers: 1, WorkBudget: &budget})
	if !exact.Found || !got.Found {
		t.Fatalf("found exact=%v bridge=%v", exact.Found, got.Found)
	}
	if got.Distance/exact.Distance > 1.15 {
		t.Fatalf("distance ratio %.4f", got.Distance/exact.Distance)
	}
}

func TestSpearman(t *testing.T) {
	if r := traffic.Spearman([]float64{1, 2, 3, 4}, []float64{2, 4, 6, 8}); math.Abs(r-1) > 1e-12 {
		t.Fatalf("r=%f", r)
	}
}

