package healthy

import (
	"context"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
	"math"
)

func validateExact(ctx context.Context, g core.Graph, run traffic.BenchmarkRun, p HealthProfile) ExactValidation {
	in := graphInput(g)
	res, err := gate.NewRouter().ExecuteOnce(ctx, gate.ExecuteRequest{SchemaVersion: gate.ExecuteRequestSchemaV1, Target: gate.ExecuteTargetInput{ID: p.ExactReferenceAlgorithm}, Graph: in, Route: gate.RouteInput{Source: uint32(run.QueryProfile.Source), Target: uint32(run.QueryProfile.Target), Mode: core.ModeExact, Workers: 1}, Observation: gate.ObservationInput{Mode: gate.ObservationMinimum}}, gate.RouteOptions{})
	if err != nil {
		return ExactValidation{Verifiable: false, ExactClaimValid: !run.ExecutionResult.OptimalityProven}
	}
	x := ExactValidation{Verifiable: true, ReferenceFound: res.Found, FalsePositive: run.ExecutionResult.PathFound && !res.Found, FalseNegative: !run.ExecutionResult.PathFound && res.Found, ExactClaimValid: true}
	if res.Distance != nil {
		d := *res.Distance
		x.ReferenceDistance = &d
	}
	if run.ExecutionResult.PathFound && res.Found && run.ExecutionResult.PathCost != nil && res.Distance != nil {
		if *res.Distance == 0 {
			r := 1.0
			if *run.ExecutionResult.PathCost != 0 {
				r = math.Inf(1)
			}
			x.DistanceRatio = &r
		} else {
			r := *run.ExecutionResult.PathCost / *res.Distance
			x.DistanceRatio = &r
		}
		if run.ExecutionResult.OptimalityProven && !closeFloat(*run.ExecutionResult.PathCost, *res.Distance, p.Validation.DistanceAbsoluteTolerance, p.Validation.DistanceRelativeTolerance) {
			x.ExactClaimValid = false
		}
	} else if run.ExecutionResult.OptimalityProven && run.ExecutionResult.PathFound != res.Found {
		x.ExactClaimValid = false
	}
	return x
}
func graphInput(g core.Graph) gate.GraphInput {
	in := gate.GraphInput{Type: "inline", Directed: g.Directed(), Nodes: make([]gate.GraphNode, g.NodeCount())}
	for i := 0; i < g.NodeCount(); i++ {
		in.Nodes[i] = gate.GraphNode{ID: uint32(i)}
		for _, e := range g.EdgesFrom(core.NodeID(i)) {
			if !g.Directed() && uint32(i) > uint32(e.To) {
				continue
			}
			in.Edges = append(in.Edges, gate.GraphEdge{From: uint32(i), To: uint32(e.To), Weight: e.Weight})
		}
	}
	return in
}
