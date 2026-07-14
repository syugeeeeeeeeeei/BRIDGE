package bolts

import (
	"container/heap"
	"context"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"math"
	"sort"
	"time"
)

type Solver interface {
	Name() string
	Solve(context.Context, core.Graph, core.RouteRequest, core.WorkBudget, bearing.Observer) core.RouteResult
}
type item struct {
	node               core.NodeID
	priority, distance float64
	sequence           uint64
	index              int
}
type pq []*item

func (p pq) Len() int { return len(p) }
func (p pq) Less(i, j int) bool {
	if p[i].priority != p[j].priority {
		return p[i].priority < p[j].priority
	}
	if p[i].distance != p[j].distance {
		return p[i].distance < p[j].distance
	}
	if p[i].sequence != p[j].sequence {
		return p[i].sequence < p[j].sequence
	}
	return p[i].node < p[j].node
}
func (p pq) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p *pq) Push(x any)   { *p = append(*p, x.(*item)) }
func (p *pq) Pop() any     { a := *p; n := len(a); x := a[n-1]; *p = a[:n-1]; return x }
func reconstruct(prev []core.NodeID, has []bool, s, t core.NodeID) []core.NodeID {
	if s == t {
		return []core.NodeID{s}
	}
	if !has[t] {
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

type SeedState struct {
	Dist     []float64
	Prev     []core.NodeID
	HasPrev  []bool
	Settled  []bool
	Frontier []core.NodeID
}

type SeededWeightedAStar struct {
	Weight      float64
	Seed        SeedState
	RequireSeed bool
}

func (s SeededWeightedAStar) Name() string { return "seeded_weighted_astar" }
func (s SeededWeightedAStar) Solve(ctx context.Context, g core.Graph, r core.RouteRequest, b core.WorkBudget, o bearing.Observer) core.RouteResult {
	w := s.Weight
	if w <= 1 {
		w = 1.12
	}
	return seededSearch(ctx, g, r, b, o, s.Name(), w, s.Seed, s.RequireSeed)
}

func seededSearch(ctx context.Context, g core.Graph, r core.RouteRequest, b core.WorkBudget, o bearing.Observer, name string, hWeight float64, seed SeedState, requireSeed bool) core.RouteResult {
	start := time.Now()
	n := g.NodeCount()
	dist := make([]float64, n)
	prev := make([]core.NodeID, n)
	hasPrev := make([]bool, n)
	settled := make([]bool, n)
	for i := range dist {
		dist[i] = math.Inf(1)
	}
	if len(seed.Dist) == n {
		copy(dist, seed.Dist)
	}
	if len(seed.Prev) == n {
		copy(prev, seed.Prev)
	}
	if len(seed.HasPrev) == n {
		copy(hasPrev, seed.HasPrev)
	}
	if math.IsInf(dist[r.Source], 1) {
		dist[r.Source] = 0
	}
	q := &pq{}
	heap.Init(q)
	metrics := core.WorkMetrics{WorkerCount: uint32(maxInt(1, r.Workers))}
	consume := func(kind string) bool {
		if b.MaxWork != nil && metrics.TotalActions >= *b.MaxWork {
			return false
		}
		metrics.AddAction(kind)
		metrics.LogicalSteps++
		metrics.ScheduledSteps++
		return true
	}
	heuristicUnitScale := graphHeuristicUnitScale(g)
	h := func(v core.NodeID) float64 {
		a, ok1 := g.Position(v)
		z, ok2 := g.Position(r.Target)
		if ok1 && ok2 {
			return heuristicUnitScale * core.Euclidean(a, z) * hWeight
		}
		return 0
	}
	seq := uint64(0)
	seedQueued := uint64(0)
	seedExpanded := uint64(0)
	seedSet := make(map[core.NodeID]struct{}, len(seed.Frontier))
	for _, v := range seed.Frontier {
		seedSet[v] = struct{}{}
		if int(v) >= n || math.IsInf(dist[v], 1) {
			continue
		}
		if !consume("enqueue") {
			break
		}
		heap.Push(q, &item{node: v, priority: dist[v] + h(v), distance: dist[v], sequence: seq})
		seq++
		seedQueued++
	}
	if q.Len() == 0 {
		if requireSeed {
			return core.RouteResult{Distance: math.Inf(1), SolverName: name, Work: metrics, TerminationStatus: core.TerminationUnreachable, Telemetry: map[string]any{"seed_frontier_count": uint64(len(seed.Frontier)), "seed_queued_count": seedQueued, "seed_expanded_count": seedExpanded, "seed_required": true}}
		}
		if !consume("enqueue") {
			return core.RouteResult{Distance: math.Inf(1), SolverName: name, Work: metrics, BudgetExhausted: true}
		}
		heap.Push(q, &item{node: r.Source, priority: 0, distance: 0})
	}
	found := false
	exhausted := false
	for q.Len() > 0 {
		select {
		case <-ctx.Done():
			return core.RouteResult{Distance: math.Inf(1), SolverName: name, Work: metrics, ErrorCode: core.ErrCancelled, TimeMS: float64(time.Since(start).Nanoseconds()) / 1e6}
		default:
		}
		if !consume("select") {
			exhausted = true
			break
		}
		it := heap.Pop(q).(*item)
		u := it.node
		if it.distance != dist[u] {
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
		if _, ok := seedSet[u]; ok {
			seedExpanded++
		}
		if u == r.Target {
			found = true
			break
		}
		for _, e := range g.EdgesFrom(u) {
			if !consume("evaluate") {
				exhausted = true
				break
			}
			if !consume("relax") {
				exhausted = true
				break
			}
			nd := dist[u] + e.Weight
			if nd < dist[e.To] {
				dist[e.To] = nd
				prev[e.To] = u
				hasPrev[e.To] = true
				settled[e.To] = false
				if !consume("enqueue") {
					exhausted = true
					break
				}
				heap.Push(q, &item{node: e.To, priority: nd + h(e.To), distance: nd, sequence: seq})
				seq++
			} else {
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
	if !consume("terminate") {
		exhausted = true
	}
	path := reconstruct(prev, hasPrev, r.Source, r.Target)
	if found && len(path) == 0 {
		found = false
	}
	d := math.Inf(1)
	if found {
		d = dist[r.Target]
	}
	status := core.TerminationUnreachable
	if exhausted {
		status = core.TerminationUnknownBudget
	} else if found {
		status = core.TerminationFound
	}
	seedPathContribution := uint64(0)
	for _, v := range path {
		if _, ok := seedSet[v]; ok {
			seedPathContribution++
		}
	}
	return core.RouteResult{Path: path, Distance: d, Found: found, SolverName: name, Work: metrics, TimeMS: float64(time.Since(start).Nanoseconds()) / 1e6, TerminationStatus: status, BudgetExhausted: exhausted, Telemetry: map[string]any{"seed_frontier_count": uint64(len(seed.Frontier)), "seed_queued_count": seedQueued, "seed_expanded_count": seedExpanded, "seed_path_contribution_count": seedPathContribution, "seed_required": requireSeed}}
}

type Dijkstra struct{}

func (Dijkstra) Name() string { return "dijkstra" }
func (Dijkstra) Solve(ctx context.Context, g core.Graph, r core.RouteRequest, b core.WorkBudget, o bearing.Observer) core.RouteResult {
	return search(ctx, g, r, b, o, "dijkstra", 1, false)
}

type AStar struct{}

func (AStar) Name() string { return "astar" }
func (AStar) Solve(ctx context.Context, g core.Graph, r core.RouteRequest, b core.WorkBudget, o bearing.Observer) core.RouteResult {
	return search(ctx, g, r, b, o, "astar", 1, true)
}

type WeightedAStar struct {
	Weight float64
	ID     string
}

func (s WeightedAStar) Name() string {
	if s.ID != "" {
		return s.ID
	}
	return "emergency_approx"
}
func (s WeightedAStar) Solve(ctx context.Context, g core.Graph, r core.RouteRequest, b core.WorkBudget, o bearing.Observer) core.RouteResult {
	w := s.Weight
	if w <= 1 {
		w = 1.12
	}
	return search(ctx, g, r, b, o, s.Name(), w, true)
}
func search(ctx context.Context, g core.Graph, r core.RouteRequest, b core.WorkBudget, o bearing.Observer, name string, hWeight float64, useH bool) core.RouteResult {
	start := time.Now()
	n := g.NodeCount()
	dist := make([]float64, n)
	prev := make([]core.NodeID, n)
	hasPrev := make([]bool, n)
	settled := make([]bool, n)
	collectDiagnostics := bearing.Wants(o, "state_delta") || bearing.Wants(o, "debug")
	var evaluatedEdges map[[2]core.NodeID]struct{}
	if collectDiagnostics {
		evaluatedEdges = make(map[[2]core.NodeID]struct{})
	}
	for i := range dist {
		dist[i] = math.Inf(1)
	}
	dist[r.Source] = 0
	q := &pq{}
	heap.Init(q)
	metrics := core.WorkMetrics{WorkerCount: uint32(maxInt(1, r.Workers))}
	o.Observe(bearing.Event{TaskID: name, Kind: "search_started", Phase: name, Attributes: map[string]any{"action_budget": b.MaxWork, "expand_budget": b.MaxExpand, "work_definition": "work_model_v2"}})
	emit := func(kind string, attrs map[string]any) {
		o.Observe(bearing.Event{TaskID: name, Component: "BOLTS", Kind: kind, Phase: name, LogicalStep: metrics.LogicalSteps, ScheduledStep: metrics.ScheduledSteps, WorkAfter: metrics.TotalActions, Attributes: attrs})
	}
	consume := func(kind string) bool {
		if b.MaxWork != nil && metrics.TotalActions >= *b.MaxWork {
			return false
		}
		before := metrics.TotalActions
		metrics.AddAction(kind)
		// Current BOLTS implementation is sequential: every action is one step.
		metrics.LogicalSteps++
		metrics.ScheduledSteps++
		if bearing.Wants(o, "action") {
			o.Observe(bearing.Event{TaskID: name, Component: "BOLTS", Kind: "action", Action: kind, Phase: name, LogicalStep: metrics.LogicalSteps, ScheduledStep: metrics.ScheduledSteps, WorkBefore: before, WorkAfter: metrics.TotalActions})
		}
		return true
	}
	if !consume("enqueue") {
		return core.RouteResult{Distance: math.Inf(1), SolverName: name, Work: metrics, BudgetExhausted: true, ErrorCode: core.ErrBudgetExhausted}
	}
	heap.Push(q, &item{node: r.Source, priority: 0, distance: 0})
	if bearing.Wants(o, "state_delta") {
		emit("frontier_enqueued", map[string]any{"node": r.Source, "priority": 0.0, "reason": "source"})
	}
	seq := uint64(1)
	var exp, rel, push, pop uint64
	push = 1
	exhausted := false
	heuristicUnitScale := graphHeuristicUnitScale(g)
	h := func(v core.NodeID) float64 {
		if !useH {
			return 0
		}
		a, ok1 := g.Position(v)
		z, ok2 := g.Position(r.Target)
		if ok1 && ok2 {
			return heuristicUnitScale * core.Euclidean(a, z) * hWeight
		}
		return 0
	}
	found := false
	for q.Len() > 0 {
		select {
		case <-ctx.Done():
			return core.RouteResult{Distance: math.Inf(1), SolverName: name, Work: metrics, ErrorCode: core.ErrCancelled, TimeMS: float64(time.Since(start).Nanoseconds()) / 1_000_000}
		default:
		}
		if !consume("select") {
			exhausted = true
			break
		}
		it := heap.Pop(q).(*item)
		pop++
		u := it.node
		if bearing.Wants(o, "state_delta") {
			emit("frontier_selected", map[string]any{"node": u, "priority": it.priority, "frontier_size": q.Len()})
		}
		if it.distance != dist[u] {
			if !consume("reject") {
				exhausted = true
				break
			}
			continue
		}
		if b.MaxExpand != nil && metrics.ExpandActions >= *b.MaxExpand {
			exhausted = true
			break
		}
		if !consume("expand") {
			exhausted = true
			break
		}
		settled[u] = true
		exp++
		emit("node_expanded", map[string]any{"node": u, "distance": dist[u], "frontier_size": q.Len()})
		if u == r.Target {
			found = true
			break
		}
		for _, e := range g.EdgesFrom(u) {
			if collectDiagnostics {
				evaluatedEdges[[2]core.NodeID{u, e.To}] = struct{}{}
			}
			if bearing.Wants(o, "state_delta") {
				emit("edge_evaluated", map[string]any{"from": u, "to": e.To, "weight": e.Weight})
			}
			if !consume("evaluate") {
				exhausted = true
				break
			}
			if !consume("relax") {
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
				hasPrev[e.To] = true
				settled[e.To] = false
				if !consume("enqueue") {
					exhausted = true
					break
				}
				priority := nd + h(e.To)
				heap.Push(q, &item{node: e.To, priority: priority, distance: nd, sequence: seq})
				seq++
				push++
				if bearing.Wants(o, "state_delta") {
					emit("frontier_enqueued", map[string]any{"node": e.To, "from": u, "priority": priority, "distance": nd})
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
	if consume("terminate") == false {
		exhausted = true
	}
	path := reconstruct(prev, hasPrev, r.Source, r.Target)
	if r.Source == r.Target {
		path = []core.NodeID{r.Source}
		found = true
	}
	if len(path) > 0 {
		found = true
	}
	exact := !useH && !exhausted && (found || q.Len() == 0)
	if useH && hWeight == 1 && !exhausted {
		exact = true
	}
	emit("search_finished", map[string]any{"found": found, "work": metrics.TotalActions, "expand": metrics.ExpandActions, "relax": rel, "queue_pushes": push, "queue_pops": pop, "budget_exhausted": exhausted, "path": path})
	res := core.RouteResult{Path: path, Distance: dist[r.Target], Found: found, Exact: exact, SolverName: name, Work: metrics, WorkRelaxations: rel, WorkExpandedNodes: exp, QueuePushes: push, QueuePops: pop, ParallelSteps: metrics.ScheduledSteps, TimeMS: float64(time.Since(start).Nanoseconds()) / 1_000_000, BudgetExhausted: exhausted, Telemetry: map[string]any{"budget_exhausted": exhausted, "work_definition": "work_model_v2", "investigated_nodes": metrics.ExpandActions, "investigated_node_ratio": float64(metrics.ExpandActions) / float64(maxInt(1, n)), "investigated_edges": len(evaluatedEdges), "investigated_edge_ratio": float64(len(evaluatedEdges)) / float64(maxInt(1, edgeSlots(g))), "investigated_node_ids": boolNodeIDs(settled), "investigated_edge_ids": encodedEdgeIDs(evaluatedEdges), "candidate_paths": func() uint64 {
		if found {
			return 1
		}
		return 0
	}(), "path_node_count": len(path)}}
	if exact && found {
		x := 1.0
		res.LowerBound = res.Distance
		res.CertifiedRatio = &x
		res.QualityCertified = true
	}
	res.SearchCompleted = !exhausted
	switch {
	case exhausted:
		res.TerminationStatus = core.TerminationUnknownBudget
		res.ErrorCode = core.ErrBudgetExhausted
	case found:
		res.TerminationStatus = core.TerminationFound
	case !found:
		res.TerminationStatus = core.TerminationUnreachable
		res.ErrorCode = core.ErrNoPath
		if exact {
			res.ReachabilityProven = true
		}
	}
	return finalizeTiming(res, "bolts")
}

func graphHeuristicUnitScale(g core.Graph) float64 {
	minScale := math.Inf(1)
	for u := 0; u < g.NodeCount(); u++ {
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
			if d <= 1e-12 {
				continue
			}
			scale := e.Weight / d
			if scale < minScale {
				minScale = scale
			}
		}
	}
	if math.IsInf(minScale, 1) {
		return 0
	}
	return minScale
}

func edgeSlots(g core.Graph) int {
	n := 0
	for i := 0; i < g.NodeCount(); i++ {
		n += len(g.EdgesFrom(core.NodeID(i)))
	}
	return n
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type Reachability struct{}

func (Reachability) Name() string { return "reachability" }
func (Reachability) Solve(ctx context.Context, g core.Graph, r core.RouteRequest, b core.WorkBudget, o bearing.Observer) core.RouteResult {
	started := time.Now()
	finish := func(result core.RouteResult) core.RouteResult {
		elapsedNS := time.Since(started).Nanoseconds()
		if elapsedNS <= 0 {
			elapsedNS = 1
		}
		result.TimeMS = float64(elapsedNS) / 1_000_000
		result.TimeBreakdown = core.TimeBreakdown{
			TotalNS: elapsedNS, SolverNS: elapsedNS, BoltsNS: elapsedNS,
			TotalMS: result.TimeMS, SolverMS: result.TimeMS, BoltsMS: result.TimeMS,
		}
		return result
	}

	seen := make([]bool, g.NodeCount())
	prev := make([]core.NodeID, g.NodeCount())
	hp := make([]bool, g.NodeCount())
	q := make([]core.NodeID, 1, g.NodeCount())
	q[0] = r.Source
	seen[r.Source] = true
	metrics := core.WorkMetrics{WorkerCount: uint32(maxInt(1, r.Workers))}
	consume := func(kind string) bool {
		if b.MaxWork != nil && metrics.TotalActions >= *b.MaxWork {
			return false
		}
		before := metrics.TotalActions
		metrics.AddAction(kind)
		metrics.LogicalSteps++
		metrics.ScheduledSteps++
		if bearing.Wants(o, "action") {
			o.Observe(bearing.Event{TaskID: "reachability", Component: "BOLTS", Kind: "action", Action: kind, Phase: "reachability", LogicalStep: metrics.LogicalSteps, ScheduledStep: metrics.ScheduledSteps, WorkBefore: before, WorkAfter: metrics.TotalActions})
		}
		return true
	}
	if !consume("enqueue") {
		return finish(core.RouteResult{Distance: math.Inf(1), SolverName: "reachability", Work: metrics, BudgetExhausted: true, TerminationStatus: core.TerminationUnknownBudget, ErrorCode: core.ErrBudgetExhausted})
	}
	exhausted := false
	cancelled := false
	for head := 0; head < len(q); head++ {
		select {
		case <-ctx.Done():
			cancelled = true
			exhausted = true
		default:
		}
		if cancelled {
			break
		}
		if !consume("select") || !consume("expand") {
			exhausted = true
			break
		}
		u := q[head]
		if u == r.Target {
			break
		}
		for _, e := range g.EdgesFrom(u) {
			if !consume("evaluate") {
				exhausted = true
				break
			}
			if !seen[e.To] {
				seen[e.To] = true
				prev[e.To] = u
				hp[e.To] = true
				if !consume("enqueue") {
					exhausted = true
					break
				}
				q = append(q, e.To)
			} else if !consume("reject") {
				exhausted = true
				break
			}
		}
		if exhausted {
			break
		}
	}
	if !cancelled && !consume("terminate") {
		exhausted = true
	}
	path := reconstruct(prev, hp, r.Source, r.Target)
	found := len(path) > 0
	distance := math.Inf(1)
	if found {
		distance = routePathDistance(g, path)
	}
	result := core.RouteResult{
		Path: path, Distance: distance, Found: found,
		// Reachability only certifies existence/non-existence. It never certifies
		// weighted shortest-path optimality.
		Exact: false, SolverName: "reachability", Work: metrics,
		WorkExpandedNodes: metrics.ExpandActions, ParallelSteps: metrics.ScheduledSteps,
		BudgetExhausted:    exhausted && !cancelled,
		SearchCompleted:    !exhausted,
		ReachabilityProven: found || !exhausted,
	}
	switch {
	case cancelled:
		result.TerminationStatus = core.TerminationCancelled
		result.ErrorCode = core.ErrCancelled
	case exhausted:
		result.TerminationStatus = core.TerminationUnknownBudget
		result.ErrorCode = core.ErrBudgetExhausted
	case found:
		result.TerminationStatus = core.TerminationFound
	default:
		result.TerminationStatus = core.TerminationUnreachable
		result.ErrorCode = core.ErrNoPath
	}
	return finish(result)
}

type BidirectionalDijkstra struct{}

func (BidirectionalDijkstra) Name() string { return "bidirectional_dijkstra" }

// Solve runs deterministic bidirectional Dijkstra. Forward and backward lanes
// are interleaved in a fixed order, so the current implementation is
// sequential and therefore ScheduledSteps == TotalActions.
func (BidirectionalDijkstra) Solve(ctx context.Context, g core.Graph, r core.RouteRequest, b core.WorkBudget, o bearing.Observer) core.RouteResult {
	start := time.Now()
	n := g.NodeCount()
	reverse := make([][]core.Edge, n)
	for u := 0; u < n; u++ {
		for _, e := range g.EdgesFrom(core.NodeID(u)) {
			reverse[e.To] = append(reverse[e.To], core.Edge{To: core.NodeID(u), Weight: e.Weight})
		}
	}
	for i := range reverse {
		sort.Slice(reverse[i], func(a, c int) bool {
			if reverse[i][a].To != reverse[i][c].To {
				return reverse[i][a].To < reverse[i][c].To
			}
			return reverse[i][a].Weight < reverse[i][c].Weight
		})
	}
	df, db := make([]float64, n), make([]float64, n)
	pf, pb := make([]core.NodeID, n), make([]core.NodeID, n)
	hf, hb := make([]bool, n), make([]bool, n)
	sf, sb := make([]bool, n), make([]bool, n)
	for i := 0; i < n; i++ {
		df[i], db[i] = math.Inf(1), math.Inf(1)
	}
	df[r.Source], db[r.Target] = 0, 0
	qf, qb := &pq{}, &pq{}
	heap.Init(qf)
	heap.Init(qb)
	metrics := core.WorkMetrics{WorkerCount: uint32(maxInt(1, r.Workers))}
	consume := func(kind string) bool {
		if b.MaxWork != nil && metrics.TotalActions >= *b.MaxWork {
			return false
		}
		before := metrics.TotalActions
		metrics.AddAction(kind)
		metrics.LogicalSteps++
		metrics.ScheduledSteps++
		if bearing.Wants(o, "action") {
			o.Observe(bearing.Event{TaskID: "bidirectional_dijkstra", Component: "BOLTS", Kind: "action", Action: kind, Phase: "bidirectional_dijkstra", LogicalStep: metrics.LogicalSteps, ScheduledStep: metrics.ScheduledSteps, WorkBefore: before, WorkAfter: metrics.TotalActions})
		}
		return true
	}
	emit := func(kind string, lane string, attrs map[string]any) {
		if bearing.Wants(o, "state_delta") {
			o.Observe(bearing.Event{TaskID: "bidirectional_dijkstra", Component: "BOLTS", Kind: kind, Phase: "bidirectional_dijkstra", Lane: lane, LogicalStep: metrics.LogicalSteps, ScheduledStep: metrics.ScheduledSteps, WorkAfter: metrics.TotalActions, Attributes: attrs})
		}
	}
	if !consume("enqueue") {
		return core.RouteResult{Distance: math.Inf(1), SolverName: "bidirectional_dijkstra", Work: metrics, BudgetExhausted: true}
	}
	heap.Push(qf, &item{node: r.Source})
	emit("frontier_enqueued", "forward", map[string]any{"node": r.Source, "priority": 0.0, "distance": 0.0, "reason": "source", "lane": "forward"})
	if !consume("enqueue") {
		return core.RouteResult{Distance: math.Inf(1), SolverName: "bidirectional_dijkstra", Work: metrics, BudgetExhausted: true}
	}
	heap.Push(qb, &item{node: r.Target})
	emit("frontier_enqueued", "backward", map[string]any{"node": r.Target, "priority": 0.0, "distance": 0.0, "reason": "target", "lane": "backward"})
	seq := uint64(1)
	best := math.Inf(1)
	meet := core.NodeID(0)
	hasMeet := false
	var exp, rel, push, pop uint64
	push = 2
	collectDiagnostics := bearing.Wants(o, "state_delta") || bearing.Wants(o, "debug")
	var evaluatedEdges map[[3]uint64]struct{}
	if collectDiagnostics {
		evaluatedEdges = make(map[[3]uint64]struct{})
	}
	exhausted := false
	expandLane := func(forward bool) bool {
		q := qf
		dist, other, settled := df, db, sf
		edges := func(u core.NodeID) []core.Edge { return g.EdgesFrom(u) }
		prev, has := pf, hf
		if !forward {
			q, dist, other, settled, edges, prev, has = qb, db, df, sb, func(u core.NodeID) []core.Edge { return reverse[u] }, pb, hb
		}
		for q.Len() > 0 {
			if !consume("select") {
				exhausted = true
				return false
			}
			it := heap.Pop(q).(*item)
			pop++
			u := it.node
			laneName := map[bool]string{true: "forward", false: "backward"}[forward]
			emit("frontier_selected", laneName, map[string]any{"node": u, "priority": it.priority, "frontier_size": q.Len(), "lane": laneName})
			if it.distance != dist[u] {
				if !consume("reject") {
					exhausted = true
					return false
				}
				continue
			}
			if !consume("expand") {
				exhausted = true
				return false
			}
			settled[u] = true
			exp++
			emit("node_expanded", laneName, map[string]any{"node": u, "distance": dist[u], "frontier_size": q.Len(), "lane": laneName})
			if !math.IsInf(other[u], 1) && dist[u]+other[u] < best {
				if !consume("connect") {
					exhausted = true
					return false
				}
				best = dist[u] + other[u]
				meet = u
				hasMeet = true
			}
			for _, e := range edges(u) {
				emit("edge_evaluated", laneName, map[string]any{"from": u, "to": e.To, "weight": e.Weight, "lane": laneName})
				lane := uint64(0)
				if !forward {
					lane = 1
				}
				if collectDiagnostics {
					evaluatedEdges[[3]uint64{lane, uint64(u), uint64(e.To)}] = struct{}{}
				}
				if !consume("evaluate") || !consume("relax") {
					exhausted = true
					return false
				}
				rel++
				nd := dist[u] + e.Weight
				oldDistance := dist[e.To]
				if nd < dist[e.To] {
					dist[e.To] = nd
					emit("relaxation", laneName, map[string]any{"from": u, "to": e.To, "old_distance": oldDistance, "new_distance": nd, "accepted": true, "lane": laneName})
					prev[e.To] = u
					has[e.To] = true
					if !consume("enqueue") {
						exhausted = true
						return false
					}
					heap.Push(q, &item{node: e.To, priority: nd, distance: nd, sequence: seq})
					emit("frontier_enqueued", laneName, map[string]any{"node": e.To, "from": u, "priority": nd, "distance": nd, "lane": laneName})
					seq++
					push++
					if !math.IsInf(other[e.To], 1) && nd+other[e.To] < best {
						if !consume("connect") {
							exhausted = true
							return false
						}
						best = nd + other[e.To]
						meet = e.To
						hasMeet = true
					}
				} else {
					emit("relaxation", laneName, map[string]any{"from": u, "to": e.To, "old_distance": oldDistance, "new_distance": nd, "accepted": false, "lane": laneName})
					if !consume("reject") {
						exhausted = true
						return false
					}
				}
			}
			return true
		}
		return false
	}
	for qf.Len() > 0 && qb.Len() > 0 {
		select {
		case <-ctx.Done():
			return core.RouteResult{Distance: math.Inf(1), SolverName: "bidirectional_dijkstra", Work: metrics, ErrorCode: core.ErrCancelled}
		default:
		}
		if hasMeet && qf.Len() > 0 && qb.Len() > 0 && (*qf)[0].priority+(*qb)[0].priority >= best {
			break
		}
		forward := (*qf)[0].priority <= (*qb)[0].priority
		if !expandLane(forward) || exhausted {
			break
		}
	}
	if !consume("terminate") {
		exhausted = true
	}
	var path []core.NodeID
	if r.Source == r.Target {
		path = []core.NodeID{r.Source}
		best = 0
		hasMeet = true
	}
	if hasMeet && len(path) == 0 {
		left := reconstruct(pf, hf, r.Source, meet)
		if meet == r.Source {
			left = []core.NodeID{r.Source}
		}
		right := []core.NodeID{meet}
		cur := meet
		for cur != r.Target {
			if !hb[cur] {
				right = nil
				break
			}
			cur = pb[cur]
			right = append(right, cur)
		}
		if len(left) > 0 && len(right) > 0 {
			path = append(left, right[1:]...)
		}
	}
	found := len(path) > 0
	emit("search_finished", "", map[string]any{"found": found, "path": path, "distance": best, "meet": meet, "budget_exhausted": exhausted})
	exact := !exhausted
	unique := uint64(0)
	for i := 0; i < n; i++ {
		if sf[i] || sb[i] {
			unique++
		}
	}
	res := core.RouteResult{Path: path, Distance: best, Found: found, Exact: exact, SolverName: "bidirectional_dijkstra", Work: metrics, WorkRelaxations: rel, WorkExpandedNodes: exp, QueuePushes: push, QueuePops: pop, ParallelSteps: metrics.ScheduledSteps, TimeMS: float64(time.Since(start).Nanoseconds()) / 1_000_000, BudgetExhausted: exhausted, Telemetry: map[string]any{"investigated_nodes": unique, "investigated_node_ratio": float64(unique) / float64(maxInt(1, n)), "investigated_edges": len(evaluatedEdges), "investigated_edge_ratio": float64(len(evaluatedEdges)) / float64(maxInt(1, edgeSlots(g))), "investigated_node_ids": mergedNodeIDs(sf, sb), "investigated_edge_ids": encodedBiEdgeIDs(evaluatedEdges), "candidate_paths": func() uint64 {
		if found {
			return 1
		}
		return 0
	}(), "path_node_count": len(path)}}
	if exact && found {
		x := 1.0
		res.LowerBound = best
		res.CertifiedRatio = &x
		res.QualityCertified = true
	}
	res.SearchCompleted = !exhausted
	switch {
	case exhausted:
		res.TerminationStatus = core.TerminationUnknownBudget
		res.ErrorCode = core.ErrBudgetExhausted
	case found:
		res.TerminationStatus = core.TerminationFound
	case !found:
		res.TerminationStatus = core.TerminationUnreachable
		res.ErrorCode = core.ErrNoPath
		res.ReachabilityProven = exact
	}
	return finalizeTiming(res, "bolts")
}

func boolNodeIDs(v []bool) []uint64 {
	out := make([]uint64, 0)
	for i, ok := range v {
		if ok {
			out = append(out, uint64(i))
		}
	}
	return out
}
func mergedNodeIDs(a, b []bool) []uint64 {
	out := make([]uint64, 0)
	for i := range a {
		if a[i] || b[i] {
			out = append(out, uint64(i))
		}
	}
	return out
}
func encodedEdgeIDs(v map[[2]core.NodeID]struct{}) []uint64 {
	out := make([]uint64, 0, len(v))
	for e := range v {
		out = append(out, (uint64(e[0])<<32)|uint64(e[1]))
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func encodedBiEdgeIDs(v map[[3]uint64]struct{}) []uint64 {
	out := make([]uint64, 0, len(v))
	for e := range v {
		out = append(out, (e[0]<<63)^(e[1]<<31)^e[2])
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func finalizeTiming(result core.RouteResult, component string) core.RouteResult {
	if result.TimeBreakdown.TotalMS == 0 {
		result.TimeBreakdown.TotalMS = result.TimeMS
		result.TimeBreakdown.SolverMS = result.TimeMS
		result.TimeBreakdown.TotalNS = int64(result.TimeMS * 1_000_000)
		result.TimeBreakdown.SolverNS = result.TimeBreakdown.TotalNS
		if component == "anchor" {
			result.TimeBreakdown.AnchorMS = result.TimeMS
		}
		if component == "bolts" {
			result.TimeBreakdown.BoltsMS = result.TimeMS
		}
	}
	return result
}

func routePathDistance(g core.Graph, path []core.NodeID) float64 {
	total := 0.0
	for i := 1; i < len(path); i++ {
		found := false
		for _, e := range g.EdgesFrom(path[i-1]) {
			if e.To == path[i] {
				total += e.Weight
				found = true
				break
			}
		}
		if !found {
			return math.Inf(1)
		}
	}
	return total
}
