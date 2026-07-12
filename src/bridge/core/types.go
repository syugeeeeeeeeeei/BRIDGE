package core

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"
)

type NodeID uint32

type Point struct{ X, Y float64 }
type Edge struct {
	To     NodeID
	Weight float64
}

type Graph interface {
	NodeCount() int
	EdgeCount() int
	EdgesFrom(NodeID) []Edge
	HasNode(NodeID) bool
	Position(NodeID) (Point, bool)
	Directed() bool
}

type AdjacencyGraph struct {
	Adj        [][]Edge
	Pos        []Point
	HasPos     []bool
	IsDirected bool
	edges      int
}

func NewAdjacencyGraph(nodes int, directed bool) *AdjacencyGraph {
	return &AdjacencyGraph{Adj: make([][]Edge, nodes), Pos: make([]Point, nodes), HasPos: make([]bool, nodes), IsDirected: directed}
}
func (g *AdjacencyGraph) AddEdge(from, to NodeID, weight float64) error {
	if weight < 0 || math.IsNaN(weight) || math.IsInf(weight, 0) {
		return errors.New("edge weight must be finite and non-negative")
	}
	if !g.HasNode(from) || !g.HasNode(to) {
		return fmt.Errorf("node out of range: %d -> %d", from, to)
	}
	g.Adj[from] = insertEdgeCanonical(g.Adj[from], Edge{To: to, Weight: weight})
	g.edges++
	if !g.IsDirected {
		g.Adj[to] = insertEdgeCanonical(g.Adj[to], Edge{To: from, Weight: weight})
	}
	return nil
}

// insertEdgeCanonical keeps every adjacency list ordered by destination and then
// weight. Search algorithms therefore observe the same edge order regardless of
// the order in which callers constructed the graph.
func insertEdgeCanonical(edges []Edge, edge Edge) []Edge {
	i := sort.Search(len(edges), func(i int) bool {
		if edges[i].To != edge.To {
			return edges[i].To > edge.To
		}
		return edges[i].Weight >= edge.Weight
	})
	edges = append(edges, Edge{})
	copy(edges[i+1:], edges[i:])
	edges[i] = edge
	return edges
}

func (g *AdjacencyGraph) SetPosition(n NodeID, p Point) error {
	if !g.HasNode(n) {
		return fmt.Errorf("node out of range: %d", n)
	}
	g.Pos[n] = p
	g.HasPos[n] = true
	return nil
}
func (g *AdjacencyGraph) NodeCount() int { return len(g.Adj) }
func (g *AdjacencyGraph) EdgeCount() int { return g.edges }
func (g *AdjacencyGraph) EdgesFrom(n NodeID) []Edge {
	if !g.HasNode(n) {
		return nil
	}
	return g.Adj[n]
}
func (g *AdjacencyGraph) HasNode(n NodeID) bool { return uint64(n) < uint64(len(g.Adj)) }
func (g *AdjacencyGraph) Position(n NodeID) (Point, bool) {
	if !g.HasNode(n) || !g.HasPos[n] {
		return Point{}, false
	}
	return g.Pos[n], true
}
func (g *AdjacencyGraph) Directed() bool { return g.IsDirected }

type RouteMode string

const (
	ModeFast     RouteMode = "fast"
	ModeBalanced RouteMode = "balanced"
	ModeQuality  RouteMode = "quality"
	ModeExact    RouteMode = "exact"
)

type AblationOptions struct {
	DisableFallback           bool `json:"disable_fallback,omitempty"`
	DisableCertification      bool `json:"disable_certification,omitempty"`
	DisableDetour             bool `json:"disable_detour,omitempty"`
	DisableBudgetReallocation bool `json:"disable_budget_reallocation,omitempty"`
	DisableStateReuse         bool `json:"disable_state_reuse,omitempty"`
}

type RouteRequest struct {
	Source           NodeID          `json:"source"`
	Target           NodeID          `json:"target"`
	Mode             RouteMode       `json:"mode"`
	MaxSuboptimality *float64        `json:"max_suboptimality,omitempty"`
	Deadline         time.Duration   `json:"-"`
	DeadlineMS       *float64        `json:"deadline_ms,omitempty"`
	WorkBudget       *uint64         `json:"work_budget,omitempty"`
	MemoryBudgetKiB  *float64        `json:"memory_budget_kib,omitempty"`
	Workers          int             `json:"workers"`
	Seed             uint64          `json:"seed"`
	AnchorStrategy   string          `json:"-"`
	Ablation         AblationOptions `json:"ablation,omitempty"`
}

func (r RouteRequest) Validate(g Graph) error {
	if r.Mode == "" {
		r.Mode = ModeBalanced
	}
	switch r.Mode {
	case ModeFast, ModeBalanced, ModeQuality, ModeExact:
	default:
		return fmt.Errorf("unsupported BRIDGE mode: %s", r.Mode)
	}
	if !g.HasNode(r.Source) || !g.HasNode(r.Target) {
		return errors.New("source or target node does not exist")
	}
	if r.MaxSuboptimality != nil && *r.MaxSuboptimality < 1 {
		return errors.New("max_suboptimality must be >= 1.0")
	}
	if r.DeadlineMS != nil && *r.DeadlineMS <= 0 {
		return errors.New("deadline_ms must be positive")
	}
	if r.Workers < 1 {
		return errors.New("workers must be >= 1")
	}
	return nil
}

type ErrorCode string

const (
	ErrInvalidRequest   ErrorCode = "INVALID_REQUEST"
	ErrNoPath           ErrorCode = "NO_PATH"
	ErrBudgetExhausted  ErrorCode = "BUDGET_EXHAUSTED"
	ErrDeadlineExceeded ErrorCode = "DEADLINE_EXCEEDED"
	ErrCancelled        ErrorCode = "CANCELLED"
)

type WorkAction string

const (
	WorkSelect    WorkAction = "select"
	WorkExpand    WorkAction = "expand"
	WorkEvaluate  WorkAction = "evaluate"
	WorkRelax     WorkAction = "relax"
	WorkEnqueue   WorkAction = "enqueue"
	WorkReject    WorkAction = "reject"
	WorkBacktrack WorkAction = "backtrack"
	WorkConnect   WorkAction = "connect"
	WorkCandidate WorkAction = "candidate"
	WorkRepair    WorkAction = "repair"
	WorkBound     WorkAction = "bound"
	WorkTerminate WorkAction = "terminate"
)

func IsWorkAction(kind string) bool {
	switch WorkAction(kind) {
	case WorkSelect, WorkExpand, WorkEvaluate, WorkRelax, WorkEnqueue, WorkReject, WorkBacktrack, WorkConnect, WorkCandidate, WorkRepair, WorkBound, WorkTerminate:
		return true
	default:
		return false
	}
}

type WorkMetrics struct {
	TotalActions     uint64 `json:"total_actions"`
	SelectActions    uint64 `json:"select_actions"`
	ExpandActions    uint64 `json:"expand_actions"`
	EvaluateActions  uint64 `json:"evaluate_actions"`
	RelaxActions     uint64 `json:"relax_actions"`
	EnqueueActions   uint64 `json:"enqueue_actions"`
	RejectActions    uint64 `json:"reject_actions"`
	BacktrackActions uint64 `json:"backtrack_actions"`
	ConnectActions   uint64 `json:"connect_actions"`
	CandidateActions uint64 `json:"candidate_actions"`
	RepairActions    uint64 `json:"repair_actions"`
	BoundActions     uint64 `json:"bound_actions"`
	TerminateActions uint64 `json:"terminate_actions"`
	LogicalSteps     uint64 `json:"logical_steps"`
	ScheduledSteps   uint64 `json:"scheduled_steps"`
	WorkerCount      uint32 `json:"worker_count"`
}

func (w *WorkMetrics) AddAction(kind string) {
	if !IsWorkAction(kind) {
		panic("unknown Work action: " + kind)
	}
	w.TotalActions++
	switch kind {
	case "select":
		w.SelectActions++
	case "expand":
		w.ExpandActions++
	case "evaluate":
		w.EvaluateActions++
	case "relax":
		w.RelaxActions++
	case "enqueue":
		w.EnqueueActions++
	case "reject":
		w.RejectActions++
	case "backtrack":
		w.BacktrackActions++
	case "connect":
		w.ConnectActions++
	case "candidate":
		w.CandidateActions++
	case "repair":
		w.RepairActions++
	case "bound":
		w.BoundActions++
	case "terminate":
		w.TerminateActions++
	}
}

func (w *WorkMetrics) Add(other WorkMetrics) {
	w.TotalActions += other.TotalActions
	w.SelectActions += other.SelectActions
	w.ExpandActions += other.ExpandActions
	w.EvaluateActions += other.EvaluateActions
	w.RelaxActions += other.RelaxActions
	w.EnqueueActions += other.EnqueueActions
	w.RejectActions += other.RejectActions
	w.BacktrackActions += other.BacktrackActions
	w.ConnectActions += other.ConnectActions
	w.CandidateActions += other.CandidateActions
	w.RepairActions += other.RepairActions
	w.BoundActions += other.BoundActions
	w.TerminateActions += other.TerminateActions
	w.LogicalSteps += other.LogicalSteps
	w.ScheduledSteps += other.ScheduledSteps
	if other.WorkerCount > w.WorkerCount {
		w.WorkerCount = other.WorkerCount
	}
}

func (w WorkMetrics) CountedActions() uint64 {
	return w.SelectActions + w.ExpandActions + w.EvaluateActions + w.RelaxActions + w.EnqueueActions + w.RejectActions + w.BacktrackActions + w.ConnectActions + w.CandidateActions + w.RepairActions + w.BoundActions + w.TerminateActions
}

func (w WorkMetrics) ValidationErrors() []string {
	errs := []string{}
	if w.TotalActions != w.CountedActions() {
		errs = append(errs, fmt.Sprintf("total_actions=%d differs from action sum=%d", w.TotalActions, w.CountedActions()))
	}
	if w.LogicalSteps > w.ScheduledSteps {
		errs = append(errs, fmt.Sprintf("logical_steps=%d exceeds scheduled_steps=%d", w.LogicalSteps, w.ScheduledSteps))
	}
	if w.ScheduledSteps > w.TotalActions {
		errs = append(errs, fmt.Sprintf("scheduled_steps=%d exceeds total_actions=%d", w.ScheduledSteps, w.TotalActions))
	}
	return errs
}

func (w WorkMetrics) Valid() bool {
	return w.TotalActions == w.SelectActions+w.ExpandActions+w.EvaluateActions+w.RelaxActions+w.EnqueueActions+w.RejectActions+w.BacktrackActions+w.ConnectActions+w.CandidateActions+w.RepairActions+w.BoundActions+w.TerminateActions && w.LogicalSteps <= w.ScheduledSteps && w.ScheduledSteps <= w.TotalActions
}

// TimeBreakdown records measured execution phases. These durations are not Work.
type TimeBreakdown struct {
	TotalMS         float64 `json:"total_ms"`
	SolverMS        float64 `json:"solver_ms"`
	TrussMS         float64 `json:"truss_ms,omitempty"`
	AnchorMS        float64 `json:"anchor_ms,omitempty"`
	BoltsMS         float64 `json:"bolts_ms,omitempty"`
	FallbackMS      float64 `json:"fallback_ms,omitempty"`
	SupervisorMS    float64 `json:"supervisor_ms,omitempty"`
	ArbiterMS       float64 `json:"arbiter_ms,omitempty"`
	OrchestrationMS float64 `json:"orchestration_ms,omitempty"`
	GateMS          float64 `json:"gate_ms,omitempty"`
}

// SystemMetrics records benchmark-process deltas. They are observational overhead, not search Work.
type SystemMetrics struct {
	AllocBytes           uint64 `json:"alloc_bytes"`
	MallocCount          uint64 `json:"malloc_count"`
	GCCount              uint32 `json:"gc_count"`
	HeapAllocBefore      uint64 `json:"heap_alloc_before"`
	HeapAllocAfter       uint64 `json:"heap_alloc_after"`
	HeapAllocBoundaryMax uint64 `json:"heap_alloc_boundary_max"`
	HeapAllocSampledPeak uint64 `json:"heap_alloc_sampled_peak,omitempty"`
}

// BudgetLedgerEntry records one solver task allocation and consumption.
// It is accounting data, not search Work itself.
type BudgetLedgerEntry struct {
	TaskID    string    `json:"task_id"`
	Component Component `json:"component"`
	Purpose   string    `json:"purpose"`
	Granted   *uint64   `json:"granted,omitempty"`
	Used      uint64    `json:"used"`
}

// BudgetLedger is the public accounting snapshot for a TRUSS execution.
type BudgetLedger struct {
	Limit       *uint64              `json:"limit,omitempty"`
	Used        uint64               `json:"used"`
	Remaining   *uint64              `json:"remaining,omitempty"`
	ByComponent map[Component]uint64 `json:"by_component"`
	Entries     []BudgetLedgerEntry  `json:"entries"`
}

type RouteResult struct {
	Path              []NodeID       `json:"path"`
	Distance          float64        `json:"distance"`
	Found             bool           `json:"found"`
	Exact             bool           `json:"exact"`
	SolverName        string         `json:"solver_name"`
	Work              WorkMetrics    `json:"work"`
	WorkRelaxations   uint64         `json:"work_relaxations"`
	WorkExpandedNodes uint64         `json:"work_expanded_nodes"`
	QueuePushes       uint64         `json:"queue_pushes"`
	QueuePops         uint64         `json:"queue_pops"`
	ParallelSteps     uint64         `json:"parallel_steps"`
	TimeMS            float64        `json:"time_ms"`
	TimeBreakdown     TimeBreakdown  `json:"time_breakdown"`
	LowerBound        float64        `json:"lower_bound"`
	CertifiedRatio    *float64       `json:"certified_ratio,omitempty"`
	QualityCertified  bool           `json:"quality_certified"`
	FirstPathWork     *uint64        `json:"first_path_work,omitempty"`
	FallbackUsed      bool           `json:"fallback_used"`
	BudgetExhausted   bool           `json:"budget_exhausted"`
	DeadlineExceeded  bool           `json:"deadline_exceeded"`
	ErrorCode         ErrorCode      `json:"error_code,omitempty"`
	SolverTrace       []TaskTrace    `json:"solver_trace,omitempty"`
	Telemetry         map[string]any `json:"telemetry,omitempty"`
	FailureReason     string         `json:"failure_reason,omitempty"`
	TimeToFirstPathMS *float64       `json:"time_to_first_path_ms,omitempty"`
	TimeToBestFoundMS *float64       `json:"time_to_best_found_ms,omitempty"`
	ImprovementCount  uint64         `json:"improvement_count"`
	BudgetLedger      *BudgetLedger  `json:"budget_ledger,omitempty"`
}

func (r RouteResult) TotalWork() uint64 { return r.Work.TotalActions }

// CompatibilityCountersValid verifies that deprecated diagnostic mirrors are
// derived from WorkMetrics and are not an independent Work source.
func (r RouteResult) CompatibilityCountersValid() bool {
	return r.WorkRelaxations == r.Work.RelaxActions &&
		r.WorkExpandedNodes == r.Work.ExpandActions &&
		r.QueuePushes == r.Work.EnqueueActions &&
		r.ParallelSteps == r.Work.ScheduledSteps
}

type WorkBudget struct {
	MaxWork   *uint64
	MaxExpand *uint64
}
type SolverProgress struct {
	TaskID          string
	WorkUsed        uint64
	ElapsedMS       float64
	Found           bool
	BestDistance    *float64
	LowerBound      *float64
	CandidateCount  int
	StagnationScore float64
	Finished        bool
	FailureReason   string
}
type SolverTask struct {
	ID, SolverKind, Purpose string
	Budget                  WorkBudget
	Workers                 int
	QualityTarget           float64
	Parameters              map[string]string
}
type TaskTrace struct {
	TaskID, Solver, Purpose, Reason string
	Allocation                      float64
	Budget                          *uint64
	Found                           bool
	Distance                        float64
	WorkUsed                        uint64
}

func Euclidean(a, b Point) float64 { return math.Hypot(a.X-b.X, a.Y-b.Y) }
func PathDistance(g Graph, path []NodeID) float64 {
	if len(path) == 0 {
		return math.Inf(1)
	}
	total := 0.0
	for i := 0; i+1 < len(path); i++ {
		ok := false
		for _, e := range g.EdgesFrom(path[i]) {
			if e.To == path[i+1] {
				total += e.Weight
				ok = true
				break
			}
		}
		if !ok {
			return math.Inf(1)
		}
	}
	return total
}
