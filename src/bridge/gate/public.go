package gate

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

const (
	RouteRequestSchemaV1   = "bridge.route.request.v2"
	RouteResultSchemaV1    = "bridge.route.result.v2"
	ExecuteRequestSchemaV1 = "bridge.execute_once.request.v2"
	ExecuteResultSchemaV1  = "bridge.execute_once.result.v2"
)

type GraphNode struct {
	ID uint32   `json:"id"`
	X  *float64 `json:"x,omitempty"`
	Y  *float64 `json:"y,omitempty"`
}

type GraphEdge struct {
	From   uint32  `json:"from"`
	To     uint32  `json:"to"`
	Weight float64 `json:"weight"`
}

type GraphInput struct {
	Type     string      `json:"type"`
	Directed bool        `json:"directed,omitempty"`
	Nodes    []GraphNode `json:"nodes,omitempty"`
	Edges    []GraphEdge `json:"edges,omitempty"`
	Path     string      `json:"path,omitempty"`
}

type RouteInput struct {
	Source           uint32         `json:"source"`
	Target           uint32         `json:"target"`
	Mode             core.RouteMode `json:"route_mode,omitempty"`
	MaxSuboptimality *float64       `json:"max_suboptimality,omitempty"`
	Workers          int            `json:"logical_worker_count,omitempty"`
	Seed             uint64         `json:"seed,omitempty"`
}

type BudgetInput struct {
	TotalWork *uint64  `json:"total_work,omitempty"`
	TimeoutMS *float64 `json:"timeout_ms,omitempty"`
}

type ObservationMode string

const (
	ObservationOff     ObservationMode = "off"
	ObservationSummary ObservationMode = "summary"
	ObservationProfile ObservationMode = "profile"
	ObservationTrace   ObservationMode = "trace"
)

type AblationInput = core.AblationOptions

type ObservationInput struct {
	Mode       ObservationMode `json:"level,omitempty"`
	SampleRate *float64        `json:"sample_rate,omitempty"`
}

type RouteRequest struct {
	SchemaVersion string           `json:"schema_version"`
	RequestID     string           `json:"request_id,omitempty"`
	Graph         GraphInput       `json:"graph"`
	Route         RouteInput       `json:"route"`
	Budget        BudgetInput      `json:"budget,omitempty"`
	Observation   ObservationInput `json:"observation_config,omitempty"`
	Ablation      AblationInput    `json:"ablation,omitempty"`
}

type ExecuteTargetInput struct {
	ID string `json:"id"`
}

type ExecuteRequest struct {
	SchemaVersion string             `json:"schema_version"`
	RequestID     string             `json:"request_id,omitempty"`
	Target        ExecuteTargetInput `json:"target"`
	Graph         GraphInput         `json:"graph"`
	Route         RouteInput         `json:"route"`
	Budget        BudgetInput        `json:"budget,omitempty"`
	Observation   ObservationInput   `json:"observation_config,omitempty"`
	Ablation      AblationInput      `json:"ablation,omitempty"`
}

type ObservationReporter interface {
	ObservationMode() string
	ObservationEventCount() uint64
	ObservationDroppedEvents() uint64
	ObservationTruncated() bool
}

type ObservationOptions struct {
	Mode     ObservationMode
	Observer bearing.Observer
	Reporter ObservationReporter
}

type RouteOptions struct{ Observation ObservationOptions }

type ObservationResult struct {
	Mode          ObservationMode `json:"level"`
	EventCount    uint64          `json:"event_count"`
	DroppedEvents uint64          `json:"dropped_events,omitempty"`
	Truncated     bool            `json:"truncated"`
	OverheadNS    int64           `json:"overhead_ns,omitempty"`
	SinkWriteNS   int64           `json:"sink_write_ns,omitempty"`
	Summary       any             `json:"summary,omitempty"`
}

