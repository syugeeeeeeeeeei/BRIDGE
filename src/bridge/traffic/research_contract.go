package traffic

import (
	"math"
	"math/rand"
	"runtime"
	"sort"
	"time"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/ultrasound"
)

const TerminologyVersionV1 = "bridge.terminology.v1"

// EnvironmentMetadata records the execution environment required to interpret a benchmark artifact.
type EnvironmentMetadata struct {
	GoVersion  string `json:"go_version"`
	GOOS       string `json:"goos"`
	GOARCH     string `json:"goarch"`
	CPUs       int    `json:"cpus"`
	CapturedAt string `json:"captured_at"`
}

type BenchmarkRunMetadata struct {
	RunOrdinal          int     `json:"run_ordinal"`
	RunID               string  `json:"run_id"`
	RunName             string  `json:"run_name"`
	ScenarioID          string  `json:"scenario_id"`
	GraphInstanceID     string  `json:"graph_instance_id"`
	QueryID             string  `json:"query_id"`
	AlgorithmID         string  `json:"algorithm_id"`
	ExecutionSeed       int64   `json:"execution_seed"`
	RepetitionIndex     int     `json:"repetition_index"`
	WarmupRun           bool    `json:"warmup_run"`
	ExecutionStartedAt  string  `json:"execution_started_at,omitempty"`
	ExecutionSucceeded  bool    `json:"execution_succeeded"`
	StableDigest        string  `json:"stable_digest,omitempty"`
	ObservationOverhead float64 `json:"observation_overhead_ratio,omitempty"`
}

type ScenarioDefinition struct {
	ScenarioID             string            `json:"scenario_id"`
	GraphGenerator         string            `json:"graph_generator"`
	GraphGeneratorSettings GeneratorSpec     `json:"graph_generator_settings"`
	RouteConfiguration     RouteSpec         `json:"route_configuration"`
	BudgetConfiguration    BudgetSpec        `json:"budget_configuration,omitempty"`
	AblationConfiguration  AblationSpec      `json:"ablation_configuration,omitempty"`
	QuerySelectionMethod   string            `json:"query_selection_method"`
	ObservationMode        string            `json:"observation_mode"`
	ObservationSampleRate  float64           `json:"observation_sample_rate,omitempty"`
	OutputMetadata         map[string]string `json:"output_metadata,omitempty"`
}

// GraphProfile describes the generated graph instance without exposing solver-private state.
type GraphProfile struct {
	GraphInstanceID  string           `json:"graph_instance_id"`
	Generator        string           `json:"graph_generator"`
	Topology         string           `json:"graph_topology,omitempty"`
	GraphSeed        int64            `json:"graph_seed"`
	ActualNodeCount  int              `json:"actual_node_count"`
	EdgeCount        int              `json:"edge_count"`
	Directed         bool             `json:"directed"`
	AverageOutDegree float64          `json:"average_out_degree,omitempty"`
	EdgeDensity      float64          `json:"edge_density,omitempty"`
	Dataset          *DatasetMetadata `json:"dataset,omitempty"`
}

// QueryProfile identifies one source-target query within a graph instance.
type QueryProfile struct {
	QueryID              string      `json:"query_id"`
	QuerySeed            int64       `json:"query_seed"`
	QueryHash            string      `json:"query_hash"`
	QuerySelectionMethod string      `json:"query_selection_method"`
	Source               core.NodeID `json:"source"`
	Target               core.NodeID `json:"target"`
}

type AlgorithmConfiguration struct {
	AlgorithmID           string       `json:"algorithm_id"`
	ExecutionPath         string       `json:"execution_path"`
	TargetKind            string       `json:"target_kind"`
	RouteConfiguration    RouteSpec    `json:"route_configuration"`
	BudgetConfiguration   BudgetSpec   `json:"budget_configuration,omitempty"`
	AblationConfiguration AblationSpec `json:"ablation_configuration,omitempty"`
}

