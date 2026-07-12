package truss

import (
	"context"
	"math"
	"time"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/anchor"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bolts"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

type Truss struct {
	Observer      bearing.Observer
	Anchor        anchor.Solver
	Dijkstra      bolts.BidirectionalDijkstra
	Reachability  bolts.Reachability
	Emergency     bolts.WeightedAStar
	BudgetFactory func(*uint64) *Budget
	Supervisor    Supervisor
	Arbiter       Arbiter
}

func New(o bearing.Observer) *Truss {
	if o == nil {
		o = bearing.NullObserver{}
	}
	return &Truss{Observer: bearing.SafeObserver{Inner: o}, Anchor: anchor.Solver{Config: anchor.DefaultConfig()}, Emergency: bolts.WeightedAStar{Weight: 1.12}, BudgetFactory: NewBudget}
}

type solver interface {
	Name() string
	Solve(context.Context, core.Graph, core.RouteRequest, core.WorkBudget, bearing.Observer) core.RouteResult
}

func (t *Truss) emit(kind string, attrs map[string]any) {
	t.Observer.Observe(bearing.Event{TaskID: "truss", Kind: kind, Phase: "orchestration", Attributes: attrs})
}

func (t *Truss) Route(ctx context.Context, g core.Graph, r core.RouteRequest) (core.RouteResult, error) {
	if r.Workers == 0 {
		r.Workers = 1
	}
	if r.Mode == "" {
		r.Mode = core.ModeBalanced
	}
	if err := r.Validate(g); err != nil {
		return core.RouteResult{Distance: math.Inf(1), ErrorCode: core.ErrInvalidRequest}, err
	}
	start := time.Now()
	if r.DeadlineMS != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*r.DeadlineMS*float64(time.Millisecond)))
		defer cancel()
	}
	ledger := t.BudgetFactory(r.WorkBudget)
	var portfolio core.WorkMetrics
	var totalRelax, totalExpand, totalPush, totalPop uint64
	var investigatedNodes, investigatedEdges, candidatePaths, pathNodes uint64
	componentNodeIDs := map[core.Component][]uint64{}
	componentEdgeIDs := map[core.Component][]uint64{}
	var anchorTimeMS, boltsTimeMS, fallbackTimeMS, supervisorTimeMS, arbiterTimeMS float64
	traces := []core.TaskTrace{}
	run := func(id, purpose, reason string, target resolvedExecution, limit *uint64) core.RouteResult {
		grant := ledger.Grant(limit)
		res := t.runResolved(ctx, g, r, target, purpose, reason, grant)
		ledger.Consume(target.component, res.TotalWork())
		ledger.Record(id, target.component, purpose, grant, res.TotalWork())
		portfolio.Add(res.Work)
		totalRelax += res.WorkRelaxations
		totalExpand += res.WorkExpandedNodes
		totalPush += res.QueuePushes
		totalPop += res.QueuePops
		investigatedNodes += telemetryUint(res.Telemetry, "investigated_nodes")
		investigatedEdges += telemetryUint(res.Telemetry, "investigated_edges")
		componentNodeIDs[target.component] = append(componentNodeIDs[target.component], telemetryIDs(res.Telemetry, "investigated_node_ids")...)
		componentEdgeIDs[target.component] = append(componentEdgeIDs[target.component], telemetryIDs(res.Telemetry, "investigated_edge_ids")...)
		if target.component == core.ComponentAnchor {
			anchorTimeMS += res.TimeMS
		} else {
			boltsTimeMS += res.TimeMS
			if purpose == "fallback" || purpose == "reachability" {
				fallbackTimeMS += res.TimeMS
			}
		}
		candidatePaths += telemetryUint(res.Telemetry, "candidate_paths")
		if v := telemetryUint(res.Telemetry, "path_node_count"); v > pathNodes {
			pathNodes = v
		}
		traces = append(traces, core.TaskTrace{TaskID: id, Solver: target.solver.Name(), Purpose: purpose, Reason: reason, Budget: grant.MaxWork, Found: res.Found, Distance: res.Distance, WorkUsed: res.TotalWork()})
		return res
	}
	anchorTarget, _ := t.resolveExecutionTarget("anchor")
	exactTarget := resolvedExecution{targetID: "bidirectional_dijkstra", targetKind: TargetKindBoltsSolver, executionPath: "route", component: core.ComponentBolts, solver: t.Dijkstra}
	reachabilityTarget := resolvedExecution{targetID: "reachability", targetKind: TargetKindBoltsSolver, executionPath: "route", component: core.ComponentBolts, solver: t.Reachability}
	emergencyTarget := resolvedExecution{targetID: "emergency_approx", targetKind: TargetKindBoltsSolver, executionPath: "route", component: core.ComponentBolts, solver: t.Emergency}
	var best core.RouteResult
	if r.Mode == core.ModeExact {
		best = run("bolts-exact", "certification", "exact_mode", exactTarget, nil)
	} else {
		strategy := anchor.Analyze(g)
		r.AnchorStrategy = strategy
		best = run("anchor-primary", "first_path", "strategy:"+strategy, anchorTarget, nil)
		supervisorStarted := time.Now()
		progress, emergency := t.Supervisor.Inspect(core.ComponentAnchor, g, best)
		supervisorTimeMS += float64(time.Since(supervisorStarted).Microseconds()) / 1000
		t.emit("progress_reported", map[string]any{"component": progress.Component, "state": progress.State, "node_ratio": progress.NodeRatio, "edge_ratio": progress.EdgeRatio, "duplicates": progress.DuplicateInvestigated})
		if emergency != nil {
			t.emit("emergency_reported", map[string]any{"component": emergency.Component, "kind": emergency.Kind, "evidence": emergency.Evidence})
			directive := t.Supervisor.Recommend(emergency, ledger.Remaining())
			t.emit("directive_issued", map[string]any{"kind": directive.Kind, "capability": directive.Capability, "reason": directive.Reason})
			if directive.Kind == core.DirectiveStartBolts && !r.Ablation.DisableFallback {
				var recovery core.RouteResult
				if directive.Capability == "CHECK_REACHABILITY" {
					recovery = run("bolts-reachability", "reachability", directive.Reason, reachabilityTarget, nil)
					if !recovery.Exact || recovery.Found {
						recovery = run("bolts-recovery", "fallback", "reachability_not_proven", emergencyTarget, nil)
					}
				} else {
					recovery = run("bolts-recovery", "fallback", directive.Reason, emergencyTarget, nil)
				}
				arbiterStarted := time.Now()
				best = t.Arbiter.Choose(best, recovery)
				arbiterTimeMS += float64(time.Since(arbiterStarted).Microseconds()) / 1000
			}
		}
		if r.Mode == core.ModeQuality && !r.Ablation.DisableCertification && (ledger.Remaining() == nil || *ledger.Remaining() > 0) {
			exact := run("bolts-certify", "certification", "quality_mode", exactTarget, nil)
			arbiterStarted := time.Now()
			best = t.Arbiter.Choose(best, exact)
			arbiterTimeMS += float64(time.Since(arbiterStarted).Microseconds()) / 1000
		}
	}
	best.SolverName = "bridge_" + string(r.Mode)
	best.Work = portfolio
	best.Work.TotalActions = ledger.Used()
	best.WorkRelaxations = totalRelax
	best.WorkExpandedNodes = totalExpand
	best.QueuePushes = totalPush
	best.QueuePops = totalPop
	best.ParallelSteps = portfolio.ScheduledSteps
	best.SolverTrace = traces
	ledgerSnapshot := ledger.Snapshot()
	best.BudgetLedger = &ledgerSnapshot
	best.TimeMS = float64(time.Since(start).Microseconds()) / 1000
	best.TimeBreakdown = core.TimeBreakdown{
		TotalMS: best.TimeMS, SolverMS: anchorTimeMS + boltsTimeMS, TrussMS: best.TimeMS,
		AnchorMS: anchorTimeMS, BoltsMS: boltsTimeMS, FallbackMS: fallbackTimeMS,
		SupervisorMS: supervisorTimeMS, ArbiterMS: arbiterTimeMS,
		OrchestrationMS: math.Max(0, best.TimeMS-anchorTimeMS-boltsTimeMS),
	}
	best.FallbackUsed = ledger.ComponentUsed(core.ComponentBolts) > 0 && r.Mode != core.ModeExact
	best.BudgetExhausted = r.WorkBudget != nil && ledger.Used() >= *r.WorkBudget
	if ctx.Err() == context.DeadlineExceeded {
		best.DeadlineExceeded = true
		best.ErrorCode = core.ErrDeadlineExceeded
	}
	if best.Exact && best.Found {
		x := 1.0
		best.LowerBound = best.Distance
		best.CertifiedRatio = &x
		best.QualityCertified = true
	}
	if best.Telemetry == nil {
		best.Telemetry = map[string]any{}
	}
	portfolioUniqueNodes, duplicateNodes := unionAndOverlap(componentNodeIDs[core.ComponentAnchor], componentNodeIDs[core.ComponentBolts])
	portfolioUniqueEdges, duplicateEdges := unionAndOverlap(componentEdgeIDs[core.ComponentAnchor], componentEdgeIDs[core.ComponentBolts])
	best.Telemetry["architecture"] = "TRUSS(Orchestrator/Budget/Supervisor/Arbiter)/ANCHOR/BOLTS/BEARING"
	best.Telemetry["portfolio_work_used"] = ledger.Used()
	best.Telemetry["anchor_work"] = ledger.ComponentUsed(core.ComponentAnchor)
	best.Telemetry["bolts_work"] = ledger.ComponentUsed(core.ComponentBolts)
	best.Telemetry["investigated_nodes"] = investigatedNodes
	best.Telemetry["portfolio_unique_nodes"] = portfolioUniqueNodes
	best.Telemetry["portfolio_unique_node_ratio"] = float64(portfolioUniqueNodes) / float64(maxInt(1, g.NodeCount()))
	best.Telemetry["cross_component_duplicate_nodes"] = duplicateNodes
	best.Telemetry["portfolio_unique_edges"] = portfolioUniqueEdges
	best.Telemetry["portfolio_unique_edge_ratio"] = float64(portfolioUniqueEdges) / float64(maxInt(1, edgeSlots(g)))
	best.Telemetry["cross_component_duplicate_edges"] = duplicateEdges
	best.Telemetry["anchor_time_ms"] = anchorTimeMS
	best.Telemetry["bolts_time_ms"] = boltsTimeMS
	best.Telemetry["supervisor_time_ms"] = supervisorTimeMS
	best.Telemetry["arbiter_time_ms"] = arbiterTimeMS
	best.Telemetry["orchestration_overhead_ms"] = math.Max(0, best.TimeMS-anchorTimeMS-boltsTimeMS)
	best.Telemetry["investigated_node_ratio"] = float64(investigatedNodes) / float64(maxInt(1, g.NodeCount()))
	best.Telemetry["investigated_edges"] = investigatedEdges
	best.Telemetry["investigated_edge_ratio"] = float64(investigatedEdges) / float64(maxInt(1, edgeSlots(g)))
	best.Telemetry["candidate_paths"] = candidatePaths
	best.Telemetry["path_node_count"] = pathNodes
	best.Telemetry["requested_mode"] = r.Mode
	best.Telemetry["anchor_strategy"] = r.AnchorStrategy
	if !best.Found {
		switch {
		case best.DeadlineExceeded:
			best.FailureReason = "timeout"
		case best.BudgetExhausted:
			best.FailureReason = "budget_exhausted"
		case best.ErrorCode == core.ErrNoPath:
			best.FailureReason = "disconnected"
		case best.FallbackUsed:
			best.FailureReason = "fallback_failure"
		default:
			best.FailureReason = "no_path"
		}
	}
	for _, tr := range traces {
		if tr.Found {
			v := anchorTimeMS + boltsTimeMS
			if best.TimeToFirstPathMS == nil {
				best.TimeToFirstPathMS = &v
			}
			best.TimeToBestFoundMS = &v
			best.ImprovementCount++
		}
	}
	if best.ImprovementCount > 0 {
		best.ImprovementCount--
	}
	if best.TimeMS > 0 {
		best.Telemetry["bridge_overhead_ratio"] = best.TimeBreakdown.OrchestrationMS / best.TimeMS
	}
	if portfolio.TotalActions > 0 {
		best.Telemetry["duplicated_work_ratio"] = float64(duplicateNodes+duplicateEdges) / float64(portfolio.TotalActions)
	}
	best.Telemetry["state_reuse_ratio"] = 0.0
	best.Telemetry["target_id"] = "bridge"
	best.Telemetry["target_kind"] = string(TargetKindSystem)
	best.Telemetry["execution_path"] = "route"
	best.Telemetry["solver_time_ms"] = anchorTimeMS + boltsTimeMS
	return best, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func telemetryIDs(m map[string]any, k string) []uint64 {
	if m == nil {
		return nil
	}
	switch v := m[k].(type) {
	case []uint64:
		return v
	case []any:
		out := make([]uint64, 0, len(v))
		for _, x := range v {
			if f, ok := x.(float64); ok {
				out = append(out, uint64(f))
			}
		}
		return out
	}
	return nil
}
func unionAndOverlap(a, b []uint64) (int, int) {
	if len(b) == 0 {
		return len(a), 0
	}
	seen := make(map[uint64]struct{}, len(a)+len(b))
	for _, x := range a {
		seen[x] = struct{}{}
	}
	overlap := 0
	for _, x := range b {
		if _, ok := seen[x]; ok {
			overlap++
		}
		seen[x] = struct{}{}
	}
	return len(seen), overlap
}
