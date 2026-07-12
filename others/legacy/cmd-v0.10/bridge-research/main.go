package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
	"math"
	"os"
)

func main() {
	output := flag.String("output", "go_research.json", "output JSON")
	flag.Parse()
	sizes := []int{100, 225, 400, 625, 900}
	topologies := []traffic.GridTopology{traffic.TopologyOpen, traffic.TopologyWall, traffic.TopologyUShape, traffic.TopologyCulDeSac, traffic.TopologyDisconnected}
	seeds := []int64{1, 2, 3}
	rows := make([]traffic.ResearchRow, 0)
	runner := gate.New(nil)
	for _, top := range topologies {
		for _, n := range sizes {
			for _, seed := range seeds {
				g, s, t, err := traffic.TopologyGrid(n, top, seed, .05)
				if err != nil {
					panic(err)
				}
				exact, err := runner.Route(context.Background(), g, core.RouteRequest{Source: s, Target: t, Mode: core.ModeExact, Workers: 1, Seed: uint64(seed)})
				if err != nil {
					panic(err)
				}
				budget := uint64(g.NodeCount() * 40)
				got, err := runner.Route(context.Background(), g, core.RouteRequest{Source: s, Target: t, Mode: core.ModeBalanced, Workers: 1, Seed: uint64(seed), WorkBudget: &budget})
				if err != nil {
					panic(err)
				}
				ratio := math.Inf(1)
				match := false
				if got.Found && exact.Found && exact.Distance > 0 {
					ratio = got.Distance / exact.Distance
					match = math.Abs(got.Distance-exact.Distance) <= 1e-9*math.Max(1, exact.Distance)
				}
				if !got.Found && !exact.Found {
					ratio = 1
					match = true
				}
				gd, ed := got.Distance, exact.Distance
				if math.IsInf(gd, 0) || math.IsNaN(gd) {
					gd = 0
				}
				if math.IsInf(ed, 0) || math.IsNaN(ed) {
					ed = 0
				}
				if math.IsInf(ratio, 0) || math.IsNaN(ratio) {
					ratio = 1e300
				}
				rows = append(rows, traffic.ResearchRow{Implementation: "go", Topology: string(top), Nodes: g.NodeCount(), Seed: seed, Mode: "balanced", Found: got.Found, Distance: gd, ExactDistance: ed, DistanceRatio: ratio, ExactMatch: match, TotalWork: got.TotalWork(), ScheduledSteps: got.Work.ScheduledSteps, TimeMS: got.TimeMS})
			}
		}
	}
	payload, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*output, payload, 0644); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %d rows to %s\n", len(rows), *output)
}
