package traffic_test

import (
	"context"
	"testing"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
)

func TestVerifyRepeatability(t *testing.T) {
	graph, err := traffic.Grid(30, 10, 73)
	if err != nil {
		t.Fatal(err)
	}
	budget := uint64(250)
	request := core.RouteRequest{
		Source: 0, Target: core.NodeID(graph.NodeCount() - 1),
		Mode: core.ModeBalanced, Workers: 1, Seed: 73, WorkBudget: &budget,
	}
	_, digest, err := traffic.VerifyRepeatability(context.Background(), gate.New(nil), graph, request, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(digest) != 64 {
		t.Fatalf("digest=%q", digest)
	}
}

func TestConstructionOrderDoesNotChangeRoute(t *testing.T) {
	first := core.NewAdjacencyGraph(5, false)
	second := core.NewAdjacencyGraph(5, false)
	edges := [][3]float64{{0, 1, 1}, {0, 2, 1}, {1, 3, 1}, {2, 3, 1}, {3, 4, 1}}
	for _, edge := range edges {
		_ = first.AddEdge(core.NodeID(edge[0]), core.NodeID(edge[1]), edge[2])
	}
	for i := len(edges) - 1; i >= 0; i-- {
		edge := edges[i]
		_ = second.AddEdge(core.NodeID(edge[0]), core.NodeID(edge[1]), edge[2])
	}
	request := core.RouteRequest{Source: 0, Target: 4, Mode: core.ModeExact, Workers: 1, Seed: 1}
	a, err := gate.New(nil).Route(context.Background(), first, request)
	if err != nil {
		t.Fatal(err)
	}
	b, err := gate.New(nil).Route(context.Background(), second, request)
	if err != nil {
		t.Fatal(err)
	}
	da, _ := traffic.StableDigest(a)
	db, _ := traffic.StableDigest(b)
	if da != db {
		t.Fatalf("construction order changed result: %s != %s; paths=%v %v", da, db, a.Path, b.Path)
	}
}