// ExecutionResult records result facts and algorithm claims without acceptance judgment.
type ExecutionResult struct {
	Path                []core.NodeID           `json:"returned_path,omitempty"`
	PathFound           bool                    `json:"path_found"`
	SearchCompleted     bool                    `json:"search_completed"`
	ReachabilityProven  bool                    `json:"reachability_proven"`
	OptimalityProven    bool                    `json:"optimality_proven"`
	PathCost            *float64                `json:"path_cost,omitempty"`
	ErrorCode           core.ErrorCode          `json:"error_code,omitempty"`
	FailureReason       string                  `json:"failure_reason,omitempty"`
	TerminationReason   string                  `json:"termination_reason,omitempty"`
	TimeToFirstPathMS   *float64                `json:"first_path_elapsed_ms,omitempty"`
	TimeToBestFoundMS   *float64                `json:"best_path_elapsed_ms,omitempty"`
	ImprovementCount    uint64                  `json:"improvement_count"`
	BridgeOverheadRatio float64                 `json:"bridge_overhead_ratio,omitempty"`
	DuplicatedWorkRatio float64                 `json:"duplicated_work_ratio,omitempty"`
	StateReuseRatio     float64                 `json:"state_reuse_ratio,omitempty"`
	BudgetLedger        *core.BudgetLedger      `json:"budget_ledger,omitempty"`
	HandoffMetrics      *core.HandoffMetrics    `json:"handoff_metrics,omitempty"`
	BottleneckProfile   *core.BottleneckProfile `json:"bottleneck_profile,omitempty"`
	QualityClaims       QualityClaims           `json:"quality_claims"`
}

type QualityClaims struct {
	LowerBound         *float64 `json:"lower_bound,omitempty"`
	ProvenCostRatio    *float64 `json:"proven_cost_ratio,omitempty"`
	QualityBoundProven bool     `json:"quality_bound_proven"`
}

type Measurement struct {
	Work           core.WorkMetrics   `json:"work"`
	TimeBreakdown  core.TimeBreakdown `json:"time_breakdown"`
	SystemMetrics  core.SystemMetrics `json:"system_metrics"`
	SolverTimeNS   int64              `json:"solver_time_ns"`
	EndToEndTimeNS int64              `json:"end_to_end_time_ns"`
	SolverTimeMS   float64            `json:"solver_time_ms"`
	EndToEndTimeMS float64            `json:"end_to_end_time_ms"`
	ZeroDuration   bool               `json:"zero_duration"`
	TimingValid    bool               `json:"timing_valid"`
	TimingIssue    string             `json:"timing_issue,omitempty"`
}

type DebugSummary struct {
	ActionCounts           map[string]uint64       `json:"action_counts"`
	WorkByComponent        map[string]uint64       `json:"work_by_component,omitempty"`
	BudgetGrantedByPurpose map[string]uint64       `json:"budget_granted_by_purpose,omitempty"`
	BudgetUsedByPurpose    map[string]uint64       `json:"budget_used_by_purpose,omitempty"`
	CandidateUpdateCount   uint64                  `json:"candidate_update_count"`
	FallbackCount          uint64                  `json:"fallback_count"`
	CertificationCount     uint64                  `json:"certification_count"`
	StateReuseAppliedCount uint64                  `json:"state_reuse_applied_count"`
	MaxFrontierSize        uint64                  `json:"max_frontier_size"`
	ComponentEventCounts   map[string]uint64       `json:"component_event_counts,omitempty"`
	ObservationOverheadNS  int64                   `json:"observation_overhead_ns"`
	TraceSinkWriteNS       int64                   `json:"trace_sink_write_ns"`
	DroppedEvents          uint64                  `json:"dropped_events"`
	Truncated              bool                    `json:"truncated"`
	HandoffMetrics         *core.HandoffMetrics    `json:"handoff_metrics,omitempty"`
	BottleneckProfile      *core.BottleneckProfile `json:"bottleneck_profile,omitempty"`
}

type Observations struct {
	ObservationData  any                          `json:"observation_data,omitempty"`
	QualityHistory   []ultrasound.QualityPoint    `json:"quality_history,omitempty"`
	BudgetHistory    []ultrasound.BudgetPoint     `json:"budget_history,omitempty"`
	CollectorMetrics *ultrasound.CollectorMetrics `json:"collector_metrics,omitempty"`
	DebugSummary     *DebugSummary                `json:"debug_summary,omitempty"`
}