type RouteResult struct {
	SchemaVersion       string             `json:"schema_version"`
	RequestID           string             `json:"request_id,omitempty"`
	Status              string             `json:"status"`
	Found               bool               `json:"path_found"`
	SearchCompleted     bool               `json:"search_completed"`
	ReachabilityProven  bool               `json:"reachability_proven"`
	Distance            *float64           `json:"path_cost,omitempty"`
	Path                []uint32           `json:"path"`
	Exact               bool               `json:"optimality_proven"`
	SolverName          string             `json:"solver_name,omitempty"`
	Work                core.WorkMetrics   `json:"work"`
	SolverTimeMS        float64            `json:"solver_time_ms,omitempty"`
	TimeBreakdown       core.TimeBreakdown `json:"time_breakdown"`
	TimeMS              float64            `json:"end_to_end_time_ms"`
	ErrorCode           core.ErrorCode     `json:"error_code,omitempty"`
	Observation         *ObservationResult `json:"observation_data,omitempty"`
	FailureReason       string             `json:"failure_reason,omitempty"`
	TimeToFirstPathMS   *float64           `json:"first_path_elapsed_ms,omitempty"`
	TimeToBestFoundMS   *float64           `json:"best_path_elapsed_ms,omitempty"`
	ImprovementCount    uint64             `json:"improvement_count"`
	BridgeOverheadRatio float64            `json:"bridge_overhead_ratio,omitempty"`
	DuplicatedWorkRatio float64            `json:"duplicated_work_ratio,omitempty"`
	StateReuseRatio     float64            `json:"state_reuse_ratio,omitempty"`
	BudgetLedger        *core.BudgetLedger `json:"budget_ledger,omitempty"`
}

type ExecuteResult struct {
	SchemaVersion       string             `json:"schema_version"`
	RequestID           string             `json:"request_id,omitempty"`
	Status              string             `json:"status"`
	Found               bool               `json:"path_found"`
	SearchCompleted     bool               `json:"search_completed"`
	ReachabilityProven  bool               `json:"reachability_proven"`
	Distance            *float64           `json:"path_cost,omitempty"`
	Path                []uint32           `json:"path"`
	Exact               bool               `json:"optimality_proven"`
	SolverName          string             `json:"solver_name,omitempty"`
	TargetID            string             `json:"target_id"`
	TargetKind          string             `json:"target_kind"`
	ExecutionPath       string             `json:"execution_path"`
	Work                core.WorkMetrics   `json:"work"`
	SolverTimeMS        float64            `json:"solver_time_ms"`
	TimeBreakdown       core.TimeBreakdown `json:"time_breakdown"`
	EndToEndMS          float64            `json:"end_to_end_time_ms"`
	ErrorCode           core.ErrorCode     `json:"error_code,omitempty"`
	Observation         *ObservationResult `json:"observation_data,omitempty"`
	FailureReason       string             `json:"failure_reason,omitempty"`
	TimeToFirstPathMS   *float64           `json:"first_path_elapsed_ms,omitempty"`
	TimeToBestFoundMS   *float64           `json:"best_path_elapsed_ms,omitempty"`
	ImprovementCount    uint64             `json:"improvement_count"`
	BridgeOverheadRatio float64            `json:"bridge_overhead_ratio,omitempty"`
	DuplicatedWorkRatio float64            `json:"duplicated_work_ratio,omitempty"`
	StateReuseRatio     float64            `json:"state_reuse_ratio,omitempty"`
	BudgetLedger        *core.BudgetLedger `json:"budget_ledger,omitempty"`
}

type PublicError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *PublicError) Error() string { return e.Message }

// Router is the stable external boundary used by CLI, SDKs and servers.
type Router struct{}

func NewRouter() *Router { return &Router{} }

