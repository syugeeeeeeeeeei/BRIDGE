package truss

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/anchor"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bolts"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

type solver interface {
	Name() string
	Solve(context.Context, core.Graph, core.RouteRequest, core.WorkBudget, bearing.Observer) core.RouteResult
}

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

func (t *Truss) emit(kind string, attrs map[string]any) {
	t.Observer.Observe(bearing.Event{TaskID: "truss", Kind: kind, Phase: "orchestration", Attributes: attrs})
}

func anchorWeight(mode core.RouteMode) float64 {
	switch mode {
	case core.ModeFast:
		return 2.75
	case core.ModeBalanced:
		return 1.80
	case core.ModeQuality:
		return 1.55
	case core.ModeExact:
		return 1.0
	default:
		return 1.80
	}
}

func shouldReturnCandidate(r core.RouteRequest, progress anchor.SessionProgress, used uint64, limit *uint64) bool {
	if !progress.Found {
		return false
	}
	if r.Mode == core.ModeFast || r.Mode == core.ModeBalanced {
		return true
	}
	if r.MaxSuboptimality != nil && progress.LowerBound > 0 && !math.IsInf(progress.LowerBound, 0) && progress.BestDistance/progress.LowerBound <= *r.MaxSuboptimality {
		return true
	}
	if limit != nil && used >= *limit {
		return true
	}
	return false
}

func handoffBudget(nodeCount int, transferred uint64, remaining *uint64) uint64 {
	grant := uint64(maxInt(64, nodeCount/4))
	stateBound := transferred*8 + 64
	if stateBound < grant {
		grant = stateBound
	}
	if remaining != nil && *remaining < grant {
		grant = *remaining
	}
	return grant
}

func handoffTelemetryUint(m map[string]any, key string) uint64 {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case uint64:
		return v
	case int:
		if v > 0 {
			return uint64(v)
		}
	case float64:
		if v > 0 {
			return uint64(v)
		}
	}
	return 0
}

