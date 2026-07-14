package anchor

import (
	"container/heap"
	"context"
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"math"
	"time"
)

type SessionProgress struct {
	WorkUsed     uint64
	Found        bool
	BestDistance float64
	LowerBound   float64
	Finished     bool
	Status       core.TerminationStatus
	Hypotheses   []core.Hypothesis
}
type StepResult struct {
	Consumed    uint64
	Candidate   *core.RouteResult
	UpperBound  *float64
	LowerBound  *float64
	Checkpoints []core.Checkpoint
	Evidence    []core.Evidence
	NextAction  string
	Finished    bool
}
type Snapshot struct {
	Request                core.RouteRequest
	Dist                   []float64
	Prev                   []core.NodeID
	HasPrev                []bool
	Settled                []bool
	Queue                  []qitem
	Seq                    uint64
	Metrics                core.WorkMetrics
	Best                   *core.RouteResult
	Finished               bool
	Cancelled              bool
	Status                 core.TerminationStatus
	Hypotheses             []core.Hypothesis
	ActiveNode             *core.NodeID
	ActiveEdge             int
	HypothesisID           string
	HeuristicScale         float64
	HeuristicUnitScale     float64
	HeuristicCache         []float64
	FirstPathWork          *uint64
	FirstPathElapsed       *float64
	CandidateUpdates       uint64
	LastCandidateWork      uint64
	MaxFrontier            int
	ProgressSamples        []core.ProgressSample
	NextProgressSampleWork uint64
}
type Session struct {
	g                      core.Graph
	request                core.RouteRequest
	dist                   []float64
	prev                   []core.NodeID
	hasPrev                []bool
	settled                []bool
	q                      queue
	seq                    uint64
	metrics                core.WorkMetrics
	best                   *core.RouteResult
	finished, cancelled    bool
	status                 core.TerminationStatus
	observer               bearing.Observer
	hypotheses             []core.Hypothesis
	activeNode             *core.NodeID
	activeEdge             int
	started                time.Time
	hypothesisID           string
	heuristicScale         float64
	heuristicCache         []float64
	heuristicUnitScale     float64
	firstPathWork          *uint64
	firstPathElapsed       *float64
	candidateUpdates       uint64
	lastCandidateWork      uint64
	maxFrontier            int
	progressSamples        []core.ProgressSample
	nextProgressSampleWork uint64
	bestHeuristic          float64
}

func NewSession(g core.Graph, r core.RouteRequest, o bearing.Observer) (*Session, error) {
	return NewHypothesisSession(g, r, o, "main", "adaptive_fast_path", RecommendedWeight(g, r.Mode))
}

