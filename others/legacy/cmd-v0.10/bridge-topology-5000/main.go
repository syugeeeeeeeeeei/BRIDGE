package main

import (
	"context"
	"encoding/csv"
	"math"
	"math/rand"
	"os"
	"strconv"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bolts"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
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
	res, _ := truss.New(o).Route(ctx, g, r)
	return res
}

type instance struct {
	name, family string
	g            *core.AdjacencyGraph
	s, t         core.NodeID
}

const N = 5000

func main() {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()
	header := []string{"nodes", "edges", "topology", "family", "seed", "solver", "found", "exact_found", "distance", "exact_distance", "distance_ratio", "total_work", "select", "expand", "evaluate", "relax", "enqueue", "reject", "backtrack", "connect", "candidate", "repair", "bound", "terminate", "logical_steps", "scheduled_steps", "queue_pushes", "queue_pops", "parallel_steps", "time_ms", "budget_exhausted", "error_code", "trace_events", "investigated_nodes", "investigated_node_ratio", "investigated_edges", "investigated_edge_ratio", "candidate_paths", "path_node_count", "portfolio_unique_nodes", "portfolio_unique_node_ratio", "cross_component_duplicate_nodes", "portfolio_unique_edges", "portfolio_unique_edge_ratio", "cross_component_duplicate_edges", "anchor_time_ms", "bolts_time_ms", "supervisor_time_ms", "arbiter_time_ms", "orchestration_overhead_ms"}
	must(w.Write(header))
	solvers := []solver{bridgeSolver{}, bolts.AStar{}, bolts.BidirectionalDijkstra{}}
	for _, seed := range []int64{1, 2, 3, 4, 5} {
		for _, in := range buildAll(seed) {
			req := core.RouteRequest{Source: in.s, Target: in.t, Mode: core.ModeBalanced, Workers: 1, Seed: uint64(seed)}
			exact := bolts.BidirectionalDijkstra{}.Solve(context.Background(), in.g, req, core.WorkBudget{}, bearing.NullObserver{})
			for _, sv := range solvers {
				max := uint64(100 * N)
				wb := core.WorkBudget{MaxWork: &max}

				mo := &ultrasound.MemoryObserver{}
				r := sv.Solve(context.Background(), in.g, req, wb, mo)
				ratio := math.NaN()
				if r.Found && exact.Found && exact.Distance > 0 {
					ratio = r.Distance / exact.Distance
				}
				row := []string{i(N), i(in.g.EdgeCount()), in.name, in.family, i64(seed), sv.Name(), b(r.Found), b(exact.Found), f(r.Distance), f(exact.Distance), f(ratio), u(r.Work.TotalActions), u(r.Work.SelectActions), u(r.Work.ExpandActions), u(r.Work.EvaluateActions), u(r.Work.RelaxActions), u(r.Work.EnqueueActions), u(r.Work.RejectActions), u(r.Work.BacktrackActions), u(r.Work.ConnectActions), u(r.Work.CandidateActions), u(r.Work.RepairActions), u(r.Work.BoundActions), u(r.Work.TerminateActions), u(r.Work.LogicalSteps), u(r.Work.ScheduledSteps), u(r.QueuePushes), u(r.QueuePops), u(r.ParallelSteps), f(r.TimeMS), b(r.BudgetExhausted), string(r.ErrorCode), i(len(mo.Events)), anyU(r.Telemetry, "investigated_nodes"), anyF(r.Telemetry, "investigated_node_ratio"), anyI(r.Telemetry, "investigated_edges"), anyF(r.Telemetry, "investigated_edge_ratio"), anyU(r.Telemetry, "candidate_paths"), anyI(r.Telemetry, "path_node_count"), anyI(r.Telemetry, "portfolio_unique_nodes"), anyF(r.Telemetry, "portfolio_unique_node_ratio"), anyI(r.Telemetry, "cross_component_duplicate_nodes"), anyI(r.Telemetry, "portfolio_unique_edges"), anyF(r.Telemetry, "portfolio_unique_edge_ratio"), anyI(r.Telemetry, "cross_component_duplicate_edges"), anyF(r.Telemetry, "anchor_time_ms"), anyF(r.Telemetry, "bolts_time_ms"), anyF(r.Telemetry, "supervisor_time_ms"), anyF(r.Telemetry, "arbiter_time_ms"), anyF(r.Telemetry, "orchestration_overhead_ms")}
				must(w.Write(row))
			}
		}
	}
	must(w.Error())
}

