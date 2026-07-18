package traffic

import (
	"errors"
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"time"
)

const BenchmarkSchemaV1 = "bridge.benchmark.v1"
const BenchmarkResultSchemaV1 = "bridge.benchmark.artifact.v1"

type BenchmarkScenario struct {
	SchemaVersion string          `json:"schema_version" yaml:"schema_version"`
	Suite         SuiteSpec       `json:"suite" yaml:"suite"`
	Execution     ExecutionSpec   `json:"execution" yaml:"execution"`
	Algorithms    []string        `json:"algorithms" yaml:"algorithms"`
	Observation   ObservationSpec `json:"observation" yaml:"observation"`
	Output        OutputSpec      `json:"output,omitempty" yaml:"output,omitempty"`
	Scenarios     []ScenarioCase  `json:"scenarios" yaml:"scenarios"`
}

type SuiteSpec struct {
	ID          string `json:"id" yaml:"id"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}
type ExecutionSpec struct {
	Repetitions    int     `json:"repetitions" yaml:"repetitions"`
	WarmupRuns     int     `json:"warmup_runs,omitempty" yaml:"warmup_runs,omitempty"`
	Seeds          []int64 `json:"seeds" yaml:"seeds"`
	RunTimeout     string  `json:"run_timeout,omitempty" yaml:"run_timeout,omitempty"`
	RandomizeOrder bool    `json:"randomize_order,omitempty" yaml:"randomize_order,omitempty"`
}
type ObservationSpec struct {
	Mode string `json:"mode" yaml:"mode"`
}
type OutputSpec struct {
	Directory string `json:"directory" yaml:"directory"`
}
type ScenarioCase struct {
	ID       string        `json:"id" yaml:"id"`
	Graph    GeneratorSpec `json:"graph" yaml:"graph"`
	Queries  []QuerySpec   `json:"queries,omitempty" yaml:"queries,omitempty"`
	Route    RouteSpec     `json:"route,omitempty" yaml:"route,omitempty"`
	Budget   BudgetSpec    `json:"budget,omitempty" yaml:"budget,omitempty"`
	Ablation AblationSpec  `json:"ablation,omitempty" yaml:"ablation,omitempty"`
}

type QuerySpec struct {
	ID        string             `json:"id" yaml:"id"`
	Selection QuerySelectionSpec `json:"selection" yaml:"selection"`
}
type QuerySelectionSpec struct {
	Method string  `json:"method" yaml:"method"`
	Source *uint32 `json:"source,omitempty" yaml:"source,omitempty"`
	Target *uint32 `json:"target,omitempty" yaml:"target,omitempty"`
}
type GeneratorSpec struct {
	Generator     string  `json:"generator" yaml:"generator"`
	Nodes         int     `json:"requested_node_count,omitempty" yaml:"requested_node_count,omitempty"`
	Width         int     `json:"width,omitempty" yaml:"width,omitempty"`
	Height        int     `json:"height,omitempty" yaml:"height,omitempty"`
	Topology      string  `json:"topology,omitempty" yaml:"topology,omitempty"`
	K             int     `json:"neighbor_candidate_count,omitempty" yaml:"neighbor_candidate_count,omitempty"`
	Noise         float64 `json:"edge_weight_noise,omitempty" yaml:"edge_weight_noise,omitempty"`
	Communities   int     `json:"community_count,omitempty" yaml:"community_count,omitempty"`
	DatasetPath   string  `json:"dataset_path,omitempty" yaml:"dataset_path,omitempty"`
	DatasetFormat string  `json:"dataset_format,omitempty" yaml:"dataset_format,omitempty"`
}
type RouteSpec struct {
	Mode                 core.RouteMode `json:"mode,omitempty" yaml:"mode,omitempty"`
	Workers              int            `json:"logical_worker_count,omitempty" yaml:"logical_worker_count,omitempty"`
	HandoffWorkThreshold *uint64        `json:"handoff_work_threshold,omitempty" yaml:"handoff_work_threshold,omitempty"`
}
type AblationSpec = core.AblationOptions

type BudgetSpec struct {
	WorkLimit       *uint64 `json:"work_limit,omitempty" yaml:"work_limit,omitempty"`
	SearchTimeLimit string  `json:"search_time_limit,omitempty" yaml:"search_time_limit,omitempty"`
}
type BenchmarkResult struct {
	SchemaVersion      string               `json:"schema_version"`
	TerminologyVersion string               `json:"terminology_version"`
	SuiteID            string               `json:"suite_id"`
	ExecutionID        string               `json:"execution_id"`
	OutputDirectory    string               `json:"output_directory"`
	RunMetadata        ArtifactRunMetadata  `json:"run_metadata"`
	Execution          ExecutionManifest    `json:"execution"`
	Environment        *EnvironmentMetadata `json:"environment,omitempty"`
	Runs               []BenchmarkRun       `json:"runs"`
	ScenarioSummaries  []ScenarioSummary    `json:"scenario_summaries"`
	Failures           []string             `json:"failures,omitempty"`
}

type ArtifactRunMetadata struct {
	ScenarioSchemaVersion string  `json:"scenario_schema_version"`
	StartedAt             string  `json:"started_at"`
	CompletedAt           string  `json:"completed_at,omitempty"`
	DurationMS            float64 `json:"duration_ms"`
	ExecutionSucceeded    bool    `json:"execution_succeeded"`
	ObservationMode       string  `json:"observation_mode"`
}
type ExecutionManifest struct {
	Randomized       bool     `json:"randomized"`
	ShuffleSeed      int64    `json:"shuffle_seed,omitempty"`
	ShuffleAlgorithm string   `json:"shuffle_algorithm,omitempty"`
	RunOrder         []string `json:"run_order"`
}

type ScenarioSummary struct {
	ScenarioID           string                       `json:"scenario_id"`
	Algorithm            string                       `json:"algorithm"`
	QueryID              string                       `json:"query_id"`
	TargetKind           string                       `json:"target_kind"`
	ExecutionPath        string                       `json:"execution_path"`
	Runs                 int                          `json:"runs"`
	FoundRate            float64                      `json:"path_found_rate"`
	ExactRate            float64                      `json:"optimality_proven_rate"`
	AverageDistance      float64                      `json:"mean_path_cost"`
	AverageWork          float64                      `json:"mean_work_actions"`
	AverageTimeMS        float64                      `json:"mean_time_ms"`
	AverageSolverTimeMS  float64                      `json:"mean_solver_time_ms"`
	AverageEndToEndMS    float64                      `json:"mean_end_to_end_time_ms"`
	MinDistance          *float64                     `json:"min_distance,omitempty"`
	MaxDistance          *float64                     `json:"max_distance,omitempty"`
	WorkStatistics       SummaryStatistics            `json:"work_statistics"`
	SolverTimeStatistics SummaryStatistics            `json:"solver_time_statistics"`
	EndToEndStatistics   SummaryStatistics            `json:"end_to_end_time_statistics"`
	MetricStatistics     map[string]SummaryStatistics `json:"metric_statistics"`
	FailureReasons       map[string]int               `json:"failure_reasons,omitempty"`
	Ablation             AblationSpec                 `json:"ablation,omitempty"`
}

func (s *BenchmarkScenario) ApplyDefaults() {
	if s.Output.Directory == "" {
		s.Output.Directory = "./artifacts"
	}
	if s.Execution.Repetitions == 0 {
		s.Execution.Repetitions = 1
	}
	if len(s.Execution.Seeds) == 0 {
		s.Execution.Seeds = []int64{1}
	}

	if len(s.Algorithms) == 0 {
		s.Algorithms = []string{"bridge"}
	}
	if s.Observation.Mode == "" {
		s.Observation.Mode = "minimum"
	}
	for i := range s.Scenarios {
		if s.Scenarios[i].Graph.Generator == "" {
			s.Scenarios[i].Graph.Generator = "grid"
		}
		if s.Scenarios[i].Graph.Topology == "" {
			s.Scenarios[i].Graph.Topology = "open"
		}
		if len(s.Scenarios[i].Queries) == 0 {
			s.Scenarios[i].Queries = []QuerySpec{{ID: "default", Selection: QuerySelectionSpec{Method: "generator_default"}}}
		}
		for q := range s.Scenarios[i].Queries {
			if s.Scenarios[i].Queries[q].Selection.Method == "" {
				s.Scenarios[i].Queries[q].Selection.Method = "generator_default"
			}
		}
		if s.Scenarios[i].Route.Mode == "" {
			s.Scenarios[i].Route.Mode = core.ModeBalanced
		}
		if s.Scenarios[i].Route.Workers == 0 {
			s.Scenarios[i].Route.Workers = 1
		}
	}
}

func (s BenchmarkScenario) Validate() error {
	if s.SchemaVersion != BenchmarkSchemaV1 {
		return fmt.Errorf("schema_version must be %q", BenchmarkSchemaV1)
	}
	if s.Suite.ID == "" {
		return errors.New("suite.id is required")
	}
	if s.Execution.Repetitions < 1 || s.Execution.Repetitions > 1000 {
		return errors.New("execution.repetitions must be between 1 and 1000")
	}
	if s.Execution.WarmupRuns < 0 || s.Execution.WarmupRuns > 1000 {
		return errors.New("execution.warmup_runs must be between 0 and 1000")
	}
	if len(s.Execution.Seeds) == 0 {
		return errors.New("execution.seeds must not be empty")
	}
	if s.Execution.RunTimeout != "" {
		if _, err := time.ParseDuration(s.Execution.RunTimeout); err != nil {
			return fmt.Errorf("execution.run_timeout: %w", err)
		}
	}
	switch s.Observation.Mode {
	case "minimum", "debug", "trace":
	default:
		return fmt.Errorf("observation.mode must be one of minimum, debug, trace")
	}
	if s.Output.Directory == "" {
		return errors.New("output.directory is required")
	}
	if err := validateSafeID("suite.id", s.Suite.ID); err != nil {
		return err
	}
	for _, c := range s.Scenarios {
		if c.Ablation.DisableDetour || c.Ablation.DisableBudgetReallocation || c.Ablation.DisableStateReuse {
			return fmt.Errorf("scenario %q configures an unsupported ablation; detour, budget_reallocation, and state_reuse are not independently implemented", c.ID)
		}
	}
	for _, a := range s.Algorithms {
		switch a {
		case "bridge", "anchor", "dijkstra", "bidirectional_dijkstra", "astar", "weighted_astar", "reachability":
		default:
			return fmt.Errorf("unsupported algorithm %q", a)
		}
	}
	if len(s.Scenarios) == 0 {
		return errors.New("scenarios must not be empty")
	}
	ids := map[string]bool{}
	for _, c := range s.Scenarios {
		if c.ID == "" {
			return errors.New("scenario.id is required")
		}
		if ids[c.ID] {
			return fmt.Errorf("duplicate scenario id %q", c.ID)
		}
		ids[c.ID] = true
		switch c.Graph.Generator {
		case "grid":
			if err := validateGridGraphSpec(c.ID, c.Graph); err != nil {
				return err
			}
		case "random_geometric":
			if err := validateRandomGeometricSpec(c.ID, c.Graph); err != nil {
				return err
			}
		case "community", "maze", "adversarial":
			if c.Graph.Nodes < 4 {
				return fmt.Errorf("scenario %q: graph.requested_node_count must be >= 4", c.ID)
			}
			if c.Graph.Generator == "community" && c.Graph.Communities < 0 {
				return fmt.Errorf("scenario %q: graph.community_count must be >= 0", c.ID)
			}
		case "dataset":
			if c.Graph.DatasetPath == "" {
				return fmt.Errorf("scenario %q: graph.dataset_path is required", c.ID)
			}
			if c.Graph.DatasetFormat != "" && c.Graph.DatasetFormat != "bridge.dataset.v1.json" {
				return fmt.Errorf("scenario %q: unsupported dataset_format %q", c.ID, c.Graph.DatasetFormat)
			}
		default:
			return fmt.Errorf("scenario %q: unsupported generator %q", c.ID, c.Graph.Generator)
		}
		nodeCount, err := scenarioNodeCount(c.Graph)
		if err != nil {
			return fmt.Errorf("scenario %q: %w", c.ID, err)
		}
		queries := effectiveQueries(c)
		queryIDs := map[string]bool{}
		for _, q := range queries {
			if q.ID == "" {
				return fmt.Errorf("scenario %q: query.id is required", c.ID)
			}
			if queryIDs[q.ID] {
				return fmt.Errorf("scenario %q: duplicate query id %q", c.ID, q.ID)
			}
			queryIDs[q.ID] = true
			if q.Selection.Method != "generator_default" && q.Selection.Method != "explicit" {
				return fmt.Errorf("scenario %q query %q: unsupported selection.method", c.ID, q.ID)
			}
			if q.Selection.Method == "explicit" && (q.Selection.Source == nil || q.Selection.Target == nil) {
				return fmt.Errorf("scenario %q query %q: explicit requires source and target", c.ID, q.ID)
			}
			if q.Selection.Method == "explicit" && (int(*q.Selection.Source) >= nodeCount || int(*q.Selection.Target) >= nodeCount) {
				return fmt.Errorf("scenario %q query %q: endpoint is outside graph node range 0..%d", c.ID, q.ID, nodeCount-1)
			}
		}
		switch c.Route.Mode {
		case core.ModeFast, core.ModeBalanced, core.ModeQuality, core.ModeExact:
		default:
			return fmt.Errorf("scenario %q: unsupported route mode %q", c.ID, c.Route.Mode)
		}
		if c.Route.Workers < 1 {
			return fmt.Errorf("scenario %q: route.logical_worker_count must be >= 1", c.ID)
		}
		if c.Budget.WorkLimit != nil && *c.Budget.WorkLimit == 0 {
			return fmt.Errorf("scenario %q: budget.work_limit must be > 0", c.ID)
		}
		if c.Budget.SearchTimeLimit != "" {
			d, err := time.ParseDuration(c.Budget.SearchTimeLimit)
			if err != nil || d <= 0 {
				return fmt.Errorf("scenario %q: budget.search_time_limit must be a positive duration", c.ID)
			}
			if s.Execution.RunTimeout != "" {
				rt, _ := time.ParseDuration(s.Execution.RunTimeout)
				if rt < d {
					return fmt.Errorf("scenario %q: execution.run_timeout must be >= budget.search_time_limit", c.ID)
				}
			}
		}
		if c.Graph.Noise < 0 {
			return fmt.Errorf("scenario %q: graph.edge_weight_noise must be >= 0", c.ID)
		}
	}
	return nil
}
