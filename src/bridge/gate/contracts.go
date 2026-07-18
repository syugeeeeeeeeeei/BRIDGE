package gate

import (
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

const (
	RouteRequestSchemaV1   = "bridge.route.request.v1"
	RouteResultSchemaV1    = "bridge.route.result.v1"
	ExecuteRequestSchemaV1 = "bridge.execute_once.request.v1"
	ExecuteResultSchemaV1  = "bridge.execute_once.result.v1"
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
	Source               uint32         `json:"source"`
	Target               uint32         `json:"target"`
	Mode                 core.RouteMode `json:"route_mode,omitempty"`
	MaxSuboptimality     *float64       `json:"max_suboptimality,omitempty"`
	Workers              int            `json:"logical_worker_count,omitempty"`
	Seed                 uint64         `json:"seed,omitempty"`
	HandoffWorkThreshold *uint64        `json:"handoff_work_threshold,omitempty"`
}

type BudgetInput struct {
	TotalWork *uint64  `json:"total_work,omitempty"`
	TimeoutMS *float64 `json:"timeout_ms,omitempty"`
}

type ObservationMode string

const (
	ObservationMinimum ObservationMode = "minimum"
	ObservationDebug   ObservationMode = "debug"
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
	Spans         any             `json:"spans,omitempty"`
}

type RouteResult struct {
	SchemaVersion       string                  `json:"schema_version"`
	RequestID           string                  `json:"request_id,omitempty"`
	Status              string                  `json:"status"`
	Found               bool                    `json:"path_found"`
	SearchCompleted     bool                    `json:"search_completed"`
	ReachabilityProven  bool                    `json:"reachability_proven"`
	Distance            *float64                `json:"path_cost,omitempty"`
	Path                []uint32                `json:"path"`
	Exact               bool                    `json:"optimality_proven"`
	SolverName          string                  `json:"solver_name,omitempty"`
	Work                core.WorkMetrics        `json:"work"`
	SolverTimeMS        float64                 `json:"solver_time_ms,omitempty"`
	TimeBreakdown       core.TimeBreakdown      `json:"time_breakdown"`
	TimeMS              float64                 `json:"end_to_end_time_ms"`
	ErrorCode           core.ErrorCode          `json:"error_code,omitempty"`
	Observation         *ObservationResult      `json:"observation_data,omitempty"`
	FailureReason       string                  `json:"failure_reason,omitempty"`
	TimeToFirstPathMS   *float64                `json:"first_path_elapsed_ms,omitempty"`
	TimeToBestFoundMS   *float64                `json:"best_path_elapsed_ms,omitempty"`
	ImprovementCount    uint64                  `json:"improvement_count"`
	BridgeOverheadRatio float64                 `json:"bridge_overhead_ratio,omitempty"`
	DuplicatedWorkRatio float64                 `json:"duplicated_work_ratio,omitempty"`
	StateReuseRatio     float64                 `json:"state_reuse_ratio,omitempty"`
	BudgetLedger        *core.BudgetLedger      `json:"budget_ledger,omitempty"`
	HandoffMetrics      *core.HandoffMetrics    `json:"handoff_metrics,omitempty"`
	BottleneckProfile   *core.BottleneckProfile `json:"bottleneck_profile,omitempty"`
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
