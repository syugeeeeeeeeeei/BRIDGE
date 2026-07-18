package gate

import (
	"context"
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"time"
)

type Router struct{}

func NewRouter() *Router { return &Router{} }

func (r *Router) Route(ctx context.Context, req RouteRequest, opts RouteOptions) (RouteResult, error) {
	gateStarted := time.Now()
	observer := opts.Observation.Observer
	if observer == nil {
		observer = bearing.NullObserver{}
	}
	runID := req.RequestID
	requestSpan, finishRequest := bearing.BeginLifecycle(observer, runID, "gate-route", "", "GATE", "route_request")
	defer finishRequest(false)
	if req.SchemaVersion != RouteRequestSchemaV1 {
		return RouteResult{}, &PublicError{Code: "INVALID_SCHEMA_VERSION", Message: fmt.Sprintf("schema_version must be %q", RouteRequestSchemaV1)}
	}
	graphInput := req.Graph
	_, finishGraphBuild := bearing.BeginLifecycle(observer, runID, "gate-route", requestSpan, "GATE", "graph_build")
	g, err := buildGraph(graphInput)
	finishGraphBuild(err != nil)
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
	obsMode, err := resolveObservationMode(opts, req.Observation.Mode)
	if err != nil {
		return RouteResult{}, err
	}
	internal := core.RouteRequest{Source: core.NodeID(req.Route.Source), Target: core.NodeID(req.Route.Target), Mode: mode, MaxSuboptimality: req.Route.MaxSuboptimality, DeadlineMS: req.Budget.TimeoutMS, WorkBudget: req.Budget.TotalWork, Workers: workers, Seed: req.Route.Seed, Ablation: req.Ablation, CollectProgressSamples: obsMode == ObservationDebug, HandoffWorkThreshold: req.Route.HandoffWorkThreshold}
	_, finishValidation := bearing.BeginLifecycle(observer, runID, "gate-route", requestSpan, "GATE", "validation")
	validationErr := internal.Validate(g)
	finishValidation(validationErr != nil)
	if err := validationErr; err != nil {
		return RouteResult{}, &PublicError{Code: "INVALID_REQUEST", Message: err.Error()}
	}
	_, finishDispatch := bearing.BeginLifecycle(observer, runID, "gate-route", requestSpan, "GATE", "dispatch")
	result, err := New(observer).Route(ctx, g, internal)
	finishDispatch(err != nil)
	publicNS := time.Since(gateStarted).Nanoseconds()
	result.TimeBreakdown.GateNS = publicNS
	result.TimeBreakdown.GateMS = float64(publicNS) / 1_000_000
	_ = obsMode
	if err != nil {
		return RouteResult{}, err
	}
	searchCompleted := result.SearchCompleted && !result.BudgetExhausted && !result.DeadlineExceeded && result.ErrorCode != core.ErrCancelled
	_, finishResultConversion := bearing.BeginLifecycle(observer, runID, "gate-route", requestSpan, "GATE", "result_conversion")
	out := RouteResult{SchemaVersion: RouteResultSchemaV1, RequestID: req.RequestID, Found: result.Found, SearchCompleted: searchCompleted, ReachabilityProven: result.Found || result.ReachabilityProven, Exact: result.Exact, SolverName: result.SolverName, Work: result.Work, SolverTimeMS: telemetryFloat(result.Telemetry, "solver_time_ms", result.TimeMS), TimeBreakdown: result.TimeBreakdown, TimeMS: result.TimeMS, ErrorCode: result.ErrorCode, Observation: nil, FailureReason: result.FailureReason, TimeToFirstPathMS: result.TimeToFirstPathMS, TimeToBestFoundMS: result.TimeToBestFoundMS, ImprovementCount: result.ImprovementCount, BridgeOverheadRatio: telemetryFloat(result.Telemetry, "bridge_overhead_ratio", 0), DuplicatedWorkRatio: telemetryFloat(result.Telemetry, "duplicated_work_ratio", 0), StateReuseRatio: telemetryFloat(result.Telemetry, "state_reuse_ratio", 0), BudgetLedger: result.BudgetLedger, HandoffMetrics: result.HandoffMetrics, BottleneckProfile: result.BottleneckProfile, Path: make([]uint32, len(result.Path))}
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
	finishResultConversion(false)
	finishRequest(false)
	out.Observation = observationResultFromReporter(opts.Observation.Reporter)
	return out, nil
}

func (r *Router) ExecuteOnce(ctx context.Context, req ExecuteRequest, opts RouteOptions) (ExecuteResult, error) {
	gateStarted := time.Now()
	observer := opts.Observation.Observer
	if observer == nil {
		observer = bearing.NullObserver{}
	}
	runID := req.RequestID
	requestSpan, finishRequest := bearing.BeginLifecycle(observer, runID, "gate-execute-once", "", "GATE", "execute_once_request")
	defer finishRequest(false)
	if req.SchemaVersion != ExecuteRequestSchemaV1 {
		return ExecuteResult{}, &PublicError{Code: "INVALID_SCHEMA_VERSION", Message: fmt.Sprintf("schema_version must be %q", ExecuteRequestSchemaV1)}
	}
	_, finishGraphBuild := bearing.BeginLifecycle(observer, runID, "gate-execute-once", requestSpan, "GATE", "graph_build")
	g, err := buildGraph(req.Graph)
	finishGraphBuild(err != nil)
	if err != nil {
		return ExecuteResult{}, &PublicError{Code: "INVALID_GRAPH", Message: err.Error()}
	}
	obsMode, err := resolveObservationMode(opts, req.Observation.Mode)
	if err != nil {
		return ExecuteResult{}, err
	}
	_ = obsMode
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
	_, finishDispatch := bearing.BeginLifecycle(observer, runID, "gate-execute-once", requestSpan, "GATE", "dispatch")
	result, err := New(observer).ExecuteOnce(ctx, g, internal)
	finishDispatch(err != nil)
	publicNS := time.Since(gateStarted).Nanoseconds()
	result.Result.TimeBreakdown.GateNS = publicNS
	result.Result.TimeBreakdown.GateMS = float64(publicNS) / 1_000_000
	if err != nil {
		code := "INVALID_REQUEST"
		if internal.TargetID == "" || result.Result.ErrorCode == "" {
			code = "INVALID_TARGET"
		}
		return ExecuteResult{}, &PublicError{Code: code, Message: err.Error()}
	}
	out := ExecuteResult{
		SchemaVersion:      ExecuteResultSchemaV1,
		RequestID:          req.RequestID,
		Found:              result.Result.Found,
		SearchCompleted:    result.Result.SearchCompleted && !result.Result.BudgetExhausted && !result.Result.DeadlineExceeded && result.Result.ErrorCode != core.ErrCancelled,
		ReachabilityProven: result.Result.Found || result.Result.ReachabilityProven,
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
		Observation:        nil,
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
	finishRequest(false)
	out.Observation = observationResultFromReporter(opts.Observation.Reporter)
	return out, nil
}
