package gate

import (
	"context"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/truss"
)

type Gate struct{ truss *truss.Truss }

func New(observer bearing.Observer) *Gate { return &Gate{truss: truss.New(observer)} }
func (g *Gate) Route(ctx context.Context, graph core.Graph, request core.RouteRequest) (core.RouteResult, error) {
	return g.truss.Route(ctx, graph, request)
}
func (g *Gate) ExecuteOnce(ctx context.Context, graph core.Graph, request ExecuteOnceRequest) (ExecuteOnceResult, error) {
	result, err := g.truss.ExecuteOnce(ctx, graph, truss.ExecuteOnceRequest{
		TargetID:   request.TargetID,
		Source:     request.Source,
		Target:     request.Target,
		Mode:       request.Mode,
		DeadlineMS: request.DeadlineMS,
		WorkBudget: request.WorkBudget,
		Workers:    request.Workers,
		Seed:       request.Seed,
	})
	if err != nil {
		return ExecuteOnceResult{}, err
	}
	return ExecuteOnceResult{
		Result:        result.Result,
		TargetID:      result.TargetID,
		TargetKind:    result.TargetKind,
		ExecutionPath: result.ExecutionPath,
		SolverTimeMS:  result.SolverTimeMS,
		EndToEndMS:    result.EndToEndMS,
	}, nil
}