// NewHypothesisSession creates an independent local-search state for one
// partial-graph hypothesis. Scales in (0,1] preserve admissibility while
// producing distinct expansion orders.
func NewHypothesisSession(g core.Graph, r core.RouteRequest, o bearing.Observer, id, kind string, heuristicScale float64) (*Session, error) {
	if err := r.Validate(g); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, fmt.Errorf("hypothesis id is required")
	}
	if heuristicScale <= 0 || heuristicScale > 8 {
		return nil, fmt.Errorf("heuristic scale must be in (0,8]")
	}
	n := g.NodeCount()
	s := &Session{g: g, request: r, dist: make([]float64, n), prev: make([]core.NodeID, n), hasPrev: make([]bool, n), settled: make([]bool, n), status: core.TerminationRunning, observer: o, started: time.Now(), hypothesisID: id, heuristicScale: heuristicScale, heuristicCache: make([]float64, n), heuristicUnitScale: graphHeuristicUnitScale(g), maxFrontier: 1, nextProgressSampleWork: 64, bestHeuristic: math.Inf(1)}
	for i := range s.heuristicCache {
		s.heuristicCache[i] = math.NaN()
	}
	for i := range s.dist {
		s.dist[i] = math.Inf(1)
	}
	s.dist[r.Source] = 0
	s.q = queue{}
	heap.Init(&s.q)
	heap.Push(&s.q, qitem{n: r.Source, pri: heuristicScale * s.heuristic(r.Source), seq: 1})
	s.seq = 1
	s.metrics.WorkerCount = uint32(r.Workers)
	s.metrics.AddAction(string(core.WorkEnqueue))
	if o != nil {
		o.Observe(bearing.Event{TaskID: "anchor-" + id, Component: "ANCHOR", Phase: "session", Kind: "action", Action: string(core.WorkEnqueue), LogicalStep: 1, ScheduledStep: 1, WorkBefore: 0, WorkAfter: 1})
		if bearing.Wants(o, "state_delta") {
			o.Observe(bearing.Event{TaskID: "anchor-" + id, Component: "ANCHOR", Phase: "session", Kind: "frontier_enqueued", LogicalStep: 1, ScheduledStep: 1, WorkAfter: 1, Attributes: map[string]any{"node": r.Source, "priority": heuristicScale * s.heuristic(r.Source), "distance": 0.0, "reason": "source"}})
		}
	}
	s.metrics.LogicalSteps = 1
	s.metrics.ScheduledSteps = 1
	s.hypotheses = []core.Hypothesis{{ID: id, Kind: kind, Region: core.Region{Nodes: []core.NodeID{r.Source}, Version: 1}, State: core.HypothesisRunnable, WorkUsed: 1}}
	return s, nil
}
func Resume(g core.Graph, snap Snapshot, o bearing.Observer) (*Session, error) {
	s := &Session{g: g, request: snap.Request, dist: append([]float64{}, snap.Dist...), prev: append([]core.NodeID{}, snap.Prev...), hasPrev: append([]bool{}, snap.HasPrev...), settled: append([]bool{}, snap.Settled...), q: append(queue{}, snap.Queue...), seq: snap.Seq, metrics: snap.Metrics, finished: snap.Finished, cancelled: snap.Cancelled, status: snap.Status, observer: o, hypotheses: append([]core.Hypothesis{}, snap.Hypotheses...), activeEdge: snap.ActiveEdge, started: time.Now(), hypothesisID: snap.HypothesisID, heuristicScale: snap.HeuristicScale, heuristicUnitScale: snap.HeuristicUnitScale, heuristicCache: append([]float64{}, snap.HeuristicCache...), firstPathWork: snap.FirstPathWork, firstPathElapsed: snap.FirstPathElapsed, candidateUpdates: snap.CandidateUpdates, lastCandidateWork: snap.LastCandidateWork, maxFrontier: snap.MaxFrontier, progressSamples: append([]core.ProgressSample(nil), snap.ProgressSamples...), nextProgressSampleWork: snap.NextProgressSampleWork, bestHeuristic: math.Inf(1)}
	if snap.Best != nil {
		b := *snap.Best
		b.Path = append([]core.NodeID{}, snap.Best.Path...)
		s.best = &b
	}
	if snap.ActiveNode != nil {
		v := *snap.ActiveNode
		s.activeNode = &v
	}
	heap.Init(&s.q)
	return s, nil
}
func (s *Session) Snapshot() Snapshot {
	q := append([]qitem{}, s.q...)
	var b *core.RouteResult
	if s.best != nil {
		x := *s.best
		x.Path = append([]core.NodeID{}, s.best.Path...)
		b = &x
	}
	snap := Snapshot{Request: s.request, Dist: append([]float64{}, s.dist...), Prev: append([]core.NodeID{}, s.prev...), HasPrev: append([]bool{}, s.hasPrev...), Settled: append([]bool{}, s.settled...), Queue: q, Seq: s.seq, Metrics: s.metrics, Best: b, Finished: s.finished, Cancelled: s.cancelled, Status: s.status, Hypotheses: append([]core.Hypothesis{}, s.hypotheses...), ActiveEdge: s.activeEdge, HypothesisID: s.hypothesisID, HeuristicScale: s.heuristicScale, HeuristicUnitScale: s.heuristicUnitScale, HeuristicCache: append([]float64{}, s.heuristicCache...), FirstPathWork: s.firstPathWork, FirstPathElapsed: s.firstPathElapsed, CandidateUpdates: s.candidateUpdates, LastCandidateWork: s.lastCandidateWork, MaxFrontier: s.maxFrontier}
	if s.activeNode != nil {
		v := *s.activeNode
		snap.ActiveNode = &v
	}
	return snap
}