func buildAll(seed int64) []instance {
	return []instance{
		gridOpen(seed, false), gridOpen(seed, true), alternatingWalls(seed), randomObstacles(seed), roomsDoors(seed), comb(seed), snake(seed), deceptiveWeights(seed), portals(seed), hubSpoke(seed), ringLattice(seed), treeBranches(seed), disconnected(seed),
	}
}

func baseGrid() *core.AdjacencyGraph {
	g := core.NewAdjacencyGraph(N, false)
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			id := core.NodeID(y*100 + x)
			_ = g.SetPosition(id, core.Point{X: float64(x), Y: float64(y)})
		}
	}
	return g
}
func addGridEdges(g *core.AdjacencyGraph, blocked []bool, weight func(u, v, x, y, nx, ny int) float64) {
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			u := y*100 + x
			if blocked != nil && blocked[u] {
				continue
			}
			for _, d := range [][2]int{{1, 0}, {0, 1}} {
				nx, ny := x+d[0], y+d[1]
				if nx >= 100 || ny >= 50 {
					continue
				}
				v := ny*100 + nx
				if blocked != nil && blocked[v] {
					continue
				}
				_ = g.AddEdge(core.NodeID(u), core.NodeID(v), weight(u, v, x, y, nx, ny))
			}
		}
	}
}
func noise(seed int64, u, v int) float64 {
	r := rand.New(rand.NewSource(seed + int64(u*1000003+v*9176)))
	return r.Float64()
}
func gridOpen(seed int64, noisy bool) instance {
	g := baseGrid()
	addGridEdges(g, nil, func(u, v, x, y, nx, ny int) float64 {
		if noisy {
			return 1 + .5*noise(seed, u, v)
		}
		return 1
	})
	name := "open_uniform"
	if noisy {
		name = "open_noisy50"
	}
	return instance{name, "grid", g, 0, N - 1}
}
func alternatingWalls(seed int64) instance {
	g := baseGrid()
	bl := make([]bool, N)
	for x := 10; x < 100; x += 10 {
		gap := 3
		if (x/10)%2 == 0 {
			for y := 0; y < 50-gap; y++ {
				bl[y*100+x] = true
			}
		} else {
			for y := gap; y < 50; y++ {
				bl[y*100+x] = true
			}
		}
	}
	addGridEdges(g, bl, func(u, v, x, y, nx, ny int) float64 { return 1 + .05*noise(seed, u, v) })
	return instance{"alternating_walls", "maze", g, 0, N - 1}
}
func randomObstacles(seed int64) instance {
	g := baseGrid()
	bl := make([]bool, N)
	r := rand.New(rand.NewSource(seed))
	for i := range bl {
		bl[i] = r.Float64() < .22
	}
	for x := 0; x < 100; x++ {
		bl[x] = false
	}
	for y := 0; y < 50; y++ {
		bl[y*100+99] = false
	}
	bl[0] = false
	bl[N-1] = false
	addGridEdges(g, bl, func(u, v, x, y, nx, ny int) float64 { return 1 + .1*noise(seed, u, v) })
	return instance{"random_obstacles22", "obstacle", g, 0, N - 1}
}
func roomsDoors(seed int64) instance {
	g := baseGrid()
	bl := make([]bool, N)
	// A guaranteed-connected sequence of rooms. Each vertical partition has two doors.
	for x := 20; x < 100; x += 20 {
		d1 := (int(seed)*7+x)%20 + 5
		d2 := (int(seed)*11+x)%20 + 27
		for y := 0; y < 50; y++ {
			if math.Abs(float64(y-d1)) > 1 && math.Abs(float64(y-d2)) > 1 {
				bl[y*100+x] = true
			}
		}
	}
	addGridEdges(g, bl, func(u, v, x, y, nx, ny int) float64 { return 1 + .05*noise(seed, u, v) })
	return instance{"rooms_multiple_doors", "bottleneck", g, 0, N - 1}
}
func comb(seed int64) instance {
	g := baseGrid()
	bl := make([]bool, N)
	for x := 5; x < 95; x += 5 {
		for y := 0; y < 44; y++ {
			if x%10 == 0 || y > 3 {
				bl[y*100+x] = true
			}
		}
	}
	addGridEdges(g, bl, func(u, v, x, y, nx, ny int) float64 { return 1 })
	return instance{"comb_traps", "trap", g, 0, N - 1}
}
func snake(seed int64) instance {
	g := baseGrid() // one Hamiltonian-like corridor; distractor geometry but path length ~N
	for y := 0; y < 50; y++ {
		if y%2 == 0 {
			for x := 0; x < 99; x++ {
				u := y*100 + x
				_ = g.AddEdge(core.NodeID(u), core.NodeID(u+1), 1)
			}
		} else {
			for x := 99; x > 0; x-- {
				u := y*100 + x
				_ = g.AddEdge(core.NodeID(u), core.NodeID(u-1), 1)
			}
		}
		if y < 49 {
			at := 99
			if y%2 == 1 {
				at = 0
			}
			u := y*100 + at
			_ = g.AddEdge(core.NodeID(u), core.NodeID(u+100), 1)
		}
	}
	return instance{"snake_long_path", "long_path", g, 0, 4900}
}
func deceptiveWeights(seed int64) instance {
	g := baseGrid()
	addGridEdges(g, nil, func(u, v, x, y, nx, ny int) float64 { // geometrically direct center expensive, perimeter cheap
		if y >= 18 && y <= 31 && ny >= 18 && ny <= 31 {
			return 25
		}
		if y == 0 || y == 49 || ny == 0 || ny == 49 {
			return .2
		}
		return 4
	})
	return instance{"deceptive_weights", "weighted", g, 2000 + 0, 2999}
}
func portals(seed int64) instance {
	g := baseGrid()
	addGridEdges(g, nil, func(u, v, x, y, nx, ny int) float64 { return 1 })
	r := rand.New(rand.NewSource(seed))
	for k := 0; k < 80; k++ {
		u := r.Intn(N)
		v := r.Intn(N)
		if u != v {
			_ = g.AddEdge(core.NodeID(u), core.NodeID(v), .25)
		}
	}
	return instance{"cheap_random_portals", "portal", g, 0, N - 1}
}
func hubSpoke(seed int64) instance {
	g := core.NewAdjacencyGraph(N, false)
	for i := 0; i < N; i++ {
		a := 2 * math.Pi * float64(i) / N
		_ = g.SetPosition(core.NodeID(i), core.Point{X: math.Cos(a), Y: math.Sin(a)})
	}
	for i := 1; i < N; i++ {
		_ = g.AddEdge(0, core.NodeID(i), 1+float64(i%7)/10)
	}
	for i := 1; i < N-1; i++ {
		_ = g.AddEdge(core.NodeID(i), core.NodeID(i+1), .8)
	}
	return instance{"hub_spoke", "hub", g, 1, N - 1}
}
func ringLattice(seed int64) instance {
	g := core.NewAdjacencyGraph(N, false)
	for i := 0; i < N; i++ {
		a := 2 * math.Pi * float64(i) / N
		_ = g.SetPosition(core.NodeID(i), core.Point{X: math.Cos(a), Y: math.Sin(a)})
		for _, d := range []int{1, 2, 5} {
			j := (i + d) % N
			_ = g.AddEdge(core.NodeID(i), core.NodeID(j), 1)
		}
	}
	return instance{"ring_lattice", "non_grid", g, 0, N / 2}
}
func treeBranches(seed int64) instance {
	g := core.NewAdjacencyGraph(N, false)
	for i := 0; i < N; i++ {
		level := math.Floor(math.Log2(float64(i + 1)))
		_ = g.SetPosition(core.NodeID(i), core.Point{X: float64(i), Y: level})
		if i > 0 {
			p := (i - 1) / 2
			_ = g.AddEdge(core.NodeID(p), core.NodeID(i), 1)
		}
	} // target deep leaf, source another deep leaf
	return instance{"binary_tree", "tree", g, 4095, 4999}
}
func disconnected(seed int64) instance {
	g := baseGrid()
	bl := make([]bool, N)
	for y := 0; y < 50; y++ {
		bl[y*100+50] = true
	}
	addGridEdges(g, bl, func(u, v, x, y, nx, ny int) float64 { return 1 })
	return instance{"disconnected_wall", "disconnected", g, 0, N - 1}
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