func (r *Router) Route(ctx context.Context, req RouteRequest, opts RouteOptions) (RouteResult, error) {
	gateStarted := time.Now()
	if req.SchemaVersion != RouteRequestSchemaV1 {
		return RouteResult{}, &PublicError{Code: "INVALID_SCHEMA_VERSION", Message: fmt.Sprintf("schema_version must be %q", RouteRequestSchemaV1)}
	}
	graphInput := req.Graph
	g, err := buildGraph(graphInput)
	if err != nil {
		return RouteResult{}, &PublicError{Code: "INVALID_GRAPH", Message: err.Error()}
	}
	mode := req.Route.Mode
	if mode == "" {
		mode = core.ModeBalanced
	}
	workers := req.Route.Workers
	if workers == 0 {
		workers = 1
	}
	internal := core.RouteRequest{Source: core.NodeID(req.Route.Source), Target: core.NodeID(req.Route.Target), Mode: mode, MaxSuboptimality: req.Route.MaxSuboptimality, DeadlineMS: req.Budget.TimeoutMS, WorkBudget: req.Budget.TotalWork, Workers: workers, Seed: req.Route.Seed, Ablation: req.Ablation}
	if err := internal.Validate(g); err != nil {
		return RouteResult{}, &PublicError{Code: "INVALID_REQUEST", Message: err.Error()}
	}
	obsMode, err := resolveObservationMode(opts, req.Observation.Mode)
	if err != nil {
		return RouteResult{}, err
	}
	observer := opts.Observation.Observer
	if observer == nil {
		observer = bearing.NullObserver{}
	}
	result, err := New(observer).Route(ctx, g, internal)
	result.TimeBreakdown.GateMS = float64(time.Since(gateStarted).Microseconds())/1000 - result.TimeBreakdown.TotalMS
	if result.TimeBreakdown.GateMS < 0 {
		result.TimeBreakdown.GateMS = 0
	}
	_ = obsMode
	observationResult := observationResultFromReporter(opts.Observation.Reporter)
	if err != nil {
		return RouteResult{}, err
	}
	searchCompleted := !result.BudgetExhausted && !result.DeadlineExceeded && result.ErrorCode != core.ErrCancelled
	out := RouteResult{SchemaVersion: RouteResultSchemaV1, RequestID: req.RequestID, Found: result.Found, SearchCompleted: searchCompleted, ReachabilityProven: searchCompleted, Exact: result.Exact, SolverName: result.SolverName, Work: result.Work, SolverTimeMS: telemetryFloat(result.Telemetry, "solver_time_ms", result.TimeMS), TimeBreakdown: result.TimeBreakdown, TimeMS: result.TimeMS, ErrorCode: result.ErrorCode, Observation: observationResult, FailureReason: result.FailureReason, TimeToFirstPathMS: result.TimeToFirstPathMS, TimeToBestFoundMS: result.TimeToBestFoundMS, ImprovementCount: result.ImprovementCount, BridgeOverheadRatio: telemetryFloat(result.Telemetry, "bridge_overhead_ratio", 0), DuplicatedWorkRatio: telemetryFloat(result.Telemetry, "duplicated_work_ratio", 0), StateReuseRatio: telemetryFloat(result.Telemetry, "state_reuse_ratio", 0), BudgetLedger: result.BudgetLedger, Path: make([]uint32, len(result.Path))}
	for i, n := range result.Path {
		out.Path[i] = uint32(n)
	}
	switch {
	case result.Found:
		out.Status = "found"
		d := result.Distance
		out.Distance = &d
	case result.BudgetExhausted:
		out.Status = "budget_exhausted"
	case result.DeadlineExceeded:
		out.Status = "timeout"
	default:
		out.Status = "unreachable"
	}
	return out, nil
}

func (r *Router) ExecuteOnce(ctx context.Context, req ExecuteRequest, opts RouteOptions) (ExecuteResult, error) {
	gateStarted := time.Now()
	if req.SchemaVersion != ExecuteRequestSchemaV1 {
		return ExecuteResult{}, &PublicError{Code: "INVALID_SCHEMA_VERSION", Message: fmt.Sprintf("schema_version must be %q", ExecuteRequestSchemaV1)}
	}
	g, err := buildGraph(req.Graph)
	if err != nil {
		return ExecuteResult{}, &PublicError{Code: "INVALID_GRAPH", Message: err.Error()}
	}
	obsMode, err := resolveObservationMode(opts, req.Observation.Mode)
	if err != nil {
		return ExecuteResult{}, err
	}
	_ = obsMode
	observer := opts.Observation.Observer
	if observer == nil {
		observer = bearing.NullObserver{}
	}
	internal := ExecuteOnceRequest{
		TargetID:   req.Target.ID,
		Source:     core.NodeID(req.Route.Source),
		Target:     core.NodeID(req.Route.Target),
		Mode:       req.Route.Mode,
		DeadlineMS: req.Budget.TimeoutMS,
		WorkBudget: req.Budget.TotalWork,
		Workers:    req.Route.Workers,
		Seed:       req.Route.Seed,
		Ablation:   req.Ablation,
	}
	result, err := New(observer).ExecuteOnce(ctx, g, internal)
	result.Result.TimeBreakdown.GateMS = float64(time.Since(gateStarted).Microseconds())/1000 - result.Result.TimeBreakdown.TotalMS
	if result.Result.TimeBreakdown.GateMS < 0 {
		result.Result.TimeBreakdown.GateMS = 0
	}
	if err != nil {
		code := "INVALID_REQUEST"
		if internal.TargetID == "" || result.Result.ErrorCode == "" {
			code = "INVALID_TARGET"
		}
		return ExecuteResult{}, &PublicError{Code: code, Message: err.Error()}
	}
	observationResult := observationResultFromReporter(opts.Observation.Reporter)
	out := ExecuteResult{
		SchemaVersion:      ExecuteResultSchemaV1,
		RequestID:          req.RequestID,
		Found:              result.Result.Found,
		SearchCompleted:    !result.Result.BudgetExhausted && !result.Result.DeadlineExceeded && result.Result.ErrorCode != core.ErrCancelled,
		ReachabilityProven: !result.Result.BudgetExhausted && !result.Result.DeadlineExceeded && result.Result.ErrorCode != core.ErrCancelled,
		Exact:              result.Result.Exact,
		SolverName:         result.Result.SolverName,
		TargetID:           result.TargetID,
		TargetKind:         string(result.TargetKind),
		ExecutionPath:      result.ExecutionPath,
		Work:               result.Result.Work,
		SolverTimeMS:       result.SolverTimeMS,
		TimeBreakdown:      result.Result.TimeBreakdown,
		EndToEndMS:         result.EndToEndMS,
		ErrorCode:          result.Result.ErrorCode,
		Observation:        observationResult,
		BudgetLedger:       result.Result.BudgetLedger,
		Path:               make([]uint32, len(result.Result.Path)),
	}
	for i, n := range result.Result.Path {
		out.Path[i] = uint32(n)
	}
	switch {
	case result.Result.Found:
		out.Status = "found"
		d := result.Result.Distance
		out.Distance = &d
	case result.Result.BudgetExhausted:
		out.Status = "budget_exhausted"
	case result.Result.DeadlineExceeded:
		out.Status = "timeout"
	default:
		out.Status = "unreachable"
	}
	return out, nil
}