func (s *Session) HypothesisID() string { return s.hypothesisID }
func (s *Session) Freeze() {
	if !s.finished {
		s.hypotheses[0].State = core.HypothesisFrozen
	}
}
func (s *Session) ResumeHypothesis() {
	if !s.finished {
		s.hypotheses[0].State = core.HypothesisRunnable
	}
}
func (s *Session) Prune() {
	s.finished = true
	s.status = core.TerminationCancelled
	s.hypotheses[0].State = core.HypothesisPruned
}
func (s *Session) Cancel() {
	s.cancelled = true
	s.finished = true
	s.status = core.TerminationCancelled
}
func (s *Session) Finished() bool { return s.finished }
func (s *Session) Progress() SessionProgress {
	lb := math.Inf(1)
	if len(s.q) > 0 {
		lb = s.q[0].pri
	}
	bd := math.Inf(1)
	if s.best != nil {
		bd = s.best.Distance
	}
	return SessionProgress{WorkUsed: s.metrics.TotalActions, Found: s.best != nil, BestDistance: bd, LowerBound: lb, Finished: s.finished, Status: s.status, Hypotheses: append([]core.Hypothesis{}, s.hypotheses...)}
}
func (s *Session) Result() core.RouteResult {
	if s.best != nil {
		r := *s.best
		r.Work = s.metrics
		r.FirstPathWork = s.firstPathWork
		r.TimeToFirstPathMS = s.firstPathElapsed
		r.ImprovementCount = s.candidateUpdates
		if r.Telemetry == nil {
			r.Telemetry = map[string]any{}
		}
		r.Telemetry["max_frontier_size"] = s.maxFrontier
		r.Telemetry["works_since_candidate_update"] = s.metrics.TotalActions - s.lastCandidateWork
		r.Telemetry["heuristic_weight"] = s.heuristicScale
		r.TerminationStatus = s.status
		r.SearchCompleted = s.finished
		r.BudgetExhausted = s.status == core.TerminationUnknownBudget
		return r
	}
	return core.RouteResult{Distance: math.Inf(1), SolverName: "anchor", Work: s.metrics, SearchCompleted: s.finished, TerminationStatus: s.status, ReachabilityProven: s.status == core.TerminationUnreachable, ErrorCode: mapStatusError(s.status)}
}
func mapStatusError(st core.TerminationStatus) core.ErrorCode {
	switch st {
	case core.TerminationUnknownBudget:
		return core.ErrBudgetExhausted
	case core.TerminationCancelled:
		return core.ErrCancelled
	case core.TerminationUnreachable:
		return core.ErrNoPath
	}
	return ""
}
func (s *Session) Step(ctx context.Context, grant uint64) StepResult {
	out := StepResult{}
	if s.finished || grant == 0 {
		return out
	}
	start := s.metrics.TotalActions
	remaining := func() uint64 { return grant - (s.metrics.TotalActions - start) }
	emit := func(kind string, attrs map[string]any) {
		if s.observer != nil && bearing.Wants(s.observer, "state_delta") {
			s.observer.Observe(bearing.Event{TaskID: "anchor-" + s.hypothesisID, Component: "ANCHOR", Phase: "session", Kind: kind, LogicalStep: s.metrics.LogicalSteps, ScheduledStep: s.metrics.ScheduledSteps, WorkAfter: s.metrics.TotalActions, Attributes: attrs})
		}
	}
	consume := func(k core.WorkAction) {
		s.metrics.AddAction(string(k))
		if s.request.CollectProgressSamples && s.metrics.TotalActions >= s.nextProgressSampleWork {
			lb := math.Inf(1)
			if len(s.q) > 0 {
				lb = s.q[0].pri
			}
			den := s.metrics.EvaluateActions
			rejectRate := 0.0
			if den > 0 {
				rejectRate = float64(s.metrics.RejectActions) / float64(den)
			}
			bestHeuristic := s.bestHeuristic
			if math.IsInf(bestHeuristic, 0) || math.IsNaN(bestHeuristic) {
				bestHeuristic = 0
			}
			if math.IsInf(lb, 0) || math.IsNaN(lb) {
				lb = 0
			}
			s.progressSamples = append(s.progressSamples, core.ProgressSample{Work: s.metrics.TotalActions, CandidateFound: s.best != nil, WorksSinceCandidateUpdate: s.metrics.TotalActions - s.lastCandidateWork, FrontierSize: uint64(len(s.q)), RejectRate: rejectRate, BestHeuristic: bestHeuristic, LowerBound: lb})
			for s.nextProgressSampleWork <= s.metrics.TotalActions {
				s.nextProgressSampleWork += 64
			}
		}
		if s.observer != nil && bearing.Wants(s.observer, "action") {
			s.observer.Observe(bearing.Event{TaskID: "anchor-" + s.hypothesisID, Component: "ANCHOR", Phase: "session", Kind: "action", Action: string(k), LogicalStep: s.metrics.LogicalSteps + 1, ScheduledStep: s.metrics.ScheduledSteps + 1, WorkBefore: s.metrics.TotalActions - 1, WorkAfter: s.metrics.TotalActions})
		}
		s.metrics.LogicalSteps++
		s.metrics.ScheduledSteps++
	}
	for remaining() > 0 && !s.finished {
		if ctx.Err() != nil {
			s.finished = true
			s.status = core.TerminationCancelled
			break
		}
		if s.activeNode == nil {
			if len(s.q) == 0 {
				if remaining() < 1 {
					break
				}
				consume(core.WorkTerminate)
				s.finished = true
				if s.best != nil {
					s.status = core.TerminationFound
					s.best.Exact = true
					s.best.QualityCertified = true
					s.best.LowerBound = s.best.Distance
					x := 1.0
					s.best.CertifiedRatio = &x
				} else {
					s.status = core.TerminationUnreachable
				}
				break
			}
			if remaining() < 2 {
				break
			}
			consume(core.WorkSelect)
			it := heap.Pop(&s.q).(qitem)
			emit("frontier_selected", map[string]any{"node": it.n, "priority": it.pri, "frontier_size": len(s.q)})
			if s.settled[it.n] {
				continue
			}
			consume(core.WorkExpand)
			s.settled[it.n] = true
			emit("node_expanded", map[string]any{"node": it.n, "distance": s.dist[it.n], "frontier_size": len(s.q)})
			s.hypotheses[0].Region.Nodes = append(s.hypotheses[0].Region.Nodes, it.n)
			s.hypotheses[0].Region.Version++
			v := it.n
			s.activeNode = &v
			s.activeEdge = 0
		}
		u := *s.activeNode
		if u == s.request.Target {
			if remaining() < 1 {
				break
			}
			consume(core.WorkCandidate)
			path := reconstructSession(s.request.Source, s.request.Target, s.prev, s.hasPrev)
			r := core.RouteResult{Path: path, Distance: s.dist[u], Found: true, SolverName: "anchor", Work: s.metrics, TerminationStatus: core.TerminationRunning}
			if s.best == nil || r.Distance < s.best.Distance {
				s.candidateUpdates++
				s.lastCandidateWork = s.metrics.TotalActions
				if s.firstPathWork == nil {
					w := s.metrics.TotalActions
					s.firstPathWork = &w
					ms := float64(time.Since(s.started).Nanoseconds()) / 1e6
					s.firstPathElapsed = &ms
				}
				s.best = &r
				out.Candidate = &r
				emit("candidate_submitted", map[string]any{"distance": r.Distance, "path": r.Path, "frontier_size": len(s.q)})
				x := r.Distance
				out.UpperBound = &x
			}
			s.activeNode = nil
			s.activeEdge = 0
			continue
		}
		edges := s.g.EdgesFrom(u)
		if s.activeEdge >= len(edges) {
			s.activeNode = nil
			s.activeEdge = 0
			continue
		}
		e := edges[s.activeEdge]
		nd := s.dist[u] + e.Weight
		// Every edge transition is atomic to preserve snapshot/resume equivalence.
		if remaining() < 3 {
			break
		}
		consume(core.WorkEvaluate)
		emit("edge_evaluated", map[string]any{"from": u, "to": e.To, "weight": e.Weight})
		consume(core.WorkRelax)
		oldDistance := s.dist[e.To]
		if nd < s.dist[e.To] {
			s.dist[e.To] = nd
			s.prev[e.To] = u
			s.hasPrev[e.To] = true
			s.seq++
			emit("relaxation", map[string]any{"from": u, "to": e.To, "old_distance": oldDistance, "new_distance": nd, "accepted": true})
			consume(core.WorkEnqueue)
			h := s.heuristic(e.To)
			if h < s.bestHeuristic {
				s.bestHeuristic = h
			}
			priority := nd + s.heuristicScale*h
			heap.Push(&s.q, qitem{n: e.To, pri: priority, seq: s.seq})
			emit("frontier_enqueued", map[string]any{"node": e.To, "from": u, "priority": priority, "distance": nd})
			if len(s.q) > s.maxFrontier {
				s.maxFrontier = len(s.q)
			}
		} else {
			emit("relaxation", map[string]any{"from": u, "to": e.To, "old_distance": oldDistance, "new_distance": nd, "accepted": false})
			consume(core.WorkReject)
		}
		s.activeEdge++
	}
	out.Consumed = s.metrics.TotalActions - start
	out.Finished = s.finished
	if len(s.q) > 0 {
		v := s.q[0].pri
		out.LowerBound = &v
	}
	if s.finished {
		out.NextAction = "terminate"
	} else if s.best != nil {
		out.NextAction = "improve_or_certify"
	} else {
		out.NextAction = "continue"
	}
	s.hypotheses[0].WorkUsed = s.metrics.TotalActions
	if s.finished {
		s.hypotheses[0].State = core.HypothesisFinished
	}
	return out
}
func reconstructSession(src, t core.NodeID, prev []core.NodeID, has []bool) []core.NodeID {
	p := []core.NodeID{t}
	for p[len(p)-1] != src {
		v := p[len(p)-1]
		if !has[v] {
			return nil
		}
		p = append(p, prev[v])
	}
	for i, j := 0, len(p)-1; i < j; i, j = i+1, j-1 {
		p[i], p[j] = p[j], p[i]
	}
	return p
}