func (t *Truss) Route(ctx context.Context, g core.Graph, r core.RouteRequest) (core.RouteResult, error) {
	requestAdaptationSpan := bearing.StartLifecycle(t.Observer, "", "bridge-route", "", "TRUSS", "request_adaptation")
	if r.Workers == 0 {
		r.Workers = 1
	}
	if r.Mode == "" {
		r.Mode = core.ModeBalanced
	}
	if err := r.Validate(g); err != nil {
		requestAdaptationSpan.Finish(true)
		return core.RouteResult{Distance: math.Inf(1), ErrorCode: core.ErrInvalidRequest, TerminationStatus: core.TerminationInvalid}, err
	}
	requestAdaptationSpan.Finish(false)
	started := time.Now()
	routeLifecycle := bearing.StartLifecycle(t.Observer, "", "bridge-route", "", "TRUSS", "route")
	defer routeLifecycle.Finish(false)
	routeSpan := routeLifecycle.ID()
	deadlineSetupSpan := bearing.StartLifecycle(t.Observer, "", "bridge-route", routeSpan, "TRUSS", "deadline_setup")
	if r.DeadlineMS != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*r.DeadlineMS*float64(time.Millisecond)))
		defer cancel()
	}
	deadlineSetupSpan.Finish(false)

	budgetSetupSpan := bearing.StartLifecycle(t.Observer, "", "bridge-route", routeSpan, "TRUSS", "budget_setup")
	budget := NewBudgetManager(r.WorkBudget)
	budgetSetupSpan.Finish(false)
	observerSetupSpan := bearing.StartLifecycle(t.Observer, "", "bridge-route", routeSpan, "TRUSS", "observer_setup")
	routeObserver := newSequencedObserver(t.Observer)
	if bearing.Wants(routeObserver, "search_started") {
		routeObserver.Observe(bearing.Event{TaskID: "bridge-route", Component: "TRUSS", Phase: "orchestration", Kind: "search_started", Attributes: map[string]any{"mode": string(r.Mode)}})
	}
	observerSetupSpan.Finish(false)
	policySetupSpan := bearing.StartLifecycle(t.Observer, "", "bridge-route", routeSpan, "TRUSS", "policy_setup")
	weight := anchor.RecommendedWeight(g, r.Mode)
	if budget.Grant(1) < 1 || !budget.Consume(1) {
		policySetupSpan.Finish(true)
		return core.RouteResult{Distance: math.Inf(1), BudgetExhausted: true, ErrorCode: core.ErrBudgetExhausted, TerminationStatus: core.TerminationUnknownBudget}, nil
	}
	policySetupSpan.Finish(false)
	anchorLifecycle := bearing.StartLifecycle(t.Observer, "", "anchor-session", routeSpan, "ANCHOR", "solve")
	anchorSpan := anchorLifecycle.ID()
	sessionCreationSpan := bearing.StartLifecycle(t.Observer, "", "bridge-route", anchorSpan, "TRUSS", "session_creation")
	session, err := anchor.NewHypothesisSession(g, r, routeObserver, "main", "adaptive_fast_path", weight)
	if err != nil {
		sessionCreationSpan.Finish(true)
		anchorLifecycle.Finish(true)
		return core.RouteResult{}, err
	}
	sessionCreationSpan.Finish(false)
	defer anchorLifecycle.Finish(true)

	traces := make([]core.TaskTrace, 1, 8)
	traces[0] = core.TaskTrace{TaskID: "anchor-main-init", Solver: "anchor/session", Purpose: "initialization", Reason: "single adaptive session", WorkUsed: 1}
	var anchorNS, boltsNS int64
	var epochs, handoffs, reused uint64
	handoffMetrics := &core.HandoffMetrics{}
	var control core.WorkMetrics
	var boltsWork core.WorkMetrics
	const normalEpochGrant uint64 = 512
	stagnationThreshold := uint64(g.NodeCount() / 4)
	if stagnationThreshold < 64 {
		stagnationThreshold = 64
	}
	if stagnationThreshold > 512 {
		stagnationThreshold = 512
	}
	if r.HandoffWorkThreshold != nil {
		stagnationThreshold = *r.HandoffWorkThreshold
	}

	adaptiveExecutionSpan := bearing.StartLifecycle(t.Observer, "", "bridge-route", anchorSpan, "TRUSS", "adaptive_execution")
	defer adaptiveExecutionSpan.Finish(true)
	for !session.Finished() {
		if ctx.Err() != nil {
			session.Cancel()
			break
		}
		rem := budget.Remaining()
		if rem != nil && *rem == 0 {
			break
		}
		grant := normalEpochGrant
		_, rejectRate, _, _ := session.LiveSignals()
		usedSinceCandidate := session.WorksSinceCandidateUpdate()
		if !session.Progress().Found && usedSinceCandidate >= stagnationThreshold/2 {
			grant = 128
		}
		if !session.Progress().Found && usedSinceCandidate >= stagnationThreshold && rejectRate >= 0.35 {
			grant = 64
		}
		if r.HandoffWorkThreshold != nil && !session.Progress().Found {
			usedSinceCandidate := session.WorksSinceCandidateUpdate()
			if usedSinceCandidate < stagnationThreshold {
				untilCheck := stagnationThreshold - usedSinceCandidate
				if untilCheck < grant {
					grant = untilCheck
				}
			}
		}
		if rem != nil && *rem < grant {
			grant = *rem
		}
		if grant == 0 {
			break
		}
		epochs++
		st := time.Now()
		step := session.Step(ctx, grant)
		anchorNS += time.Since(st).Nanoseconds()
		if step.Consumed > grant || !budget.Consume(step.Consumed) {
			adaptiveExecutionSpan.Finish(true)
			return core.RouteResult{}, fmt.Errorf("anchor budget accounting failure")
		}
		traces = append(traces, core.TaskTrace{TaskID: fmt.Sprintf("anchor-main-epoch-%06d", epochs), Solver: "anchor/session", Purpose: "fast_path", Reason: step.NextAction, Budget: &grant, Found: step.Candidate != nil, WorkUsed: step.Consumed})
		p := session.Progress()
		if step.Candidate != nil && bearing.Wants(routeObserver, string(bearing.KindCandidateSubmitted)) {
			routeObserver.Observe(bearing.Event{TaskID: "anchor-main", Component: "ANCHOR", Phase: "candidate", Kind: string(bearing.KindCandidateSubmitted), WorkAfter: budget.Used(), Attributes: map[string]any{"distance": step.Candidate.Distance, "frontier_size": session.MaxFrontier()}})
		}
		if shouldReturnCandidate(r, p, budget.Used(), r.WorkBudget) {
			break
		}

		// Stagnation triggers one bounded BOLTS rescue. It is conditional and
		// uses the incumbent as an upper bound; ANCHOR state remains available.
		frontier, rejectRate, _, _ := session.LiveSignals()
		stalled := session.WorksSinceCandidateUpdate() >= stagnationThreshold && (r.HandoffWorkThreshold != nil || rejectRate >= 0.35 || frontier >= uint64(maxInt(8, g.NodeCount()/20)))
		if !r.Ablation.DisableFallback && !p.Found && stalled {
			rem = budget.Remaining()
			if rem == nil || *rem > 0 {
				availableState, transferredState := session.HandoffStateStats()
				localGrant := handoffBudget(g.NodeCount(), transferredState, rem)
				if localGrant > 0 {
					handoffSpan := bearing.StartLifecycle(t.Observer, "", "bridge-route", anchorSpan, "TRUSS", "conditional_handoff")
					control.AddAction(string(core.WorkHandoffAction))
					control.LogicalSteps++
					control.ScheduledSteps++
					if budget.Consume(1) {
						localGrant--
					}
					handoffs++
					req := core.RouteRequest{Source: r.Source, Target: r.Target, Mode: core.ModeFast, Workers: 1, Seed: r.Seed}
					anchorWorkAtHandoff := session.Result().Work.TotalActions
					st = time.Now()
					seed := session.ExportHandoffSeed()
					reason := "no_candidate_stagnation"
					rescueWeight := 1.12
					if rejectRate >= 0.50 {
						reason = "heuristic_misdirection"
						rescueWeight = 1.0
					} else if frontier >= uint64(maxInt(8, g.NodeCount()/20)) {
						reason = "frontier_explosion"
					}
					seedState := bolts.SeedState{Dist: seed.Dist, Prev: seed.Prev, HasPrev: seed.HasPrev, Settled: seed.Settled, Frontier: seed.Frontier}
					rescueSolver := bolts.SeededWeightedAStar{Weight: rescueWeight, Seed: seedState, RequireSeed: true}
					rescue := rescueSolver.Solve(ctx, g, req, core.WorkBudget{MaxWork: &localGrant}, routeObserver)
					rescueNS := time.Since(st).Nanoseconds()
					boltsNS += rescueNS
					if !budget.Consume(rescue.TotalWork()) {
						handoffSpan.Finish(true)
						adaptiveExecutionSpan.Finish(true)
						return core.RouteResult{}, fmt.Errorf("BOLTS budget accounting failure")
					}
					traces = append(traces, core.TaskTrace{TaskID: fmt.Sprintf("bolts-rescue-%06d", handoffs), Solver: rescue.SolverName, Purpose: "conditional_handoff", Reason: reason, Budget: &localGrant, Found: rescue.Found, Distance: rescue.Distance, WorkUsed: rescue.TotalWork()})
					queuedSeed := handoffTelemetryUint(rescue.Telemetry, "seed_queued_count")
					expandedSeed := handoffTelemetryUint(rescue.Telemetry, "seed_expanded_count")
					pathSeed := handoffTelemetryUint(rescue.Telemetry, "seed_path_contribution_count")
					reusedThisHandoff := expandedSeed
					if rescue.Found {
						hr := core.HandoffResult{RequestID: "rescue", Path: rescue.Path, Distance: rescue.Distance, Found: true, Work: rescue.Work}
						if err := session.ApplyHandoff(hr); err != nil {
							handoffSpan.Finish(true)
							adaptiveExecutionSpan.Finish(true)
							return core.RouteResult{}, err
						}
					}
					reused += reusedThisHandoff
					waste := anchorWorkAtHandoff
					if reusedThisHandoff < waste {
						waste -= reusedThisHandoff
					} else {
						waste = 0
					}
					record := core.HandoffRecord{Sequence: handoffs, Reason: reason, AnchorWorkAtHandoff: anchorWorkAtHandoff, BoltsWork: rescue.TotalWork(), BoltsTimeNS: rescueNS, AvailableStateUnits: availableState, TransferredStateUnits: transferredState, QueuedSeedStateUnits: queuedSeed, ExpandedSeedStateUnits: expandedSeed, PathContributingSeedStateUnits: pathSeed, ReusedStateUnits: reusedThisHandoff, PreHandoffWasteWork: waste}
					handoffMetrics.Records = append(handoffMetrics.Records, record)
					handoffMetrics.Count++
					handoffMetrics.TotalBoltsWork += rescue.TotalWork()
					boltsWork.Add(rescue.Work)
					handoffMetrics.TotalBoltsTimeNS += rescueNS
					handoffMetrics.TotalAvailableStateUnits += availableState
					handoffMetrics.TotalTransferredStateUnits += transferredState
					handoffMetrics.TotalQueuedSeedStateUnits += queuedSeed
					handoffMetrics.TotalExpandedSeedStateUnits += expandedSeed
					handoffMetrics.TotalPathContributingSeedStateUnits += pathSeed
					handoffMetrics.TotalReusedStateUnits += reusedThisHandoff
					handoffMetrics.TotalPreHandoffWasteWork += waste
					if rescue.Found {
						handoffSpan.Finish(false)
						break
					}
					handoffSpan.Finish(false)
					// Avoid repeatedly paying the same rescue cost.
					r.Ablation.DisableFallback = true
				}
			}
		}
		if step.Consumed == 0 {
			break
		}
	}
	adaptiveExecutionSpan.Finish(false)

	// Final safety handoff: if ANCHOR cannot continue without a candidate,
	// give BOLTS one continuation attempt even when the threshold boundary was missed.
	finalHandoffSpan := bearing.StartLifecycle(t.Observer, "", "bridge-route", anchorSpan, "TRUSS", "final_handoff")
	defer finalHandoffSpan.Finish(true)
	if !session.Progress().Found && !r.Ablation.DisableFallback {
		rem := budget.Remaining()
		if rem == nil || *rem > 1 {
			available, transferred := session.HandoffStateStats()
			localGrant := handoffBudget(g.NodeCount(), transferred, rem)
			if rem != nil && localGrant >= *rem {
				localGrant = *rem - 1
			}
			if localGrant > 0 && budget.Consume(1) {
				control.AddAction(string(core.WorkHandoffAction))
				control.LogicalSteps++
				control.ScheduledSteps++
				handoffs++
				seed := session.ExportHandoffSeed()
				req := core.RouteRequest{Source: r.Source, Target: r.Target, Mode: core.ModeFast, Workers: 1, Seed: r.Seed}
				st := time.Now()
				rescue := (bolts.SeededWeightedAStar{Weight: 1.0, Seed: bolts.SeedState{Dist: seed.Dist, Prev: seed.Prev, HasPrev: seed.HasPrev, Settled: seed.Settled, Frontier: seed.Frontier}, RequireSeed: true}).Solve(ctx, g, req, core.WorkBudget{MaxWork: &localGrant}, routeObserver)
				rescueNS := time.Since(st).Nanoseconds()
				boltsNS += rescueNS
				if !budget.Consume(rescue.TotalWork()) {
					finalHandoffSpan.Finish(true)
					return core.RouteResult{}, fmt.Errorf("BOLTS final handoff budget accounting failure")
				}
				boltsWork.Add(rescue.Work)
				reusedThis := handoffTelemetryUint(rescue.Telemetry, "seed_expanded_count")
				queuedSeed := handoffTelemetryUint(rescue.Telemetry, "seed_queued_count")
				pathSeed := handoffTelemetryUint(rescue.Telemetry, "seed_path_contribution_count")
				if rescue.Found {
					if err := session.ApplyHandoff(core.HandoffResult{RequestID: "final-rescue", Path: rescue.Path, Distance: rescue.Distance, Found: true, Work: rescue.Work}); err != nil {
						finalHandoffSpan.Finish(true)
						return core.RouteResult{}, err
					}
					reused += reusedThis
				}
				waste := session.Result().Work.TotalActions
				if reusedThis < waste {
					waste -= reusedThis
				} else {
					waste = 0
				}
				rec := core.HandoffRecord{Sequence: handoffs, Reason: "anchor_exhausted", AnchorWorkAtHandoff: session.Result().Work.TotalActions, BoltsWork: rescue.TotalWork(), BoltsTimeNS: rescueNS, AvailableStateUnits: available, TransferredStateUnits: transferred, QueuedSeedStateUnits: queuedSeed, ExpandedSeedStateUnits: reusedThis, PathContributingSeedStateUnits: pathSeed, ReusedStateUnits: reusedThis, PreHandoffWasteWork: waste}
				handoffMetrics.Records = append(handoffMetrics.Records, rec)
				handoffMetrics.Count++
				handoffMetrics.TotalBoltsWork += rescue.TotalWork()
				handoffMetrics.TotalBoltsTimeNS += rescueNS
				handoffMetrics.TotalAvailableStateUnits += available
				handoffMetrics.TotalTransferredStateUnits += transferred
				handoffMetrics.TotalQueuedSeedStateUnits += queuedSeed
				handoffMetrics.TotalExpandedSeedStateUnits += reusedThis
				handoffMetrics.TotalPathContributingSeedStateUnits += pathSeed
				handoffMetrics.TotalReusedStateUnits += reusedThis
				handoffMetrics.TotalPreHandoffWasteWork += waste
			}
		}
	}
	finalHandoffSpan.Finish(false)

	anchorLifecycle.Finish(false)
	finalizationSpan := bearing.StartLifecycle(t.Observer, "", "bridge-route", routeSpan, "TRUSS", "finalization")
	defer finalizationSpan.Finish(true)
	resultIntegrationSpan := bearing.StartLifecycle(t.Observer, "", "bridge-route", routeSpan, "TRUSS", "result_integration")
	defer resultIntegrationSpan.Finish(true)
	best := session.Result()
	aggregate := best.Work
	aggregate.Add(boltsWork)
	aggregate.Add(control)
	aggregate.WorkerCount = uint32(r.Workers)
	best.Work = aggregate
	best.SolverName = "bridge_" + string(r.Mode)
	best.SolverTrace = traces
	best.ParallelSteps = aggregate.ScheduledSteps
	if ctx.Err() != nil {
		best.TerminationStatus = core.TerminationDeadline
		best.DeadlineExceeded = true
		best.ErrorCode = core.ErrDeadlineExceeded
	} else if best.Found {
		best.TerminationStatus = core.TerminationFound
		if r.Mode == core.ModeExact && !r.Ablation.DisableCertification {
			certificationSpan := bearing.StartLifecycle(t.Observer, "", "bridge-route", routeSpan, "TRUSS", "certification")
			certBudget := budget.Remaining()
			if certBudget == nil || *certBudget > 0 {
				limit := uint64(g.NodeCount() * 16)
				if certBudget != nil && *certBudget < limit {
					limit = *certBudget
				}
				if limit > 0 {
					st := time.Now()
					exact := bolts.Dijkstra{}.Solve(ctx, g, r, core.WorkBudget{MaxWork: &limit}, routeObserver)
					boltsNS += time.Since(st).Nanoseconds()
					if budget.Consume(exact.TotalWork()) {
						aggregate.Add(exact.Work)
						best.Work = aggregate
						handoffs++
						if exact.Found {
							best = t.Arbiter.Choose(best, exact)
							best.SolverName = "bridge_" + string(r.Mode)
							best.Exact = exact.Exact
							best.QualityCertified = exact.QualityCertified
							best.CertifiedRatio = exact.CertifiedRatio
							best.LowerBound = exact.LowerBound
						}
					}
				}
			}
			certificationSpan.Finish(false)
		}
	} else if budget.Remaining() != nil && *budget.Remaining() == 0 {
		best.TerminationStatus = core.TerminationUnknownBudget
		best.BudgetExhausted = true
		best.ErrorCode = core.ErrBudgetExhausted
	}

	rem := budget.Remaining()
	by := map[core.Component]uint64{core.ComponentAnchor: session.Result().Work.TotalActions, core.ComponentBolts: boltsWork.TotalActions, core.ComponentTruss: control.TotalActions}
	entries := []core.BudgetLedgerEntry{{TaskID: "anchor-main", Component: core.ComponentAnchor, Purpose: "adaptive_fast_path", Used: session.Result().Work.TotalActions}}
	if handoffs > 0 {
		entries = append(entries, core.BudgetLedgerEntry{TaskID: "bolts-conditional", Component: core.ComponentBolts, Purpose: "conditional_handoff", Used: by[core.ComponentBolts]})
	}
	if control.TotalActions > 0 {
		entries = append(entries, core.BudgetLedgerEntry{TaskID: "truss-control", Component: core.ComponentTruss, Purpose: "handoff_control", Used: control.TotalActions})
	}
	best.BudgetLedger = &core.BudgetLedger{Limit: r.WorkBudget, Used: budget.Used(), Remaining: rem, ByComponent: by, Entries: entries}

	totalNS := time.Since(started).Nanoseconds()
	solverNS := anchorNS + boltsNS
	best.TimeMS = float64(totalNS) / 1e6
	best.TimeBreakdown = core.TimeBreakdown{TotalNS: totalNS, SolverNS: solverNS, TrussNS: totalNS, AnchorNS: anchorNS, BoltsNS: boltsNS, OrchestrationNS: maxInt64(0, totalNS-solverNS), TotalMS: float64(totalNS) / 1e6, SolverMS: float64(solverNS) / 1e6, TrussMS: float64(totalNS) / 1e6, AnchorMS: float64(anchorNS) / 1e6, BoltsMS: float64(boltsNS) / 1e6, OrchestrationMS: float64(maxInt64(0, totalNS-solverNS)) / 1e6}
	if best.Telemetry == nil {
		best.Telemetry = map[string]any{}
	}
	best.Telemetry["architecture"] = "TRUSS-single-adaptive-session/BOLTS-conditional-handoff"
	best.Telemetry["epochs"] = epochs
	best.Telemetry["hypothesis_count"] = 1
	best.Telemetry["handoff_count"] = handoffs
	best.Telemetry["heuristic_weight"] = weight
	best.Telemetry["candidate_update_count"] = session.CandidateUpdates()
	best.Telemetry["max_frontier_size"] = session.MaxFrontier()
	best.Telemetry["state_reuse_applied_count"] = reused
	best.Telemetry["state_reuse_ratio"] = float64(reused) / float64(maxInt(1, int(best.Work.TotalActions)))
	if bearing.Wants(routeObserver, "search_finished") {
		routeObserver.Observe(bearing.Event{TaskID: "bridge-route", Component: "TRUSS", Phase: "orchestration", Kind: "search_finished", WorkAfter: best.Work.TotalActions, Attributes: map[string]any{"found": best.Found, "distance": best.Distance}})
	}
	resultIntegrationSpan.Finish(false)
	best.Telemetry["solver_time_ns"] = solverNS
	best.Telemetry["adaptive_stagnation_threshold"] = stagnationThreshold
	if handoffMetrics.Count > 0 {
		best.HandoffMetrics = handoffMetrics
	}
	dominantWork := "ANCHOR"
	if by[core.ComponentBolts] > by[core.ComponentAnchor] {
		dominantWork = "BOLTS"
	}
	if by[core.ComponentTruss] > by[core.ComponentAnchor] && by[core.ComponentTruss] > by[core.ComponentBolts] {
		dominantWork = "TRUSS"
	}
	dominantTime := "ANCHOR"
	if boltsNS > anchorNS {
		dominantTime = "BOLTS"
	}
	if maxInt64(0, totalNS-solverNS) > anchorNS && maxInt64(0, totalNS-solverNS) > boltsNS {
		dominantTime = "ORCHESTRATION"
	}
	best.BottleneckProfile = &core.BottleneckProfile{AnchorWork: by[core.ComponentAnchor], BoltsWork: by[core.ComponentBolts], TrussWork: by[core.ComponentTruss], AnchorTimeNS: anchorNS, BoltsTimeNS: boltsNS, OrchestrationTimeNS: maxInt64(0, totalNS-solverNS), EpochCount: epochs, MaxFrontierSize: uint64(session.MaxFrontier()), CandidateUpdateCount: session.CandidateUpdates(), WorksSinceCandidateUpdate: session.WorksSinceCandidateUpdate(), DominantWorkComponent: dominantWork, DominantTimeComponent: dominantTime, ProgressSamples: session.ProgressSamples()}
	finalizationSpan.Finish(false)
	return best, nil
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
