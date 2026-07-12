package main

import (
	"context"
	"encoding/csv"
	"flag"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bolts"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/truss"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/ultrasound"
)

type solver interface {
	Name() string
	Solve(context.Context, core.Graph, core.RouteRequest, core.WorkBudget, bearing.Observer) core.RouteResult
}
type bridgeSolver struct{}

func (bridgeSolver) Name() string { return "bridge" }
func (bridgeSolver) Solve(ctx context.Context, g core.Graph, r core.RouteRequest, b core.WorkBudget, o bearing.Observer) core.RouteResult {
	r.WorkBudget = b.MaxWork
	x, _ := truss.New(o).Route(ctx, g, r)
	return x
}

type instance struct {
	name, family string
	g            *core.AdjacencyGraph
	s, t         core.NodeID
}

func main() {
	sizesFlag := flag.String("sizes", "100,500,1000,5000,10000,20000", "comma-separated node counts")
	seedsFlag := flag.String("seeds", "1,2,3", "comma-separated seeds")
	flag.Parse()
	sizes := parseInts(*sizesFlag)
	seeds := parseInt64s(*seedsFlag)
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()
	header := []string{"requested_nodes", "nodes", "edges", "topology", "family", "seed", "solver", "found", "exact_found", "distance", "exact_distance", "distance_ratio", "total_work", "select", "expand", "evaluate", "relax", "enqueue", "reject", "backtrack", "connect", "candidate", "repair", "bound", "terminate", "logical_steps", "scheduled_steps", "queue_pushes", "queue_pops", "parallel_steps", "time_ms", "budget_exhausted", "error_code", "trace_events", "investigated_nodes", "investigated_node_ratio", "investigated_edges", "investigated_edge_ratio", "candidate_paths", "path_node_count", "portfolio_unique_nodes", "portfolio_unique_node_ratio", "cross_component_duplicate_nodes", "portfolio_unique_edges", "portfolio_unique_edge_ratio", "cross_component_duplicate_edges", "anchor_time_ms", "bolts_time_ms", "supervisor_time_ms", "arbiter_time_ms", "orchestration_overhead_ms"}
	must(w.Write(header))
	solvers := []solver{bridgeSolver{}, bolts.AStar{}, bolts.BidirectionalDijkstra{}}
	for _, n := range sizes {
		for _, seed := range seeds {
			for _, in := range buildAll(n, seed) {
				req := core.RouteRequest{Source: in.s, Target: in.t, Mode: core.ModeBalanced, Workers: 1, Seed: uint64(seed)}
				exact := bolts.BidirectionalDijkstra{}.Solve(context.Background(), in.g, req, core.WorkBudget{}, bearing.NullObserver{})
				for _, sv := range solvers {
					max := uint64(100 * maxInt(n, in.g.NodeCount()))
					wb := core.WorkBudget{MaxWork: &max}
					mo := &ultrasound.MemoryObserver{}
					r := sv.Solve(context.Background(), in.g, req, wb, mo)
					ratio := math.NaN()
					if r.Found && exact.Found && exact.Distance > 0 {
						ratio = r.Distance / exact.Distance
					}
					row := []string{i(n), i(in.g.NodeCount()), i(in.g.EdgeCount()), in.name, in.family, i64(seed), sv.Name(), b(r.Found), b(exact.Found), f(r.Distance), f(exact.Distance), f(ratio), u(r.Work.TotalActions), u(r.Work.SelectActions), u(r.Work.ExpandActions), u(r.Work.EvaluateActions), u(r.Work.RelaxActions), u(r.Work.EnqueueActions), u(r.Work.RejectActions), u(r.Work.BacktrackActions), u(r.Work.ConnectActions), u(r.Work.CandidateActions), u(r.Work.RepairActions), u(r.Work.BoundActions), u(r.Work.TerminateActions), u(r.Work.LogicalSteps), u(r.Work.ScheduledSteps), u(r.QueuePushes), u(r.QueuePops), u(r.ParallelSteps), f(r.TimeMS), b(r.BudgetExhausted), string(r.ErrorCode), i(len(mo.Events)), anyU(r.Telemetry, "investigated_nodes"), anyF(r.Telemetry, "investigated_node_ratio"), anyI(r.Telemetry, "investigated_edges"), anyF(r.Telemetry, "investigated_edge_ratio"), anyU(r.Telemetry, "candidate_paths"), anyI(r.Telemetry, "path_node_count"), anyI(r.Telemetry, "portfolio_unique_nodes"), anyF(r.Telemetry, "portfolio_unique_node_ratio"), anyI(r.Telemetry, "cross_component_duplicate_nodes"), anyI(r.Telemetry, "portfolio_unique_edges"), anyF(r.Telemetry, "portfolio_unique_edge_ratio"), anyI(r.Telemetry, "cross_component_duplicate_edges"), anyF(r.Telemetry, "anchor_time_ms"), anyF(r.Telemetry, "bolts_time_ms"), anyF(r.Telemetry, "supervisor_time_ms"), anyF(r.Telemetry, "arbiter_time_ms"), anyF(r.Telemetry, "orchestration_overhead_ms")}
					must(w.Write(row))
				}
			}
		}
	}
	must(w.Error())
}