// HandoffSeed exports algorithm-neutral state for BOLTS continuation.
type HandoffSeed struct {
	Dist     []float64
	Prev     []core.NodeID
	HasPrev  []bool
	Settled  []bool
	Frontier []core.NodeID
}

func (s *Session) ExportHandoffSeed() HandoffSeed {
	frontier := make([]core.NodeID, 0, len(s.q))
	seen := make(map[core.NodeID]struct{}, len(s.q))
	for _, it := range s.q {
		if _, ok := seen[it.n]; ok {
			continue
		}
		seen[it.n] = struct{}{}
		frontier = append(frontier, it.n)
	}
	return HandoffSeed{
		Dist:     append([]float64(nil), s.dist...),
		Prev:     append([]core.NodeID(nil), s.prev...),
		HasPrev:  append([]bool(nil), s.hasPrev...),
		Settled:  append([]bool(nil), s.settled...),
		Frontier: frontier,
	}
}

func (s *Session) SetHeuristicScale(scale float64) error {
	if scale <= 0 || scale > 8 {
		return fmt.Errorf("heuristic scale must be in (0,8]")
	}
	s.heuristicScale = scale
	for i := range s.q {
		s.q[i].pri = s.dist[s.q[i].n] + scale*s.heuristic(s.q[i].n)
	}
	heap.Init(&s.q)
	return nil
}