func resolveObservationMode(opts RouteOptions, fallback ObservationMode) (ObservationMode, error) {
	obsMode := opts.Observation.Mode
	if obsMode == "" {
		obsMode = fallback
	}
	if obsMode == "" {
		obsMode = ObservationOff
	}
	switch obsMode {
	case ObservationOff, ObservationSummary, ObservationTrace, ObservationProfile:
		return obsMode, nil
	default:
		return "", &PublicError{Code: "INVALID_OBSERVATION", Message: fmt.Sprintf("unsupported observation mode %q", obsMode)}
	}
}

func observationResultFromReporter(reporter ObservationReporter) *ObservationResult {
	if reporter == nil {
		return nil
	}
	out := &ObservationResult{Mode: ObservationMode(reporter.ObservationMode()), EventCount: reporter.ObservationEventCount(), DroppedEvents: reporter.ObservationDroppedEvents(), Truncated: reporter.ObservationTruncated()}
	if r, ok := reporter.(interface{ ObservationOverheadNS() int64 }); ok {
		out.OverheadNS = r.ObservationOverheadNS()
	}
	if r, ok := reporter.(interface{ ObservationSinkWriteNS() int64 }); ok {
		out.SinkWriteNS = r.ObservationSinkWriteNS()
	}
	if r, ok := reporter.(interface{ ObservationSummary() any }); ok {
		out.Summary = r.ObservationSummary()
	}
	return out
}

func telemetryFloat(m map[string]any, k string, fallback float64) float64 {
	if m == nil {
		return fallback
	}
	switch v := m[k].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case uint64:
		return float64(v)
	default:
		return fallback
	}
}

func buildGraph(in GraphInput) (*core.AdjacencyGraph, error) {
	if in.Type != "inline" {
		return nil, fmt.Errorf("graph.type must be inline or file")
	}
	if len(in.Nodes) == 0 {
		return nil, errors.New("graph.nodes must not be empty")
	}
	ids := map[uint32]struct{}{}
	max := uint32(0)
	for _, n := range in.Nodes {
		if _, ok := ids[n.ID]; ok {
			return nil, fmt.Errorf("duplicate node id: %d", n.ID)
		}
		ids[n.ID] = struct{}{}
		if n.ID > max {
			max = n.ID
		}
	}
	if int(max)+1 != len(in.Nodes) {
		return nil, errors.New("node ids must be contiguous from 0")
	}
	g := core.NewAdjacencyGraph(len(in.Nodes), in.Directed)
	for _, n := range in.Nodes {
		if (n.X == nil) != (n.Y == nil) {
			return nil, fmt.Errorf("node %d must provide both x and y", n.ID)
		}
		if n.X != nil {
			if !finite(*n.X) || !finite(*n.Y) {
				return nil, fmt.Errorf("node %d coordinates must be finite", n.ID)
			}
			_ = g.SetPosition(core.NodeID(n.ID), core.Point{X: *n.X, Y: *n.Y})
		}
	}
	for _, e := range in.Edges {
		if _, ok := ids[e.From]; !ok {
			return nil, fmt.Errorf("edge references missing node %d", e.From)
		}
		if _, ok := ids[e.To]; !ok {
			return nil, fmt.Errorf("edge references missing node %d", e.To)
		}
		if !finite(e.Weight) || e.Weight <= 0 {
			return nil, fmt.Errorf("edge weight must be finite and positive")
		}
		if err := g.AddEdge(core.NodeID(e.From), core.NodeID(e.To), e.Weight); err != nil {
			return nil, err
		}
	}
	return g, nil
}
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