func buildAll(n int, seed int64) []instance {
	out := []instance{}
	for _, top := range []traffic.GridTopology{traffic.TopologyOpen, traffic.TopologyWall, traffic.TopologyUShape, traffic.TopologyCulDeSac, traffic.TopologyDisconnected} {
		g, s, t, e := traffic.TopologyGrid(n, top, seed, .15)
		if e == nil {
			out = append(out, instance{string(top), "grid", g, s, t})
		}
	}
	out = append(out, noisyGrid(n, seed), portals(n, seed), hub(n), tree(n), chain(n))
	return out
}
func noisyGrid(n int, seed int64) instance {
	side := int(math.Sqrt(float64(n)))
	if side < 10 {
		side = 10
	}
	nn := side * side
	g := core.NewAdjacencyGraph(nn, false)
	r := rand.New(rand.NewSource(seed))
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			id := y*side + x
			_ = g.SetPosition(core.NodeID(id), core.Point{X: float64(x), Y: float64(y)})
			if x+1 < side {
				_ = g.AddEdge(core.NodeID(id), core.NodeID(id+1), .5+2*r.Float64())
			}
			if y+1 < side {
				_ = g.AddEdge(core.NodeID(id), core.NodeID(id+side), .5+2*r.Float64())
			}
		}
	}
	return instance{"grid_weighted_noise", "weighted", g, 0, core.NodeID(nn - 1)}
}
func portals(n int, seed int64) instance {
	g := chainGrid(n)
	r := rand.New(rand.NewSource(seed))
	count := maxInt(4, n/100)
	for k := 0; k < count; k++ {
		u := r.Intn(n)
		v := r.Intn(n)
		if u != v {
			_ = g.AddEdge(core.NodeID(u), core.NodeID(v), .1)
		}
	}
	return instance{"cheap_portals", "portal", g, 0, core.NodeID(n - 1)}
}
func chainGrid(n int) *core.AdjacencyGraph {
	g := core.NewAdjacencyGraph(n, false)
	side := int(math.Ceil(math.Sqrt(float64(n))))
	for i := 0; i < n; i++ {
		x, y := i%side, i/side
		_ = g.SetPosition(core.NodeID(i), core.Point{X: float64(x), Y: float64(y)})
		if i > 0 {
			_ = g.AddEdge(core.NodeID(i-1), core.NodeID(i), 1)
		}
		if i >= side {
			_ = g.AddEdge(core.NodeID(i-side), core.NodeID(i), 1)
		}
	}
	return g
}
func hub(n int) instance {
	g := core.NewAdjacencyGraph(n, false)
	for i := 0; i < n; i++ {
		_ = g.SetPosition(core.NodeID(i), core.Point{X: float64(i), Y: 0})
	}
	for i := 1; i < n; i++ {
		_ = g.AddEdge(0, core.NodeID(i), 1+float64(i%9)/10)
	}
	return instance{"hub_spoke", "hub", g, 1, core.NodeID(n - 1)}
}
func tree(n int) instance {
	g := core.NewAdjacencyGraph(n, false)
	for i := 0; i < n; i++ {
		_ = g.SetPosition(core.NodeID(i), core.Point{X: float64(i), Y: math.Floor(math.Log2(float64(i + 1)))})
		if i > 0 {
			_ = g.AddEdge(core.NodeID((i-1)/2), core.NodeID(i), 1)
		}
	}
	s := maxInt(1, n/2)
	return instance{"binary_tree", "tree", g, core.NodeID(s), core.NodeID(n - 1)}
}
func chain(n int) instance {
	g := core.NewAdjacencyGraph(n, false)
	for i := 0; i < n; i++ {
		_ = g.SetPosition(core.NodeID(i), core.Point{X: float64(i), Y: 0})
		if i > 0 {
			_ = g.AddEdge(core.NodeID(i-1), core.NodeID(i), 1)
		}
	}
	return instance{"long_chain", "long_path", g, 0, core.NodeID(n - 1)}
}
func parseInts(s string) []int {
	var x []int
	for _, p := range strings.Split(s, ",") {
		v, e := strconv.Atoi(strings.TrimSpace(p))
		if e == nil && v > 1 {
			x = append(x, v)
		}
	}
	return x
}
func parseInt64s(s string) []int64 {
	var x []int64
	for _, p := range strings.Split(s, ",") {
		v, e := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
		if e == nil {
			x = append(x, v)
		}
	}
	return x
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func must(e error) {
	if e != nil {
		panic(e)
	}
}
func i(v int) string     { return strconv.Itoa(v) }
func i64(v int64) string { return strconv.FormatInt(v, 10) }
func u(v uint64) string  { return strconv.FormatUint(v, 10) }
func b(v bool) string    { return strconv.FormatBool(v) }
func f(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }
func anyI(m map[string]any, k string) string {
	if m == nil {
		return "0"
	}
	switch v := m[k].(type) {
	case int:
		return strconv.Itoa(v)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float64:
		return strconv.Itoa(int(v))
	}
	return "0"
}
func anyU(m map[string]any, k string) string { return anyI(m, k) }
func anyF(m map[string]any, k string) string {
	if m == nil {
		return "0"
	}
	switch v := m[k].(type) {
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64)
	case uint64:
		return strconv.FormatUint(v, 10)
	case int:
		return strconv.Itoa(v)
	}
	return "0"
}