func (s *Session) HeuristicScale() float64 { return s.heuristicScale }
func (s *Session) MaxFrontier() int        { return s.maxFrontier }
func (s *Session) WorksSinceCandidateUpdate() uint64 {
	return s.metrics.TotalActions - s.lastCandidateWork
}
func (s *Session) CandidateUpdates() uint64 { return s.candidateUpdates }
func (s *Session) ProgressSamples() []core.ProgressSample {
	return append([]core.ProgressSample(nil), s.progressSamples...)
}
func (s *Session) Checkpoints(limit int) []core.Checkpoint {
	out := make([]core.Checkpoint, 0, limit)
	for i, d := range s.dist {
		if !math.IsInf(d, 1) {
			out = append(out, core.Checkpoint{Node: core.NodeID(i), Cost: d, HypothesisID: s.hypothesisID})
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}
func (s *Session) heuristic(n core.NodeID) float64 {
	if int(n) < len(s.heuristicCache) && !math.IsNaN(s.heuristicCache[n]) {
		return s.heuristicCache[n]
	}
	v := 0.0
	pa, oka := s.g.Position(n)
	pb, okb := s.g.Position(s.request.Target)
	if oka && okb {
		v = s.heuristicUnitScale * core.Euclidean(pa, pb)
	}
	if int(n) < len(s.heuristicCache) {
		s.heuristicCache[n] = v
	}
	return v
}

func graphHeuristicUnitScale(g core.Graph) float64 {
	min := math.Inf(1)
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
			if d > 1e-12 && e.Weight/d < min {
				min = e.Weight / d
			}
		}
	}
	if math.IsInf(min, 1) {
		return 0
	}
	return min
}

func sessionHeuristic(g core.Graph, a, b core.NodeID) float64 {
	pa, oka := g.Position(a)
	pb, okb := g.Position(b)
	if !oka || !okb {
		return 0
	}
	return graphHeuristicUnitScale(g) * core.Euclidean(pa, pb)
}

// ApplyHandoff safely imports BOLTS output into this session. Only concrete
// paths, checkpoints and validated evidence are reused; empirical evidence is
// never promoted to a proof.
func (s *Session) ApplyHandoff(result core.HandoffResult) error {
	for _, ev := range result.Evidence {
		if err := ev.Validate(); err != nil {
			return err
		}
	}
	for _, cp := range result.ResumeCheckpoints {
		if !s.g.HasNode(cp.Node) {
			return fmt.Errorf("handoff checkpoint %d outside graph", cp.Node)
		}
		if cp.Cost < s.dist[cp.Node] {
			s.dist[cp.Node] = cp.Cost
			s.seq++
			heap.Push(&s.q, qitem{n: cp.Node, pri: cp.Cost + s.heuristicScale*s.heuristic(cp.Node), seq: s.seq})
			if len(s.q) > s.maxFrontier {
				s.maxFrontier = len(s.q)
			}
		}
		s.hypotheses[0].Checkpoints = append(s.hypotheses[0].Checkpoints, cp)
	}
	if result.Found && len(result.Path) > 0 {
		d := core.PathDistance(s.g, result.Path)
		if math.IsInf(d, 1) || math.Abs(d-result.Distance) > 1e-9 {
			return fmt.Errorf("invalid handoff path")
		}
		candidate := core.RouteResult{Path: append([]core.NodeID(nil), result.Path...), Distance: d, Found: true, SolverName: "anchor+bolts", TerminationStatus: core.TerminationRunning}
		if s.best == nil || candidate.Distance < s.best.Distance {
			s.best = &candidate
		}
	}
	return nil
}

// HandoffStateStats reports state that can potentially be transferred to a rescue solver.
func (s *Session) HandoffStateStats() (available, transferable uint64) {
	for _, ok := range s.settled {
		if ok {
			available++
		}
	}
	available += uint64(len(s.q))
	for i, d := range s.dist {
		if i < len(s.settled) && !s.settled[i] && !math.IsInf(d, 1) {
			available++
		}
	}
	transferable = uint64(len(s.q))
	if s.best != nil {
		transferable += uint64(len(s.best.Path))
	}
	return
}

// LiveSignals returns O(1) aggregate progress indicators without enabling debug samples.
func (s *Session) LiveSignals() (frontier uint64, rejectRate float64, bestHeuristic float64, lowerBound float64) {
	frontier = uint64(len(s.q))
	if s.metrics.TotalActions > 0 {
		rejectRate = float64(s.metrics.RejectActions) / float64(s.metrics.TotalActions)
	}
	bestHeuristic = s.bestHeuristic
	if math.IsInf(bestHeuristic, 1) {
		bestHeuristic = 0
	}
	lowerBound = s.Progress().LowerBound
	if math.IsInf(lowerBound, 0) || math.IsNaN(lowerBound) {
		lowerBound = 0
	}
	return
}
