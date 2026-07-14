package bolts

import (
	"context"
	"fmt"
	"math"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

// LocalExecutor executes bounded BOLTS capabilities. It deliberately reuses the
// canonical BOLTS solvers so work accounting and path validity stay identical
// to normal solver execution.
type LocalExecutor struct {
	Observer bearing.Observer
}

func (e LocalExecutor) Execute(ctx context.Context, g core.Graph, req core.HandoffRequest) (core.HandoffResult, error) {
	if req.ID == "" {
		return core.HandoffResult{}, fmt.Errorf("handoff id is required")
	}
	if len(req.Inputs) == 0 {
		return core.HandoffResult{}, fmt.Errorf("handoff requires at least one checkpoint")
	}
	source := req.Inputs[0].Node
	target := source
	if len(req.Inputs) > 1 {
		target = req.Inputs[len(req.Inputs)-1].Node
	}
	if !g.HasNode(source) || !g.HasNode(target) {
		return core.HandoffResult{}, fmt.Errorf("checkpoint outside graph")
	}
	o := e.Observer
	if o == nil {
		o = bearing.NullObserver{}
	}
	rr := core.RouteRequest{Source: source, Target: target, Workers: 1}
	var res core.RouteResult
	switch req.Purpose {
	case core.ConnectCheckpoints, core.RepairSegment, core.CertifyCandidate, core.TightenBound:
		res = Dijkstra{}.Solve(ctx, g, rr, req.Budget, o)
	case core.EscapeRegion:
		escape := firstEscapeTarget(g, req.Region, source)
		if escape == source {
			return core.HandoffResult{RequestID: req.ID, Found: false}, nil
		}
		rr.Target = escape
		res = Dijkstra{}.Solve(ctx, g, rr, req.Budget, o)
	case core.ProveUnreachable:
		res = Reachability{}.Solve(ctx, g, rr, req.Budget, o)
	default:
		return core.HandoffResult{}, fmt.Errorf("unsupported handoff purpose %s", req.Purpose)
	}
	out := core.HandoffResult{RequestID: req.ID, Path: append([]core.NodeID(nil), res.Path...), Distance: res.Distance, Found: res.Found, Work: res.Work}
	if res.Found && len(res.Path) > 0 {
		cp := core.Checkpoint{Node: res.Path[len(res.Path)-1], Cost: res.Distance, HypothesisID: req.HypothesisID}
		out.ResumeCheckpoints = []core.Checkpoint{cp}
	}
	switch req.Purpose {
	case core.ProveUnreachable:
		if res.ReachabilityProven && res.SearchCompleted && !res.Found {
			out.Evidence = append(out.Evidence, core.Evidence{ID: req.ID + ":unreachable", Solver: "bolts/reachability", HypothesisID: req.HypothesisID, Scope: req.Region, GeneratedWork: res.TotalWork(), Proof: core.ProofUnreachable})
		}
	case core.CertifyCandidate:
		if res.Exact && res.Found {
			out.Evidence = append(out.Evidence, core.Evidence{ID: req.ID + ":exact", Solver: "bolts/dijkstra", HypothesisID: req.HypothesisID, Scope: req.Region, GeneratedWork: res.TotalWork(), Proof: core.ProofExact, Value: res.Distance})
		}
	case core.TightenBound:
		if res.Exact && res.Found && !math.IsInf(res.Distance, 1) {
			out.Evidence = append(out.Evidence, core.Evidence{ID: req.ID + ":lower-bound", Solver: "bolts/dijkstra", HypothesisID: req.HypothesisID, Scope: req.Region, GeneratedWork: res.TotalWork(), Proof: core.ProofAdmissibleLowerBound, Value: res.Distance})
		}
	}
	return out, nil
}

func firstEscapeTarget(g core.Graph, region core.Region, source core.NodeID) core.NodeID {
	for _, n := range region.Nodes {
		for _, edge := range g.EdgesFrom(n) {
			if !region.Contains(edge.To) {
				return edge.To
			}
		}
	}
	return source
}
