package traffic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

// RouteRunner is the public boundary used by TRAFFIC. It is intentionally
// compatible with GATE without importing GATE's concrete implementation.
type RouteRunner interface {
	Route(context.Context, core.Graph, core.RouteRequest) (core.RouteResult, error)
}

type StableTaskTrace struct {
	TaskID     string  `json:"task_id"`
	Solver     string  `json:"solver"`
	Purpose    string  `json:"purpose"`
	Reason     string  `json:"reason"`
	Allocation string  `json:"allocation"`
	Budget     *uint64 `json:"budget,omitempty"`
	Found      bool    `json:"path_found"`
	Distance   string  `json:"path_cost"`
	WorkUsed   uint64  `json:"work_used"`
}

// StableRouteResult contains only deterministic, algorithmically meaningful
// fields. Runtime measurements and map iteration dependent representations are
// deliberately excluded.
type StableRouteResult struct {
	Found              bool              `json:"path_found"`
	Path               []core.NodeID     `json:"path"`
	Distance           string            `json:"path_cost"`
	SearchCompleted    bool              `json:"search_completed"`
	ReachabilityProven bool              `json:"reachability_proven"`
	Exact              bool              `json:"optimality_proven"`
	SolverName         string            `json:"solver_name"`
	Work               core.WorkMetrics  `json:"work"`
	WorkRelaxations    uint64            `json:"work_relaxations"`
	WorkExpandedNodes  uint64            `json:"work_expanded_nodes"`
	QueuePushes        uint64            `json:"queue_pushes"`
	QueuePops          uint64            `json:"queue_pops"`
	ParallelSteps      uint64            `json:"parallel_steps"`
	LowerBound         string            `json:"lower_bound"`
	CertifiedRatio     *string           `json:"proven_cost_ratio,omitempty"`
	QualityCertified   bool              `json:"quality_bound_proven"`
	FirstPathWork      *uint64           `json:"first_path_work,omitempty"`
	FallbackUsed       bool              `json:"fallback_used"`
	BudgetExhausted    bool              `json:"budget_exhausted"`
	DeadlineExceeded   bool              `json:"deadline_exceeded"`
	ErrorCode          core.ErrorCode    `json:"error_code,omitempty"`
	SolverTrace        []StableTaskTrace `json:"solver_trace,omitempty"`
}

func stableFloat(v float64) string {
	switch {
	case math.IsInf(v, 1):
		return "+Inf"
	case math.IsInf(v, -1):
		return "-Inf"
	case math.IsNaN(v):
		return "NaN"
	default:
		return fmt.Sprintf("%.17g", v)
	}
}

func StableResult(result core.RouteResult) StableRouteResult {
	var ratio *string
	if result.CertifiedRatio != nil {
		value := stableFloat(*result.CertifiedRatio)
		ratio = &value
	}
	traces := make([]StableTaskTrace, 0, len(result.SolverTrace))
	for _, trace := range result.SolverTrace {
		var budget *uint64
		if trace.Budget != nil {
			value := *trace.Budget
			budget = &value
		}
		traces = append(traces, StableTaskTrace{
			TaskID: trace.TaskID, Solver: trace.Solver, Purpose: trace.Purpose,
			Reason: trace.Reason, Allocation: stableFloat(trace.Allocation),
			Budget: budget, Found: trace.Found, Distance: stableFloat(trace.Distance),
			WorkUsed: trace.WorkUsed,
		})
	}
	return StableRouteResult{
		Found: result.Found, Path: append([]core.NodeID(nil), result.Path...),
		Distance: stableFloat(result.Distance), SearchCompleted: result.SearchCompleted,
		ReachabilityProven: result.ReachabilityProven, Exact: result.Exact,
		SolverName: result.SolverName, Work: result.Work, WorkRelaxations: result.WorkRelaxations,
		WorkExpandedNodes: result.WorkExpandedNodes, QueuePushes: result.QueuePushes,
		QueuePops: result.QueuePops, ParallelSteps: result.ParallelSteps,
		LowerBound: stableFloat(result.LowerBound), CertifiedRatio: ratio,
		QualityCertified: result.QualityCertified, FirstPathWork: result.FirstPathWork,
		FallbackUsed: result.FallbackUsed, BudgetExhausted: result.BudgetExhausted,
		DeadlineExceeded: result.DeadlineExceeded, ErrorCode: result.ErrorCode,
		SolverTrace: traces,
	}
}

func StableDigest(result core.RouteResult) (string, error) {
	payload, err := json.Marshal(StableResult(result))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

// VerifyRepeatability executes an identical route repeatedly and fails at the
// first semantic difference. TimeMS is intentionally ignored.
func VerifyRepeatability(ctx context.Context, runner RouteRunner, graph core.Graph, request core.RouteRequest, repetitions int) (core.RouteResult, string, error) {
	if repetitions < 2 {
		repetitions = 2
	}
	first, err := runner.Route(ctx, graph, request)
	if err != nil {
		return core.RouteResult{}, "", err
	}
	expected, err := StableDigest(first)
	if err != nil {
		return core.RouteResult{}, "", err
	}
	for run := 2; run <= repetitions; run++ {
		current, err := runner.Route(ctx, graph, request)
		if err != nil {
			return core.RouteResult{}, "", fmt.Errorf("repeat %d: %w", run, err)
		}
		digest, err := StableDigest(current)
		if err != nil {
			return core.RouteResult{}, "", err
		}
		if digest != expected {
			a, _ := json.Marshal(StableResult(first))
			b, _ := json.Marshal(StableResult(current))
			return core.RouteResult{}, "", fmt.Errorf("non-deterministic route result at repeat %d: expected %s got %s\nfirst=%s\ncurrent=%s", run, expected, digest, a, b)
		}
	}
	return first, expected, nil
}