type References struct {
	GraphSpecification  GeneratorSpec `json:"graph_specification"`
	GraphSnapshotPath   string        `json:"graph_snapshot_path,omitempty"`
	GraphSnapshotSHA256 string        `json:"graph_snapshot_sha256,omitempty"`
	TraceManifestPath   string        `json:"trace_manifest_path,omitempty"`
	TracePath           string        `json:"trace_path,omitempty"`
}

type BenchmarkRun struct {
	RunMetadata            BenchmarkRunMetadata   `json:"run_metadata"`
	ScenarioDefinition     ScenarioDefinition     `json:"scenario_definition"`
	GraphProfile           GraphProfile           `json:"graph_profile"`
	QueryProfile           QueryProfile           `json:"query_profile"`
	AlgorithmConfiguration AlgorithmConfiguration `json:"algorithm_configuration"`
	ExecutionResult        ExecutionResult        `json:"execution_result"`
	Measurement            Measurement            `json:"measurement"`
	Observations           Observations           `json:"observations"`
	References             References             `json:"references"`
}

// SummaryStatistics is recomputable from raw scalar observations.
type SummaryStatistics struct {
	Count     int     `json:"count"`
	Mean      float64 `json:"mean"`
	StdDev    float64 `json:"stddev"`
	Min       float64 `json:"min"`
	P50       float64 `json:"p50"`
	P95       float64 `json:"p95"`
	Max       float64 `json:"max"`
	CI95Lower float64 `json:"ci95_lower"`
	CI95Upper float64 `json:"ci95_upper"`
}

func summarizeValues(values []float64) SummaryStatistics {
	if len(values) == 0 {
		return SummaryStatistics{}
	}
	x := append([]float64(nil), values...)
	sort.Float64s(x)
	sum := 0.0
	for _, v := range x {
		sum += v
	}
	mean := sum / float64(len(x))
	variance := 0.0
	if len(x) > 1 {
		for _, v := range x {
			d := v - mean
			variance += d * d
		}
		variance /= float64(len(x) - 1)
	}
	sd := math.Sqrt(variance)
	margin := 0.0
	if len(x) > 1 {
		margin = 1.96 * sd / math.Sqrt(float64(len(x)))
	}
	return SummaryStatistics{
		Count: len(x), Mean: mean, StdDev: sd, Min: x[0],
		P50: percentile(x, 0.50), P95: percentile(x, 0.95), Max: x[len(x)-1],
		CI95Lower: mean - margin, CI95Upper: mean + margin,
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	pos := p * float64(len(sorted)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return sorted[lo]
	}
	f := pos - float64(lo)
	return sorted[lo]*(1-f) + sorted[hi]*f
}

func captureEnvironment() *EnvironmentMetadata {
	return &EnvironmentMetadata{GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, CPUs: runtime.NumCPU(), CapturedAt: time.Now().UTC().Format(time.RFC3339Nano)}
}

func effectiveQueries(c ScenarioCase) []QuerySpec {
	if len(c.Queries) > 0 {
		return c.Queries
	}
	return []QuerySpec{{ID: "default", Selection: QuerySelectionSpec{Method: "generator_default"}}}
}

type runPlan struct {
	Scenario   ScenarioCase
	Algorithm  string
	Seed       int64
	Repetition int
	Warmup     bool
	Query      QuerySpec
}

func expandRunPlans(s BenchmarkScenario) []runPlan {
	plans := make([]runPlan, 0)
	for _, c := range s.Scenarios {
		for _, a := range s.Algorithms {
			for _, seed := range s.Execution.Seeds {
				for _, q := range effectiveQueries(c) {
					for w := 1; w <= s.Execution.WarmupRuns; w++ {
						plans = append(plans, runPlan{c, a, seed, w, true, q})
					}
					for rep := 1; rep <= s.Execution.Repetitions; rep++ {
						plans = append(plans, runPlan{c, a, seed, rep, false, q})
					}
				}
			}
		}
	}
	if s.Execution.RandomizeOrder {
		r := rand.New(rand.NewSource(stablePlanSeed(s.Execution.Seeds)))
		r.Shuffle(len(plans), func(i, j int) { plans[i], plans[j] = plans[j], plans[i] })
	}
	return plans
}

func stablePlanSeed(seeds []int64) int64 {
	var out int64 = 1469598103934665603
	for _, v := range seeds {
		out ^= v
		out *= 1099511628211
	}
	return out
}
