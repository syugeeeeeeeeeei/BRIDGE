package truss

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/anchor"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bolts"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

type TargetKind string

const (
	TargetKindSystem      TargetKind = "system"
	TargetKindAlgorithm   TargetKind = "algorithm"
	TargetKindBoltsSolver TargetKind = "bolts_solver"
)

type ExecuteOnceRequest struct {
	TargetID   string
	Source     core.NodeID
	Target     core.NodeID
	Mode       core.RouteMode
	DeadlineMS *float64
	WorkBudget *uint64
	Workers    int
	Seed       uint64
}

type ExecuteOnceResult struct {
	Result        core.RouteResult
	TargetID      string
	TargetKind    TargetKind
	ExecutionPath string
	SolverTimeMS  float64
	EndToEndMS    float64
}

type resolvedExecution struct {
	targetID      string
	targetKind    TargetKind
	executionPath string
	component     core.Component
	solver        solver
}

func (t *Truss) resolveExecutionTarget(id string) (resolvedExecution, error) {
	switch id {
	case "anchor":
		return resolvedExecution{
			targetID:      "anchor",
			targetKind:    TargetKindAlgorithm,
			executionPath: "execute_once",
			component:     core.ComponentAnchor,
			solver:        t.Anchor,
		}, nil
	case "emergency_approx":
		return resolvedExecution{
			targetID:      "emergency_approx",
			targetKind:    TargetKindBoltsSolver,
			executionPath: "route",
			component:     core.ComponentBolts,
			solver:        t.Emergency,
		}, nil
	default:
		s, err := bolts.Resolve(id)
		if err != nil {
			return resolvedExecution{}, fmt.Errorf("unknown execution target %q", id)
		}
		return resolvedExecution{
			targetID:      id,
			targetKind:    TargetKindBoltsSolver,
			executionPath: "execute_once",
			component:     core.ComponentBolts,
			solver:        s,
		}, nil
	}
}

func (t *Truss) runResolved(ctx context.Context, g core.Graph, req core.RouteRequest, target resolvedExecution, purpose, reason string, budget core.WorkBudget) core.RouteResult {
	if purpose == "fallback" || purpose == "reachability" {
		t.emit("fallback_started", map[string]any{"solver": target.solver.Name(), "purpose": purpose, "reason": reason, "budget": budget.MaxWork})
	}
	if purpose == "certification" {
		t.emit("certification_started", map[string]any{"solver": target.solver.Name(), "reason": reason, "budget": budget.MaxWork})
	}
	t.emit("component_started", map[string]any{
		"component": target.component,
		"solver":    target.solver.Name(),
		"purpose":   purpose,
		"reason":    reason,
		"budget":    budget.MaxWork,
	})
	result := target.solver.Solve(ctx, g, req, budget, t.Observer)
	if result.Telemetry == nil {
		result.Telemetry = map[string]any{}
	}
	result.Telemetry["target_id"] = target.targetID
	result.Telemetry["target_kind"] = string(target.targetKind)
	result.Telemetry["execution_path"] = target.executionPath
	result.Telemetry["solver_time_ms"] = result.TimeMS
	result.Telemetry["solver_time_ns"] = result.TimeBreakdown.SolverNS
	t.emit("component_finished", map[string]any{
		"component": target.component, "solver": target.solver.Name(), "purpose": purpose,
		"found": result.Found, "work": result.TotalWork(), "distance": result.Distance,
	})
	if purpose == "fallback" || purpose == "reachability" {
		t.emit("fallback_finished", map[string]any{"solver": target.solver.Name(), "purpose": purpose, "reason": reason, "found": result.Found, "work": result.TotalWork(), "distance": result.Distance})
	}
	if purpose == "certification" {
		t.emit("certification_finished", map[string]any{"solver": target.solver.Name(), "reason": reason, "found": result.Found, "exact": result.Exact, "work": result.TotalWork(), "distance": result.Distance})
	}
	return result
}

func (t *Truss) ExecuteOnce(ctx context.Context, g core.Graph, req ExecuteOnceRequest) (ExecuteOnceResult, error) {
	if req.TargetID == "" {
		return ExecuteOnceResult{}, fmt.Errorf("target_id is required")
	}
	routeReq := core.RouteRequest{
		Source:     req.Source,
		Target:     req.Target,
		Mode:       req.Mode,
		DeadlineMS: req.DeadlineMS,
		WorkBudget: req.WorkBudget,
		Workers:    req.Workers,
		Seed:       req.Seed,
	}
	if routeReq.Mode == "" {
		routeReq.Mode = core.ModeBalanced
	}
	if routeReq.Workers == 0 {
		routeReq.Workers = 1
	}
	if err := routeReq.Validate(g); err != nil {
		return ExecuteOnceResult{Result: core.RouteResult{Distance: math.Inf(1), ErrorCode: core.ErrInvalidRequest}}, err
	}
	target, err := t.resolveExecutionTarget(req.TargetID)
	if err != nil {
		return ExecuteOnceResult{}, err
	}
	start := time.Now()
	if routeReq.DeadlineMS != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*routeReq.DeadlineMS*float64(time.Millisecond)))
		defer cancel()
	}
	result := t.runResolved(ctx, g, routeReq, target, "single_run", "execute_once", core.WorkBudget{MaxWork: routeReq.WorkBudget})
	endToEndNS := time.Since(start).Nanoseconds()
	endToEndMS := float64(endToEndNS) / 1_000_000
	result.TimeBreakdown.TotalNS = endToEndNS
	result.TimeBreakdown.TrussNS = endToEndNS
	result.TimeBreakdown.TotalMS = endToEndMS
	result.TimeBreakdown.TrussMS = endToEndMS
	result.SolverName = target.targetID
	result.TimeMS = endToEndMS
	if ctx.Err() == context.DeadlineExceeded {
		result.DeadlineExceeded = true
		result.ErrorCode = core.ErrDeadlineExceeded
	}
	return ExecuteOnceResult{
		Result:        result,
		TargetID:      target.targetID,
		TargetKind:    target.targetKind,
		ExecutionPath: target.executionPath,
		SolverTimeMS:  telemetryFloat(result.Telemetry, "solver_time_ms", result.TimeMS),
		EndToEndMS:    endToEndMS,
	}, nil
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

var _ solver = anchor.Solver{}
