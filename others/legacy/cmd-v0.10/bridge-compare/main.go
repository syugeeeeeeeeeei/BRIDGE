package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/anchor"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bolts"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/ultrasound"
)

type solver interface {
	Name() string
	Solve(context.Context, core.Graph, core.RouteRequest, core.WorkBudget, bearing.Observer) core.RouteResult
}

func main() {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()
	header := []string{"nodes", "topology", "seed", "solver", "found", "distance", "distance_ratio", "total_work", "select", "expand", "evaluate", "relax", "enqueue", "reject", "connect", "candidate", "terminate", "logical_steps", "scheduled_steps", "queue_pushes", "queue_pops", "time_ms", "budget_exhausted", "trace_events"}
	must(w.Write(header))
	sizes := []int{100, 484, 1024, 2025, 4900}
	tops := []traffic.GridTopology{traffic.TopologyOpen, traffic.TopologyWall, traffic.TopologyUShape, traffic.TopologyCulDeSac, traffic.TopologyDisconnected}
	solvers := []solver{anchor.Solver{Config: anchor.DefaultConfig(), Connector: bolts.BidirectionalDijkstra{}}, bolts.AStar{}, bolts.BidirectionalDijkstra{}}
	for _, n := range sizes {
		for _, top := range tops {
			for _, seed := range []int64{1, 2, 3} {
				g, s, t, err := traffic.TopologyGrid(n, top, seed, .05)
				must(err)
				req := core.RouteRequest{Source: s, Target: t, Mode: core.ModeBalanced, Workers: 1, Seed: uint64(seed)}
				exact := bolts.BidirectionalDijkstra{}.Solve(context.Background(), g, req, core.WorkBudget{}, bearing.NullObserver{})
				for _, sv := range solvers {
					mo := &ultrasound.MemoryObserver{}
					maxWork := uint64(100 * uint64(g.NodeCount()))
					wb := core.WorkBudget{MaxWork: &maxWork}

					res := sv.Solve(context.Background(), g, req, wb, mo)
					ratio := math.NaN()
					if res.Found && exact.Found && exact.Distance > 0 {
						ratio = res.Distance / exact.Distance
					}
					row := []string{itoa(g.NodeCount()), string(top), itoa64(seed), sv.Name(), btoa(res.Found), ftoa(res.Distance), ftoa(ratio), u(res.Work.TotalActions), u(res.Work.SelectActions), u(res.Work.ExpandActions), u(res.Work.EvaluateActions), u(res.Work.RelaxActions), u(res.Work.EnqueueActions), u(res.Work.RejectActions), u(res.Work.ConnectActions), u(res.Work.CandidateActions), u(res.Work.TerminateActions), u(res.Work.LogicalSteps), u(res.Work.ScheduledSteps), u(res.QueuePushes), u(res.QueuePops), ftoa(res.TimeMS), btoa(res.BudgetExhausted), itoa(len(mo.Events))}
					must(w.Write(row))
				}
			}
		}
	}
	must(w.Error())
}
func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func itoa(v int) string     { return strconv.Itoa(v) }
func itoa64(v int64) string { return strconv.FormatInt(v, 10) }
func u(v uint64) string     { return strconv.FormatUint(v, 10) }
func btoa(v bool) string    { return strconv.FormatBool(v) }
func ftoa(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }
