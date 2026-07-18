package core

import (
	"errors"
	"fmt"
	"time"
)

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
	Source                 NodeID          `json:"source"`
	Target                 NodeID          `json:"target"`
	Mode                   RouteMode       `json:"route_mode"`
	MaxSuboptimality       *float64        `json:"max_suboptimality,omitempty"`
	Deadline               time.Duration   `json:"-"`
	DeadlineMS             *float64        `json:"deadline_ms,omitempty"`
	WorkBudget             *uint64         `json:"work_budget,omitempty"`
	MemoryBudgetKiB        *float64        `json:"memory_budget_kib,omitempty"`
	Workers                int             `json:"logical_worker_count"`
	Seed                   uint64          `json:"seed"`
	AnchorStrategy         string          `json:"-"`
	Ablation               AblationOptions `json:"ablation,omitempty"`
	CollectProgressSamples bool            `json:"-"`
	HandoffWorkThreshold   *uint64         `json:"-"`
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
		return errors.New("logical_worker_count must be >= 1")
	}
	return nil
}

type RouteResult struct {
	TerminationStatus  TerminationStatus  `json:"termination_status"`
	Path               []NodeID           `json:"path"`
	Distance           float64            `json:"path_cost"`
	Found              bool               `json:"path_found"`
	SearchCompleted    bool               `json:"search_completed"`
	ReachabilityProven bool               `json:"reachability_proven"`
	Exact              bool               `json:"optimality_proven"`
	SolverName         string             `json:"solver_name"`
	Work               WorkMetrics        `json:"work"`
	WorkRelaxations    uint64             `json:"work_relaxations"`
	WorkExpandedNodes  uint64             `json:"work_expanded_nodes"`
	QueuePushes        uint64             `json:"queue_pushes"`
	QueuePops          uint64             `json:"queue_pops"`
	ParallelSteps      uint64             `json:"parallel_steps"`
	TimeMS             float64            `json:"end_to_end_time_ms"`
	TimeBreakdown      TimeBreakdown      `json:"time_breakdown"`
	LowerBound         float64            `json:"lower_bound"`
	CertifiedRatio     *float64           `json:"proven_cost_ratio,omitempty"`
	QualityCertified   bool               `json:"quality_bound_proven"`
	FirstPathWork      *uint64            `json:"first_path_work,omitempty"`
	FallbackUsed       bool               `json:"fallback_used"`
	BudgetExhausted    bool               `json:"budget_exhausted"`
	DeadlineExceeded   bool               `json:"deadline_exceeded"`
	ErrorCode          ErrorCode          `json:"error_code,omitempty"`
	SolverTrace        []TaskTrace        `json:"solver_trace,omitempty"`
	Telemetry          map[string]any     `json:"telemetry,omitempty"`
	FailureReason      string             `json:"failure_reason,omitempty"`
	TimeToFirstPathMS  *float64           `json:"first_path_elapsed_ms,omitempty"`
	TimeToBestFoundMS  *float64           `json:"best_path_elapsed_ms,omitempty"`
	ImprovementCount   uint64             `json:"improvement_count"`
	BudgetLedger       *BudgetLedger      `json:"budget_ledger,omitempty"`
	HandoffMetrics     *HandoffMetrics    `json:"handoff_metrics,omitempty"`
	BottleneckProfile  *BottleneckProfile `json:"bottleneck_profile,omitempty"`
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
