package anchor

import (
	"container/heap"
	"context"
	"math"
	"sort"
	"time"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

type Config struct {
	TargetWorkRatio        float64
	InitialPathBudgetRatio float64
	MinQualityBudgetRatio  float64
	MaxCorridors           int
	BaseWidthScale         float64
	RepairHops             int
	MaxRepairNodesRatio    float64
	HubCount               int
	ConnectorBudgetRatio   float64
	WeightedAStarFactor    float64
	InitialExpandRatio     float64
	ExtensionExpandRatios  []float64
	StallWindow            uint64
	ConnectorExpandRatio   float64
	MaxConnectorCalls      int
}

func DefaultConfig() Config {
	return Config{TargetWorkRatio: .45, InitialPathBudgetRatio: .18, MinQualityBudgetRatio: .06, MaxCorridors: 7, BaseWidthScale: .14, RepairHops: 1, MaxRepairNodesRatio: .22, HubCount: 8, ConnectorBudgetRatio: .16, WeightedAStarFactor: 4.0, InitialExpandRatio: .45, ExtensionExpandRatios: []float64{.75, 1.0, 1.5, 2.0}, StallWindow: 128, ConnectorExpandRatio: .20, MaxConnectorCalls: 3}
}

type Connector interface {
	Name() string
	Solve(context.Context, core.Graph, core.RouteRequest, core.WorkBudget, bearing.Observer) core.RouteResult
}

type Solver struct {
	Config    Config
	Connector Connector
}

func (Solver) Name() string { return "anchor" }

type features struct {
	hasPos                                         bool
	degreeCV, maxMeanDegreeRatio, top1DegreeShare  float64
	weightGeoRatioCV, edgeP95Median, edgeMaxMedian float64
}

func Analyze(g core.Graph) string {
	f := graphFeatures(g)
	if f.top1DegreeShare > .12 || f.maxMeanDegreeRatio > 8 || (!f.hasPos && f.degreeCV > .6) {
		return "hub_aware"
	}
	if f.hasPos && f.weightGeoRatioCV > .35 {
		return "weighted_cost"
	}
	if f.hasPos && f.degreeCV > .9 {
		return "portal"
	}
	if f.hasPos && f.edgeP95Median > 1.85 && f.edgeMaxMedian > 3 {
		return "portal"
	}
	if f.hasPos {
		return "geometric_corridor"
	}
	return "hub_aware"
}

func graphFeatures(g core.Graph) features {
	n := g.NodeCount()
	if n == 0 {
		return features{}
	}
	deg := make([]float64, n)
	sum := 0.0
	maxd := 0.0
	hasPos := false
	for i := 0; i < n; i++ {
		d := float64(len(g.EdgesFrom(core.NodeID(i))))
		deg[i] = d
		sum += d
		if d > maxd {
			maxd = d
		}
		if _, ok := g.Position(core.NodeID(i)); ok {
			hasPos = true
		}
	}
	mean := sum / float64(n)
	variance := 0.0
	for _, d := range deg {
		x := d - mean
		variance += x * x
	}
	variance /= float64(n)
	cv := math.Sqrt(variance) / math.Max(mean, 1e-12)
	sortedDeg := append([]float64(nil), deg...)
	sort.Sort(sort.Reverse(sort.Float64Slice(sortedDeg)))
	topK := int(.01 * float64(n))
	if topK < 1 {
		topK = 1
	}
	top := 0.0
	for i := 0; i < topK && i < n; i++ {
		top += sortedDeg[i]
	}
	f := features{hasPos: hasPos, degreeCV: cv, maxMeanDegreeRatio: maxd / math.Max(mean, 1e-12), top1DegreeShare: top / math.Max(sum, 1)}
	if !hasPos {
		return f
	}
	ratios := []float64{}
	lens := []float64{}
	maxSamples := 768
	for u := 0; u < n && len(lens) < maxSamples; u++ {
		pu, ok := g.Position(core.NodeID(u))
		if !ok {
			continue
		}
		for _, e := range g.EdgesFrom(core.NodeID(u)) {
			pv, ok := g.Position(e.To)
			if !ok {
				continue
			}
			d := core.Euclidean(pu, pv)
			if d > 1e-12 {
				ratios = append(ratios, e.Weight/d)
				lens = append(lens, d)
			}
			if len(lens) >= maxSamples {
				break
			}
		}
	}
	if len(ratios) > 0 {
		m := 0.0
		for _, x := range ratios {
			m += x
		}
		m /= float64(len(ratios))
		v := 0.0
		for _, x := range ratios {
			q := x - m
			v += q * q
		}
		f.weightGeoRatioCV = math.Sqrt(v/float64(len(ratios))) / math.Max(m, 1e-12)
	}
	if len(lens) > 0 {
		sort.Float64s(lens)
		med := math.Max(lens[len(lens)/2], 1e-12)
		f.edgeP95Median = lens[int(.95*float64(len(lens)-1))] / med
		f.edgeMaxMedian = lens[len(lens)-1] / med
	}
	return f
}

func (s Solver) Solve(ctx context.Context, g core.Graph, r core.RouteRequest, b core.WorkBudget, o bearing.Observer) core.RouteResult {
	cfg := s.Config
	if cfg.InitialExpandRatio == 0 {
		cfg = DefaultConfig()
	}
	initial := uint64(math.Ceil(float64(g.NodeCount()) * cfg.InitialExpandRatio))
	if initial < 1 {
		initial = 1
	}
	maxExpand := initial
	for _, ratio := range cfg.ExtensionExpandRatios {
		v := uint64(math.Ceil(float64(g.NodeCount()) * ratio))
		if v > maxExpand {
			maxExpand = v
		}
	}
	if b.MaxExpand != nil && maxExpand > *b.MaxExpand {
		maxExpand = *b.MaxExpand
	}
	res := discreteAnytimeSearch(ctx, g, r, b.MaxWork, initial, maxExpand, cfg, s.Connector, o)
	if res.Telemetry == nil {
		res.Telemetry = map[string]any{}
	}
	res.Telemetry["strategy"] = "staged_anchor_with_bounded_connector"
	res.Telemetry["initial_expand_target"] = initial
	res.Telemetry["max_expand_limit"] = maxExpand
	return finalizeTiming(res, "anchor")
}

func discreteAnytimeSearch(ctx context.Context, g core.Graph, r core.RouteRequest, actionBudget *uint64, initialExpandLimit, maxExpandLimit uint64, cfg Config, connector Connector, o bearing.Observer) core.RouteResult {
	start := time.Now()
	n := g.NodeCount()
	dist := make([]float64, n)
	prev := make([]core.NodeID, n)
	hasPrev := make([]bool, n)
	settled := make([]bool, n)
	uniqueExpanded := make([]bool, n)
	evaluatedEdges := make(map[[2]core.NodeID]struct{})
	for i := range dist {
		dist[i] = math.Inf(1)
	}
	dist[r.Source] = 0
	q := &queue{}
	heap.Init(q)
	metrics := core.WorkMetrics{WorkerCount: uint32(maxInt(1, r.Workers))}
	seq := uint64(1)
	var relax, pushes, pops, stale uint64
	maxQueue := 0
	consume := func(kind string) bool {
		if actionBudget != nil && metrics.TotalActions >= *actionBudget {
			return false
		}
		before := metrics.TotalActions
		metrics.AddAction(kind)
		metrics.LogicalSteps++
		metrics.ScheduledSteps++
		if bearing.Wants(o, "action") {
			o.Observe(bearing.Event{TaskID: "anchor-primary", Component: "ANCHOR", Kind: "action", Action: kind, Phase: "anchor_discrete_anytime", LogicalStep: metrics.LogicalSteps, ScheduledStep: metrics.ScheduledSteps, WorkBefore: before, WorkAfter: metrics.TotalActions})
		}
		return true
	}
	emit := func(kind string, attrs map[string]any) {
		o.Observe(bearing.Event{TaskID: "anchor-primary", Component: "ANCHOR", Kind: kind, Phase: "anchor_discrete_anytime", LogicalStep: metrics.LogicalSteps, ScheduledStep: metrics.ScheduledSteps, WorkAfter: metrics.TotalActions, Attributes: attrs})
	}
	emit("search_started", map[string]any{"initial_expand_limit": initialExpandLimit, "max_expand_limit": maxExpandLimit, "action_budget": actionBudget, "work_definition": "discrete_semantic_actions_v1"})
	minRatio := math.Inf(1)
	for u := 0; u < n && u < 1024; u++ {
		pu, ok := g.Position(core.NodeID(u))
		if !ok {
			continue
		}
		for _, e := range g.EdgesFrom(core.NodeID(u)) {
			pv, ok := g.Position(e.To)
			if !ok {
				continue
			}
			d := core.Euclidean(pu, pv)
			if d > 1e-12 && e.Weight/d < minRatio {
				minRatio = e.Weight / d
			}
		}
	}
	if math.IsInf(minRatio, 1) {
		minRatio = 0
	}
	h := func(v core.NodeID) float64 {
		a, ok1 := g.Position(v)
		z, ok2 := g.Position(r.Target)
		if ok1 && ok2 {
			return minRatio * core.Euclidean(a, z)
		}
		return 0
	}
	if !consume("enqueue") {
		return core.RouteResult{Distance: math.Inf(1), SolverName: "anchor", Work: metrics, BudgetExhausted: true, ErrorCode: core.ErrBudgetExhausted}
	}
	heap.Push(q, qitem{n: r.Source, pri: 4.0 * h(r.Source)})
	pushes++
	found := false
	exhausted := false
	currentExpandLimit := initialExpandLimit
	stageIndex := 0
	connectorCalls := 0
	bestH := h(r.Source)
	lastProgressExpand := uint64(0)
	lastProgressNode := r.Source
	strategy := Analyze(g)
	var connectedPath []core.NodeID
	connectedDistance := math.Inf(1)
	for q.Len() > 0 {
		select {
		case <-ctx.Done():
			return core.RouteResult{Distance: math.Inf(1), SolverName: "anchor", Work: metrics, ErrorCode: core.ErrCancelled}
		default:
		}
		if metrics.ExpandActions >= currentExpandLimit {
			if currentExpandLimit < maxExpandLimit {
				stageIndex++
				next := maxExpandLimit
				if stageIndex-1 < len(cfg.ExtensionExpandRatios) {
					next = uint64(math.Ceil(float64(g.NodeCount()) * cfg.ExtensionExpandRatios[stageIndex-1]))
					if next > maxExpandLimit {
						next = maxExpandLimit
					}
				}
				if next <= currentExpandLimit {
					next = maxExpandLimit
				}
				emit("budget_extended", map[string]any{"from_expand": currentExpandLimit, "to_expand": next, "stage": stageIndex})
				currentExpandLimit = next
			} else {
				exhausted = true
				break
			}
		}
		if !consume("select") {
			exhausted = true
			break
		}
		it := heap.Pop(q).(qitem)
		pops++
		u := it.n
		if bearing.Wants(o, "state_delta") {
			emit("frontier_selected", map[string]any{"node": u, "priority": it.pri, "frontier_size": q.Len()})
		}
		if settled[u] {
			stale++
			if !consume("reject") {
				exhausted = true
				break
			}
			continue
		}
		if !consume("expand") {
			exhausted = true
			break
		}
		settled[u] = true
		uniqueExpanded[u] = true
		emit("node_expanded", map[string]any{"node": u, "distance": dist[u], "frontier_size": q.Len()})
		uh := h(u)
		if uh+1e-12 < bestH {
			bestH = uh
			lastProgressExpand = metrics.ExpandActions
			lastProgressNode = u
		}
		stalled := cfg.StallWindow > 0 && metrics.ExpandActions-lastProgressExpand >= cfg.StallWindow
		if stalled && connector != nil && connectorCalls < cfg.MaxConnectorCalls {
			connectorCalls++
			// Connector checkpoints are deliberately far-reaching. A local shortest
			// path inside the same trap is not useful, so weighted/non-geometric
			// graphs retry from the source first; geometric graphs retry from the
			// last point that made measurable progress.
			checkpoints := []core.NodeID{lastProgressNode, u, r.Source}
			if strategy == "weighted_cost" || strategy == "portal" || strategy == "hub_aware" {
				checkpoints = []core.NodeID{r.Source, lastProgressNode, u}
			}
			from := checkpoints[(connectorCalls-1)%len(checkpoints)]
			prefix := reconstruct(prev, hasPrev, r.Source, from)
			if from == r.Source {
				prefix = []core.NodeID{r.Source}
			}
			connectorCap := uint64(math.Ceil(float64(g.NodeCount()) * cfg.ConnectorExpandRatio))
			if connectorCap < 1 {
				connectorCap = 1
			}
			remainingWork := actionBudget
			if actionBudget != nil {
				v := uint64(0)
				if metrics.TotalActions < *actionBudget {
					v = *actionBudget - metrics.TotalActions
				}
				remainingWork = &v
			}
			emit("connector_started", map[string]any{"from": from, "current": u, "to": r.Target, "expand_budget": connectorCap, "call": connectorCalls, "strategy": strategy})
			cr := connector.Solve(ctx, g, core.RouteRequest{Source: from, Target: r.Target, Mode: r.Mode, Workers: maxInt(1, r.Workers), Seed: r.Seed}, core.WorkBudget{MaxWork: remainingWork, MaxExpand: &connectorCap}, o)
			metrics.Add(cr.Work)
			relax += cr.WorkRelaxations
			pushes += cr.QueuePushes
			pops += cr.QueuePops
			if cr.Found && len(prefix) > 0 && len(cr.Path) > 0 {
				connectedPath = append(append([]core.NodeID{}, prefix...), cr.Path[1:]...)
				prefixDistance := 0.0
				if from != r.Source {
					prefixDistance = dist[from]
				}
				connectedDistance = prefixDistance + cr.Distance
				found = true
				_ = consume("connect")
				_ = consume("candidate")
				emit("connector_succeeded", map[string]any{"from": from, "distance": cr.Distance, "full_distance": connectedDistance, "call": connectorCalls})
				break
			}
			emit("connector_failed", map[string]any{"from": from, "call": connectorCalls, "budget_exhausted": cr.BudgetExhausted})
			lastProgressExpand = metrics.ExpandActions
		}

		if u == r.Target {
			found = true
			if !consume("candidate") {
				exhausted = true
			}
			emit("incumbent_updated", map[string]any{"distance": dist[u], "first_path_work": metrics.TotalActions, "first_path_expand": metrics.ExpandActions, "path": reconstruct(prev, hasPrev, r.Source, r.Target)})
			break
		}
		for _, e := range g.EdgesFrom(u) {
			evaluatedEdges[[2]core.NodeID{u, e.To}] = struct{}{}
			if bearing.Wants(o, "state_delta") {
				emit("edge_evaluated", map[string]any{"from": u, "to": e.To, "weight": e.Weight})
			}
			if !consume("evaluate") || !consume("relax") {
				exhausted = true
				break
			}
			relax++
			nd := dist[u] + e.Weight
			oldDistance := dist[e.To]
			if nd < dist[e.To] {
				dist[e.To] = nd
				if bearing.Wants(o, "state_delta") {
					emit("relaxation", map[string]any{"from": u, "to": e.To, "old_distance": oldDistance, "new_distance": nd, "accepted": true})
				}
				prev[e.To] = u
				hasPrev[e.To] = true
				if !consume("enqueue") {
					exhausted = true
					break
				}
				heap.Push(q, qitem{n: e.To, pri: nd + 4.0*h(e.To), dist: nd, seq: seq})
				seq++
				pushes++
				if q.Len() > maxQueue {
					maxQueue = q.Len()
				}
			} else {
				if bearing.Wants(o, "state_delta") {
					emit("relaxation", map[string]any{"from": u, "to": e.To, "old_distance": oldDistance, "new_distance": nd, "accepted": false})
				}
				if !consume("reject") {
					exhausted = true
					break
				}
			}
		}
		if exhausted {
			break
		}
	}
	_ = consume("terminate")
	path := reconstruct(prev, hasPrev, r.Source, r.Target)
	if len(connectedPath) > 0 {
		path = connectedPath
		dist[r.Target] = connectedDistance
	}
	if len(path) > 0 {
		found = true
	}
	var fp *uint64
	if found {
		v := metrics.TotalActions
		fp = &v
	}
	emit("search_finished", map[string]any{"found": found, "work": metrics.TotalActions, "expand": metrics.ExpandActions, "relax": relax, "queue_pushes": pushes, "queue_pops": pops, "stale_pops": stale, "max_queue": maxQueue, "budget_exhausted": exhausted})
	return core.RouteResult{Path: path, Distance: dist[r.Target], Found: found, Exact: false, SolverName: "anchor", Work: metrics, WorkRelaxations: relax, WorkExpandedNodes: metrics.ExpandActions, QueuePushes: pushes, QueuePops: pops, ParallelSteps: metrics.ScheduledSteps, TimeMS: float64(time.Since(start).Microseconds()) / 1000, FirstPathWork: fp, BudgetExhausted: exhausted, Telemetry: map[string]any{"initial_expand_limit": initialExpandLimit, "max_expand_limit": maxExpandLimit, "connector_calls": connectorCalls, "stale_pops": stale, "max_queue": maxQueue, "work_definition": "discrete_semantic_actions_v1", "investigated_nodes": countTrue(uniqueExpanded), "investigated_node_ratio": float64(countTrue(uniqueExpanded)) / float64(maxInt(1, n)), "investigated_edges": len(evaluatedEdges), "investigated_edge_ratio": float64(len(evaluatedEdges)) / float64(maxInt(1, edgeSlots(g))), "investigated_node_ids": trueNodeIDs(uniqueExpanded), "investigated_edge_ids": edgeIDs(evaluatedEdges), "candidate_paths": metrics.CandidateActions, "path_node_count": len(path)}}
}

func edgeSlots(g core.Graph) int {
	n := 0
	for i := 0; i < g.NodeCount(); i++ {
		n += len(g.EdgesFrom(core.NodeID(i)))
	}
	return n
}

func countTrue(v []bool) int {
	n := 0
	for _, x := range v {
		if x {
			n++
		}
	}
	return n
}

type qitem struct {
	n         core.NodeID
	pri, dist float64
	seq       uint64
}
type queue []qitem

func (q queue) Len() int { return len(q) }
func (q queue) Less(i, j int) bool {
	if q[i].pri != q[j].pri {
		return q[i].pri < q[j].pri
	}
	if q[i].dist != q[j].dist {
		return q[i].dist < q[j].dist
	}
	if q[i].seq != q[j].seq {
		return q[i].seq < q[j].seq
	}
	return q[i].n < q[j].n
}
func (q queue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q *queue) Push(x any)   { *q = append(*q, x.(qitem)) }
func (q *queue) Pop() any     { a := *q; x := a[len(a)-1]; *q = a[:len(a)-1]; return x }

func boundedSearch(ctx context.Context, g core.Graph, r core.RouteRequest, cap uint64, weight float64, allowed []bool, name string, o bearing.Observer) core.RouteResult {
	n := g.NodeCount()
	dist := make([]float64, n)
	prev := make([]core.NodeID, n)
	hp := make([]bool, n)
	settled := make([]bool, n)
	for i := range dist {
		dist[i] = math.Inf(1)
	}
	dist[r.Source] = 0
	minRatio := 0.0
	if _, ok := g.Position(r.Target); ok {
		vals := []float64{}
		for u := 0; u < n && len(vals) < 512; u++ {
			pu, ok := g.Position(core.NodeID(u))
			if !ok {
				continue
			}
			for _, e := range g.EdgesFrom(core.NodeID(u)) {
				pv, ok := g.Position(e.To)
				if !ok {
					continue
				}
				d := core.Euclidean(pu, pv)
				if d > 1e-12 {
					vals = append(vals, e.Weight/d)
				}
				if len(vals) >= 512 {
					break
				}
			}
		}
		if len(vals) > 0 {
			minRatio = vals[0]
			for _, v := range vals[1:] {
				if v < minRatio {
					minRatio = v
				}
			}
			minRatio *= .75
		}
	}
	h := func(v core.NodeID) float64 {
		a, ok1 := g.Position(v)
		z, ok2 := g.Position(r.Target)
		if ok1 && ok2 {
			return minRatio * core.Euclidean(a, z)
		}
		return 0
	}
	q := &queue{}
	heap.Init(q)
	metrics := core.WorkMetrics{WorkerCount: uint32(maxInt(1, r.Workers))}
	emit := func(kind string, attrs map[string]any) {
		o.Observe(bearing.Event{TaskID: name, Component: "ANCHOR", Kind: kind, Phase: name, LogicalStep: metrics.LogicalSteps, ScheduledStep: metrics.ScheduledSteps, WorkAfter: metrics.TotalActions, Attributes: attrs})
	}
	consume := func(kind string) bool {
		if metrics.TotalActions >= cap {
			return false
		}
		before := metrics.TotalActions
		metrics.AddAction(kind)
		metrics.LogicalSteps++
		metrics.ScheduledSteps++
		if bearing.Wants(o, "action") {
			o.Observe(bearing.Event{TaskID: name, Component: "ANCHOR", Kind: "action", Action: kind, Phase: name, LogicalStep: metrics.LogicalSteps, ScheduledStep: metrics.ScheduledSteps, WorkBefore: before, WorkAfter: metrics.TotalActions})
		}
		return true
	}
	if !consume("enqueue") {
		return core.RouteResult{Distance: math.Inf(1), SolverName: name, Work: metrics, BudgetExhausted: true, ErrorCode: core.ErrBudgetExhausted}
	}
	heap.Push(q, qitem{n: r.Source, pri: weight * h(r.Source)})
	seq := uint64(1)
	var exp, rel, push, pop uint64
	exhausted := false
	for q.Len() > 0 {
		select {
		case <-ctx.Done():
			return core.RouteResult{Distance: math.Inf(1), SolverName: name, Work: metrics, ErrorCode: core.ErrCancelled}
		default:
		}
		if !consume("select") {
			exhausted = true
			break
		}
		it := heap.Pop(q).(qitem)
		pop++
		u := it.n
		if settled[u] {
			if !consume("reject") {
				exhausted = true
				break
			}
			continue
		}
		if !consume("expand") {
			exhausted = true
			break
		}
		settled[u] = true
		exp++
		emit("node_expanded", map[string]any{"node": u, "distance": dist[u], "frontier_size": q.Len()})
		if u == r.Target {
			break
		}
		for _, e := range g.EdgesFrom(u) {
			if allowed != nil && !allowed[e.To] {
				if !consume("reject") {
					exhausted = true
					break
				}
				continue
			}
			if !consume("evaluate") || !consume("relax") {
				exhausted = true
				break
			}
			rel++
			nd := dist[u] + e.Weight
			oldDistance := dist[e.To]
			if nd < dist[e.To] {
				dist[e.To] = nd
				if bearing.Wants(o, "state_delta") {
					emit("relaxation", map[string]any{"from": u, "to": e.To, "old_distance": oldDistance, "new_distance": nd, "accepted": true})
				}
				prev[e.To] = u
				hp[e.To] = true
				if !consume("enqueue") {
					exhausted = true
					break
				}
				heap.Push(q, qitem{n: e.To, pri: nd + weight*h(e.To), dist: nd, seq: seq})
				seq++
				push++
			} else if !consume("reject") {
				exhausted = true
				break
			}
		}
		if exhausted {
			break
		}
	}
	if !consume("terminate") {
		exhausted = true
	}
	path := reconstruct(prev, hp, r.Source, r.Target)
	found := len(path) > 0
	return core.RouteResult{Path: path, Distance: dist[r.Target], Found: found, SolverName: name, Work: metrics, WorkRelaxations: rel, WorkExpandedNodes: exp, QueuePushes: push, QueuePops: pop, ParallelSteps: metrics.ScheduledSteps, BudgetExhausted: exhausted, Telemetry: map[string]any{"budget_cap": cap, "work_definition": "semantic_actions_v1"}}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func reconstruct(prev []core.NodeID, hp []bool, s, t core.NodeID) []core.NodeID {
	if s == t {
		return []core.NodeID{s}
	}
	if !hp[t] {
		return nil
	}
	out := []core.NodeID{t}
	for out[len(out)-1] != s {
		out = append(out, prev[out[len(out)-1]])
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
func max1(v uint64) uint64 {
	if v < 1 {
		return 1
	}
	return v
}
func better(a, b core.RouteResult) core.RouteResult {
	if !a.Found {
		return b
	}
	if !b.Found {
		return a
	}
	if b.Distance < a.Distance {
		return b
	}
	return a
}

func geometric(ctx context.Context, g core.Graph, r core.RouteRequest, budget uint64, cfg Config, o bearing.Observer) core.RouteResult {
	first := uint64(float64(g.NodeCount()) * cfg.InitialPathBudgetRatio)
	if first < 6 {
		first = 6
	}
	if first > budget {
		first = budget
	}
	best := boundedSearch(ctx, g, r, first, 2.4, nil, "dg6_geo_beam_first", o)
	agg := best.Work
	spent := best.TotalWork()
	firstWork := spent
	if !best.Found && spent < budget {
		x := boundedSearch(ctx, g, r, minU(first, budget-spent), cfg.WeightedAStarFactor, nil, "dg6_geo_first_path", o)
		best = better(best, x)
		agg.Add(x.Work)
		spent += x.TotalWork()
		firstWork = spent
	}
	if !best.Found && spent < budget {
		offsets := []float64{0, 1, -1, 2, -2}
		if cfg.MaxCorridors < len(offsets) {
			offsets = offsets[:cfg.MaxCorridors]
		}
		for i, off := range offsets {
			allowed := corridor(g, r.Source, r.Target, cfg.BaseWidthScale*1.45, off, float64(i))
			x := boundedSearch(ctx, g, r, minU(first, budget-spent), 1, allowed, "dg6_geo_corridor_first", o)
			agg.Add(x.Work)
			spent += x.TotalWork()
			best = better(best, x)
			if best.Found {
				firstWork = spent
				break
			}
			if spent >= budget {
				break
			}
		}
	}
	if best.Found && spent < budget && budget-spent >= uint64(math.Max(4, float64(g.NodeCount())*cfg.MinQualityBudgetRatio)) {
		x := repair(ctx, g, r, best, budget-spent, cfg, o)
		agg.Add(x.Work)
		spent += x.TotalWork()
		best = better(best, x)
	}
	best.Work = agg
	best.ParallelSteps = agg.ScheduledSteps
	fw := firstWork
	best.FirstPathWork = &fw
	return best
}

func corridor(g core.Graph, s, t core.NodeID, scale, offsetFactor, index float64) []bool {
	a, ok1 := g.Position(s)
	b, ok2 := g.Position(t)
	if !ok1 || !ok2 {
		return nil
	}
	vx, vy := b.X-a.X, b.Y-a.Y
	base := math.Hypot(vx, vy)
	if base < 1e-12 {
		return nil
	}
	nx, ny := -vy/base, vx/base
	off := offsetFactor * base * scale
	width := base * scale * (1 + .18*index)
	den := vx*vx + vy*vy
	allowed := make([]bool, g.NodeCount())
	for i := 0; i < g.NodeCount(); i++ {
		p, ok := g.Position(core.NodeID(i))
		if !ok {
			continue
		}
		sx, sy := a.X+nx*off, a.Y+ny*off
		tt := ((p.X-sx)*vx + (p.Y-sy)*vy) / den
		if tt >= -.12 && tt <= 1.12 {
			px, py := sx+tt*vx, sy+tt*vy
			if math.Hypot(p.X-px, p.Y-py) <= width {
				allowed[i] = true
			}
		}
	}
	allowed[s] = true
	allowed[t] = true
	return allowed
}

func hubAware(ctx context.Context, g core.Graph, r core.RouteRequest, budget uint64, cfg Config, o bearing.Observer) core.RouteResult {
	ids := make([]core.NodeID, g.NodeCount())
	for i := range ids {
		ids[i] = core.NodeID(i)
	}
	sort.Slice(ids, func(i, j int) bool {
		di, dj := len(g.EdgesFrom(ids[i])), len(g.EdgesFrom(ids[j]))
		if di != dj {
			return di > dj
		}
		return ids[i] < ids[j]
	})
	hc := cfg.HubCount
	if hc < 2 {
		hc = 2
	}
	if hc > len(ids) {
		hc = len(ids)
	}
	seeds := append([]core.NodeID{r.Source, r.Target}, ids[:hc]...)
	var best core.RouteResult
	var agg core.WorkMetrics
	spent := uint64(0)
	for _, hops := range []int{2, 3, 4, 5} {
		if spent >= budget {
			break
		}
		capNodes := int(math.Min(float64(g.NodeCount())*math.Min(.20+.08*float64(hops), .65), math.Max(16, float64(budget*3))))
		allowed := expand(g, seeds, hops, capNodes)
		per := uint64(float64(g.NodeCount()) * cfg.ConnectorBudgetRatio)
		if per < 6 {
			per = 6
		}
		per = minU(per, budget-spent)
		x := boundedSearch(ctx, g, r, per, 1, allowed, "dg6_hub_first", o)
		agg.Add(x.Work)
		spent += x.TotalWork()
		best = better(best, x)
		if best.Found {
			fw := spent
			best.FirstPathWork = &fw
			break
		}
	}
	best.Work = agg
	best.ParallelSteps = agg.ScheduledSteps
	return best
}
func expand(g core.Graph, seeds []core.NodeID, hops, capN int) []bool {
	seen := make([]bool, g.NodeCount())
	type nd struct {
		n core.NodeID
		d int
	}
	q := []nd{}
	count := 0
	for _, s := range seeds {
		if !seen[s] {
			seen[s] = true
			q = append(q, nd{s, 0})
			count++
		}
	}
	for len(q) > 0 && count < capN {
		x := q[0]
		q = q[1:]
		if x.d >= hops {
			continue
		}
		for _, e := range g.EdgesFrom(x.n) {
			if !seen[e.To] {
				seen[e.To] = true
				count++
				q = append(q, nd{e.To, x.d + 1})
				if count >= capN {
					break
				}
			}
		}
	}
	return seen
}
func portal(ctx context.Context, g core.Graph, r core.RouteRequest, budget uint64, cfg Config, o bearing.Observer) core.RouteResult { // Long-edge skeleton followed by corridor fallback.
	type ep struct {
		u, v core.NodeID
		l    float64
	}
	arr := []ep{}
	for u := 0; u < g.NodeCount(); u++ {
		pu, ok := g.Position(core.NodeID(u))
		if !ok {
			continue
		}
		for _, e := range g.EdgesFrom(core.NodeID(u)) {
			pv, ok := g.Position(e.To)
			if ok {
				arr = append(arr, ep{core.NodeID(u), e.To, core.Euclidean(pu, pv)})
			}
		}
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].l > arr[j].l })
	seeds := []core.NodeID{r.Source, r.Target}
	for i := 0; i < len(arr) && i < 40; i++ {
		seeds = append(seeds, arr[i].u, arr[i].v)
	}
	allowed := expand(g, seeds, 1, minI(g.NodeCount(), int(math.Max(16, float64(budget)*3))))
	first := uint64(math.Max(8, float64(g.NodeCount())*math.Max(cfg.ConnectorBudgetRatio, .28)))
	first = minU(first, budget)
	best := boundedSearch(ctx, g, r, first, 1, allowed, "dg6_portal_long_edge_skeleton", o)
	agg := best.Work
	spent := best.TotalWork()
	if !best.Found && spent < budget {
		x := geometric(ctx, g, r, budget-spent, cfg, o)
		agg.Add(x.Work)
		spent += x.TotalWork()
		best = better(best, x)
	}
	best.Work = agg
	best.ParallelSteps = agg.ScheduledSteps
	return best
}
func weightedCost(ctx context.Context, g core.Graph, r core.RouteRequest, budget uint64, cfg Config, o bearing.Observer) core.RouteResult {
	cap := uint64(math.Max(float64(g.NodeCount())*.72, float64(g.NodeCount())*cfg.InitialPathBudgetRatio*2))
	cap = minU(cap, budget)
	best := boundedSearch(ctx, g, r, cap, 1, nil, "dg6_weighted_bidir_first", o)
	agg := best.Work
	spent := best.TotalWork()
	if !best.Found && spent < budget {
		x := boundedSearch(ctx, g, r, budget-spent, cfg.WeightedAStarFactor, nil, "dg6_weighted_first", o)
		agg.Add(x.Work)
		spent += x.TotalWork()
		best = better(best, x)
	}
	if best.Found && spent < budget {
		x := repair(ctx, g, r, best, budget-spent, cfg, o)
		agg.Add(x.Work)
		spent += x.TotalWork()
		best = better(best, x)
	}
	best.Work = agg
	best.ParallelSteps = agg.ScheduledSteps
	return best
}
func repair(ctx context.Context, g core.Graph, r core.RouteRequest, best core.RouteResult, budget uint64, cfg Config, o bearing.Observer) core.RouteResult {
	if len(best.Path) == 0 {
		return core.RouteResult{Distance: math.Inf(1)}
	}
	capN := int(float64(g.NodeCount()) * cfg.MaxRepairNodesRatio)
	if capN < 8 {
		capN = 8
	}
	allowed := expand(g, best.Path, cfg.RepairHops, capN)
	return boundedSearch(ctx, g, r, budget, 1, allowed, "dg6_local_repair", o)
}
func minU(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
func minI(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func trueNodeIDs(v []bool) []uint64 {
	out := make([]uint64, 0)
	for i, ok := range v {
		if ok {
			out = append(out, uint64(i))
		}
	}
	return out
}
func edgeIDs(v map[[2]core.NodeID]struct{}) []uint64 {
	out := make([]uint64, 0, len(v))
	for e := range v {
		out = append(out, (uint64(e[0])<<32)|uint64(e[1]))
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func finalizeTiming(result core.RouteResult, component string) core.RouteResult {
	if result.TimeBreakdown.TotalMS == 0 {
		result.TimeBreakdown.TotalMS = result.TimeMS
		result.TimeBreakdown.SolverMS = result.TimeMS
		if component == "anchor" {
			result.TimeBreakdown.AnchorMS = result.TimeMS
		}
		if component == "bolts" {
			result.TimeBreakdown.BoltsMS = result.TimeMS
		}
	}
	return result
}
