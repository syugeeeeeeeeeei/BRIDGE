package gate

import (
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/truss"
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
	Ablation   core.AblationOptions
}

type ExecuteOnceResult struct {
	Result        core.RouteResult
	TargetID      string
	TargetKind    truss.TargetKind
	ExecutionPath string
	SolverTimeMS  float64
	EndToEndMS    float64
}
