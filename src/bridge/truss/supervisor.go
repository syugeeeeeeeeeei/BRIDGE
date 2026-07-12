package truss

import (
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

type Supervisor struct{}

func (Supervisor) Inspect(component core.Component, g core.Graph, r core.RouteResult) (core.ProgressReport, *core.EmergencyReport) {
	unique := telemetryUint(r.Telemetry, "investigated_nodes")
	edges := telemetryUint(r.Telemetry, "investigated_edges")
	duplicates := uint64(0)
	if r.Work.ExpandActions > unique {
		duplicates = r.Work.ExpandActions - unique
	}
	p := core.ProgressReport{Component: component, Phase: "finished", Work: r.Work, UniqueInvestigated: unique, DuplicateInvestigated: duplicates, InvestigatedEdges: edges, NodeRatio: ratio(unique, uint64(g.NodeCount())), EdgeRatio: ratio(edges, uint64(edgeSlots(g)))}
	if r.Found {
		p.State = core.ProgressCompleted
		p.Candidate = &r
		return p, nil
	}
	p.State = core.ProgressExhausted
	kind := core.EmergencyReachabilityUncertain
	if r.BudgetExhausted {
		kind = core.EmergencyBudgetExhaustedNoProgress
	}
	if p.EdgeRatio >= .50 {
		kind = core.EmergencyEdgeScanExplosion
	}
	if p.NodeRatio >= .50 {
		kind = core.EmergencyStallDetected
	}
	return p, &core.EmergencyReport{Component: component, Kind: kind, Recoverable: true, Evidence: map[string]any{"node_ratio": p.NodeRatio, "edge_ratio": p.EdgeRatio, "duplicate_expansions": duplicates}}
}

func (Supervisor) Recommend(e *core.EmergencyReport, remaining *uint64) core.Directive {
	if e == nil {
		return core.Directive{Kind: core.DirectiveContinue}
	}
	if remaining != nil && *remaining == 0 {
		return core.Directive{Kind: core.DirectiveTerminate, Reason: "portfolio_budget_exhausted"}
	}
	capability := "CONNECT_TO_GOAL"
	if e.Kind == core.EmergencyReachabilityUncertain || e.Kind == core.EmergencyFrontierExhausted {
		capability = "CHECK_REACHABILITY"
	}
	if e.Kind == core.EmergencyHeuristicUnreliable || e.Kind == core.EmergencyEdgeScanExplosion {
		capability = "FIND_ROUTE_TO_GOAL"
	}
	return core.Directive{Kind: core.DirectiveStartBolts, Capability: capability, Reason: string(e.Kind)}
}

func telemetryUint(m map[string]any, k string) uint64 {
	if m == nil {
		return 0
	}
	switch v := m[k].(type) {
	case uint64:
		return v
	case int:
		return uint64(v)
	case float64:
		return uint64(v)
	}
	return 0
}
func ratio(a, b uint64) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}
func edgeSlots(g core.Graph) int {
	n := 0
	for i := 0; i < g.NodeCount(); i++ {
		n += len(g.EdgesFrom(core.NodeID(i)))
	}
	return n
}
