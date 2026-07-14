package traffic

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/ultrasound"
)

const BenchmarkSchemaV1 = "bridge.benchmark.v3"
const BenchmarkResultSchemaV1 = "bridge.benchmark.artifact.v3"

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

func BuildScenarioGraph(spec GeneratorSpec, seed int64) (*core.AdjacencyGraph, error) {
	g, _, _, err := BuildScenarioGraphAndEndpoints(spec, seed)
	return g, err
}

func BuildScenarioGraphAndEndpoints(spec GeneratorSpec, seed int64) (*core.AdjacencyGraph, core.NodeID, core.NodeID, error) {
	switch spec.Generator {
	case "", "grid":
		if spec.Topology == "" || spec.Topology == string(TopologyOpen) {
			if spec.Nodes > 0 {
				g, err := GridNodes(spec.Nodes, seed)
				if err != nil {
					return nil, 0, 0, err
				}
				return g, 0, core.NodeID(g.NodeCount() - 1), nil
			}
			g, err := Grid(spec.Width, spec.Height, seed)
			if err != nil {
				return nil, 0, 0, err
			}
			return g, 0, core.NodeID(g.NodeCount() - 1), nil
		}
		noise := spec.Noise
		if noise == 0 {
			noise = 0.05
		}
		return TopologyGrid(spec.Nodes, GridTopology(spec.Topology), seed, noise)
	case "random_geometric":
		k := spec.K
		if k == 0 {
			k = 8
		}
		return RandomGeometric(spec.Nodes, k, seed)
	case "community":
		return CommunityGraph(spec.Nodes, spec.Communities, seed)
	case "maze":
		return MazeGraph(spec.Nodes, seed)
	case "adversarial":
		return AdversarialGraph(spec.Nodes, seed)
	case "dataset":
		d, err := LoadDataset(spec.DatasetPath)
		if err != nil {
			return nil, 0, 0, err
		}
		return d.Graph, d.Source, d.Target, nil
	default:
		return nil, 0, 0, fmt.Errorf("unsupported generator %q", spec.Generator)
	}
}

func validateGridGraphSpec(id string, spec GeneratorSpec) error {
	switch spec.Topology {
	case "", "open":
		hasNodes := spec.Nodes > 0
		hasDimensions := spec.Width > 0 || spec.Height > 0
		if hasNodes && hasDimensions {
			return fmt.Errorf("scenario %q: use graph.requested_node_count or width/height, not both", id)
		}
		if !hasNodes && (spec.Width < 1 || spec.Height < 1) {
			return fmt.Errorf("scenario %q: graph.requested_node_count must be >= 2 or both width and height provided", id)
		}
		if hasNodes && spec.Nodes < 2 {
			return fmt.Errorf("scenario %q: graph.requested_node_count must be >= 2", id)
		}
	default:
		switch spec.Topology {
		case "wall", "u_shape", "culdesac", "disconnected":
		default:
			return fmt.Errorf("scenario %q: unsupported topology %q", id, spec.Topology)
		}
		if spec.Nodes < 2 {
			return fmt.Errorf("scenario %q: graph.requested_node_count must be >= 2", id)
		}
		if spec.Width > 0 || spec.Height > 0 {
			return fmt.Errorf("scenario %q: non-open grid topology requires graph.requested_node_count and does not support width/height", id)
		}
	}
	return nil
}

func validateRandomGeometricSpec(id string, spec GeneratorSpec) error {
	if spec.Nodes < 2 {
		return fmt.Errorf("scenario %q: graph.requested_node_count must be >= 2", id)
	}
	if spec.Width > 0 || spec.Height > 0 {
		return fmt.Errorf("scenario %q: random_geometric does not support width/height", id)
	}
	if spec.Topology != "" && spec.Topology != "open" {
		return fmt.Errorf("scenario %q: random_geometric does not support topology %q", id, spec.Topology)
	}
	if spec.K < 0 {
		return fmt.Errorf("scenario %q: graph.neighbor_candidate_count must be >= 0", id)
	}
	return nil
}

func scenarioNodeCount(spec GeneratorSpec) (int, error) {
	switch spec.Generator {
	case "", "grid":
		if spec.Nodes > 0 {
			if spec.Topology == "" || spec.Topology == string(TopologyOpen) {
				return spec.Nodes, nil
			}
			side := int(math.Sqrt(float64(spec.Nodes)))
			if side < 10 {
				side = 10
			}
			return side * side, nil
		}
		if spec.Width > 0 && spec.Height > 0 {
			return spec.Width * spec.Height, nil
		}
	case "random_geometric", "community", "maze", "adversarial":
		return spec.Nodes, nil
	case "dataset":
		d, err := LoadDataset(spec.DatasetPath)
		if err != nil {
			return 0, err
		}
		return d.Graph.NodeCount(), nil
	}
	return 0, fmt.Errorf("could not determine graph node count")
}

type RunScenarioOptions struct {
	ScenarioPath     string
	ProgressReporter ProgressReporter
}

type RunProgress struct {
	RunName      string
	Current      int
	Completed    int
	Total        int
	RunElapsed   time.Duration
	TotalElapsed time.Duration
	ETA          time.Duration
	Finished     bool
}

type ProgressReporter interface {
	ReportRunProgress(RunProgress)
}

type ProgressReporterFunc func(RunProgress)

func (f ProgressReporterFunc) ReportRunProgress(progress RunProgress) {
	f(progress)
}

type WriterProgressReporter struct {
	Writer io.Writer
}

func (r WriterProgressReporter) ReportRunProgress(progress RunProgress) {
	if r.Writer == nil {
		return
	}
	_, _ = fmt.Fprintf(
		r.Writer,
		"[%d/%d] run=%s run_time=%s elapsed=%s eta=%s\n",
		progress.Current,
		progress.Total,
		progress.RunName,
		formatProgressDuration(progress.RunElapsed),
		formatProgressDuration(progress.TotalElapsed),
		formatProgressDuration(progress.ETA),
	)
}

func RunScenario(ctx context.Context, s BenchmarkScenario) (BenchmarkResult, error) {
	return RunScenarioWithOptions(ctx, s, RunScenarioOptions{})
}

func RunScenarioWithOptions(ctx context.Context, s BenchmarkScenario, opts RunScenarioOptions) (BenchmarkResult, error) {
	s.ApplyDefaults()
	if err := s.Validate(); err != nil {
		return BenchmarkResult{}, err
	}
	started := time.Now()
	out := BenchmarkResult{
		SchemaVersion:      BenchmarkResultSchemaV1,
		TerminologyVersion: TerminologyVersionV1,
		SuiteID:            s.Suite.ID,
		RunMetadata: ArtifactRunMetadata{
			ScenarioSchemaVersion: s.SchemaVersion,
			StartedAt:             started.UTC().Format(time.RFC3339Nano),
			ExecutionSucceeded:    true,
			ObservationMode:       s.Observation.Mode,
		},
	}
	out.Execution = ExecutionManifest{Randomized: s.Execution.RandomizeOrder, RunOrder: []string{}}
	if s.Execution.RandomizeOrder {
		out.Execution.ShuffleSeed = s.Execution.Seeds[0]
		out.Execution.ShuffleAlgorithm = "math/rand-v1"
	}
	out.Environment = captureEnvironment()
	executionID, err := newExecutionID()
	if err != nil {
		return BenchmarkResult{}, err
	}
	baseDirectory := s.Output.Directory
	if !filepath.IsAbs(baseDirectory) && opts.ScenarioPath != "" {
		baseDirectory = filepath.Join(filepath.Dir(opts.ScenarioPath), baseDirectory)
	}
	executionDirectory := filepath.Join(baseDirectory, s.Suite.ID, executionID)
	if err := os.MkdirAll(executionDirectory, 0o755); err != nil {
		return BenchmarkResult{}, err
	}
	out.ExecutionID = executionID
	out.OutputDirectory = executionDirectory
	plans := expandRunPlans(s)
	completedRuns := 0
	var timeout time.Duration
	if s.Execution.RunTimeout != "" {
		timeout, _ = time.ParseDuration(s.Execution.RunTimeout)
	}
	type accumulator struct {
		result                                  ScenarioSummary
		found, exact                            int
		distances, works, solverTimes, endTimes []float64
		metrics                                 map[string][]float64
		failureReasons                          map[string]int
	}
	groups := map[string]*accumulator{}
	for _, plan := range plans {
		c, algorithm, seed, rep, query := plan.Scenario, plan.Algorithm, plan.Seed, plan.Repetition, plan.Query
		graphSeed := seed
		graphInstanceID := fmt.Sprintf("%s/seed-%d", c.ID, graphSeed)
		runID := fmt.Sprintf("%s/%s/%s/%s/rep-%d", c.ID, algorithm, graphInstanceID, query.ID, rep)
		if plan.Warmup {
			runID += "/warmup"
		}
		runName := runID
		runDirName := fmt.Sprintf("%s__graph-seed-%d__query-%s__rep-%d", algorithm, graphSeed, query.ID, rep)
		if query.ID == "default" && !plan.Warmup {
			runDirName = fmt.Sprintf("%s__seed-%d__rep-%d", algorithm, seed, rep)
		}
		if plan.Warmup {
			runDirName += "__warmup"
		}
		runStarted := time.Now()
		stopProgress := startRunProgressReporter(started, runStarted, runName, completedRuns+1, len(plans), opts.ProgressReporter)
		g, defaultSource, defaultTarget, err := BuildScenarioGraphAndEndpoints(c.Graph, graphSeed)
		if err != nil {
			stopProgress(false, completedRuns)
			return out, err
		}
		var source, target uint32
		if query.Selection.Method == "explicit" {
			source, target = *query.Selection.Source, *query.Selection.Target
		} else {
			source, target = uint32(defaultSource), uint32(defaultTarget)
		}
		requestID := fmt.Sprintf("%s-%s-%d-%s-%d", c.ID, algorithm, seed, query.ID, rep)
		effectiveObservationMode := s.Observation.Mode
		// Warmups exist only to stabilize code/data paths. Recording traces or
		// collector state during warmup adds measurement overhead, writes data
		// that is discarded, and can dominate large scenarios.
		if plan.Warmup {
			effectiveObservationMode = "minimum"
		}
		req := gate.RouteRequest{SchemaVersion: gate.RouteRequestSchemaV1, RequestID: requestID, Graph: graphToInput(g), Route: gate.RouteInput{Source: source, Target: target, Mode: c.Route.Mode, Workers: c.Route.Workers, Seed: uint64(seed), HandoffWorkThreshold: c.Route.HandoffWorkThreshold}, Budget: gate.BudgetInput{TotalWork: c.Budget.WorkLimit, TimeoutMS: durationMillisecondsPointer(c.Budget.SearchTimeLimit)}, Observation: gate.ObservationInput{Mode: gate.ObservationMode(effectiveObservationMode)}, Ablation: c.Ablation}
		runCtx := ctx
		cancel := func() {}
		if timeout > 0 {
			runCtx, cancel = context.WithTimeout(ctx, timeout)
		}
		traceBaseDir := ""
		if effectiveObservationMode == "trace" {
			traceBaseDir = filepath.Join(executionDirectory, "traces", runDirName)
		}
		var collector *ultrasound.Collector
		var tracePath string
		if effectiveObservationMode != "off" {
			var sink ultrasound.EventSink = ultrasound.DiscardSink{}
			if traceBaseDir != "" && effectiveObservationMode == "trace" {
				tracePath = filepath.Join(traceBaseDir, "trace.jsonl")
				fs, sinkErr := ultrasound.NewFileSink(tracePath, false)
				if sinkErr != nil {
					cancel()
					stopProgress(false, completedRuns)
					return out, sinkErr
				}
				sink = fs
			}
			collector = ultrasound.NewCollector(effectiveObservationMode, sink)
		}
		obs := gate.ObservationOptions{Mode: gate.ObservationMode(effectiveObservationMode)}
		if collector != nil {
			obs.Observer = collector
			obs.Reporter = collector
		}
		router := gate.NewRouter()
		var routeResult gate.RouteResult
		var executeResult gate.ExecuteResult
		var memBefore, memAfter runtime.MemStats
		runtime.ReadMemStats(&memBefore)
		stopHeapSampling := startHeapSampler(false, memBefore.HeapAlloc)
		apiStarted := time.Now()
		if algorithm == "bridge" {
			routeResult, err = router.Route(runCtx, req, gate.RouteOptions{Observation: obs})
		} else {
			executeResult, err = router.ExecuteOnce(runCtx, gate.ExecuteRequest{SchemaVersion: gate.ExecuteRequestSchemaV1, RequestID: requestID, Target: gate.ExecuteTargetInput{ID: algorithm}, Graph: req.Graph, Route: req.Route, Budget: req.Budget, Observation: req.Observation, Ablation: c.Ablation}, gate.RouteOptions{Observation: obs})
		}
		apiElapsedNS := time.Since(apiStarted).Nanoseconds()
		runtime.ReadMemStats(&memAfter)
		sampledPeak := stopHeapSampling()
		var observationErr error
		if collector != nil {
			observationErr = collector.Close(context.Background())
		}
		cancel()
		if err != nil {
			stopProgress(false, completedRuns)
			return out, fmt.Errorf("scenario %s query %s: %w", c.ID, query.ID, err)
		}
		if observationErr != nil {
			out.Failures = append(out.Failures, fmt.Sprintf("%s/%s observation failed: %v", c.ID, query.ID, observationErr))
		}
		out.Execution.RunOrder = append(out.Execution.RunOrder, runID)
		graphMeta := GraphProfile{
			GraphInstanceID:  graphInstanceID,
			Generator:        c.Graph.Generator,
			Topology:         c.Graph.Topology,
			GraphSeed:        graphSeed,
			ActualNodeCount:  g.NodeCount(),
			EdgeCount:        g.EdgeCount(),
			Directed:         g.Directed(),
			AverageOutDegree: averageOutDegree(g),
			EdgeDensity:      edgeDensity(g),
		}
		if c.Graph.Generator == "dataset" {
			if loaded, loadErr := LoadDataset(c.Graph.DatasetPath); loadErr == nil {
				graphMeta.Dataset = &loaded.Metadata
			}
		}
		runStartedAt := runStarted.UTC().Format(time.RFC3339Nano)
		raw := BenchmarkRun{
			RunMetadata: BenchmarkRunMetadata{
				RunOrdinal:         completedRuns + 1,
				RunID:              runID,
				RunName:            runName,
				ScenarioID:         c.ID,
				GraphInstanceID:    graphInstanceID,
				QueryID:            query.ID,
				AlgorithmID:        algorithm,
				ExecutionSeed:      seed,
				RepetitionIndex:    rep,
				WarmupRun:          plan.Warmup,
				ExecutionStartedAt: runStartedAt,
				ExecutionSucceeded: true,
			},
			ScenarioDefinition: ScenarioDefinition{
				ScenarioID:             c.ID,
				GraphGenerator:         c.Graph.Generator,
				GraphGeneratorSettings: c.Graph,
				RouteConfiguration:     c.Route,
				BudgetConfiguration:    c.Budget,
				AblationConfiguration:  c.Ablation,
				QuerySelectionMethod:   query.Selection.Method,
				ObservationMode:        s.Observation.Mode,
			},
			GraphProfile: graphMeta,
			QueryProfile: QueryProfile{
				QueryID:              query.ID,
				QuerySeed:            seed,
				QueryHash:            queryStableHash(query.ID, query.Selection.Method, source, target, seed),
				QuerySelectionMethod: query.Selection.Method,
				Source:               core.NodeID(source),
				Target:               core.NodeID(target),
			},
			AlgorithmConfiguration: AlgorithmConfiguration{
				AlgorithmID:           algorithm,
				RouteConfiguration:    c.Route,
				BudgetConfiguration:   c.Budget,
				AblationConfiguration: c.Ablation,
			},
			References: References{
				GraphSpecification: c.Graph,
			},
		}
		var found, exact bool
		var distance *float64
		var work core.WorkMetrics
		var solverMS, endMS float64
		var timeBreakdown core.TimeBreakdown
		var errorCode core.ErrorCode
		if algorithm == "bridge" {
			found = routeResult.Found
			exact = routeResult.Exact
			distance = routeResult.Distance
			work = routeResult.Work
			solverMS = routeResult.SolverTimeMS
			timeBreakdown = routeResult.TimeBreakdown
			endMS = routeResult.TimeMS
			errorCode = routeResult.ErrorCode
			raw.ExecutionResult.FailureReason = routeResult.FailureReason
			raw.ExecutionResult.TimeToFirstPathMS = routeResult.TimeToFirstPathMS
			raw.ExecutionResult.TimeToBestFoundMS = routeResult.TimeToBestFoundMS
			raw.ExecutionResult.ImprovementCount = routeResult.ImprovementCount
			raw.ExecutionResult.BridgeOverheadRatio = routeResult.BridgeOverheadRatio
			raw.ExecutionResult.DuplicatedWorkRatio = routeResult.DuplicatedWorkRatio
			raw.ExecutionResult.StateReuseRatio = routeResult.StateReuseRatio
			raw.ExecutionResult.BudgetLedger = routeResult.BudgetLedger
			raw.ExecutionResult.HandoffMetrics = routeResult.HandoffMetrics
			raw.ExecutionResult.BottleneckProfile = routeResult.BottleneckProfile
			raw.AlgorithmConfiguration.TargetKind = "system"
			raw.AlgorithmConfiguration.ExecutionPath = "route"
		} else {
			found = executeResult.Found
			exact = executeResult.Exact
			distance = executeResult.Distance
			work = executeResult.Work
			solverMS = executeResult.SolverTimeMS
			timeBreakdown = executeResult.TimeBreakdown
			endMS = executeResult.EndToEndMS
			errorCode = executeResult.ErrorCode
			raw.ExecutionResult.BudgetLedger = executeResult.BudgetLedger
			raw.AlgorithmConfiguration.TargetKind = executeResult.TargetKind
			raw.AlgorithmConfiguration.ExecutionPath = executeResult.ExecutionPath
		}
		if algorithm == "bridge" {
			raw.ExecutionResult.Path = make([]core.NodeID, len(routeResult.Path))
			for i, n := range routeResult.Path {
				raw.ExecutionResult.Path[i] = core.NodeID(n)
			}
		} else {
			raw.ExecutionResult.Path = make([]core.NodeID, len(executeResult.Path))
			for i, n := range executeResult.Path {
				raw.ExecutionResult.Path[i] = core.NodeID(n)
			}
		}
		raw.ExecutionResult.PathFound = found
		if algorithm == "bridge" {
			raw.ExecutionResult.SearchCompleted = routeResult.SearchCompleted
			raw.ExecutionResult.ReachabilityProven = routeResult.ReachabilityProven
		} else {
			raw.ExecutionResult.SearchCompleted = executeResult.SearchCompleted
			raw.ExecutionResult.ReachabilityProven = executeResult.ReachabilityProven
		}
		raw.ExecutionResult.OptimalityProven = exact
		raw.Measurement.Work = work
		raw.Measurement.TimeBreakdown = timeBreakdown
		raw.Measurement.SystemMetrics = core.SystemMetrics{
			AllocBytes:      memAfter.TotalAlloc - memBefore.TotalAlloc,
			MallocCount:     memAfter.Mallocs - memBefore.Mallocs,
			GCCount:         memAfter.NumGC - memBefore.NumGC,
			HeapAllocBefore: memBefore.HeapAlloc, HeapAllocAfter: memAfter.HeapAlloc,
			HeapAllocBoundaryMax: maxUint64(memBefore.HeapAlloc, memAfter.HeapAlloc), HeapAllocSampledPeak: sampledPeak,
		}
		raw.Measurement.SolverTimeNS = timeBreakdown.SolverNS
		raw.Measurement.EndToEndTimeNS = apiElapsedNS
		raw.Measurement.SolverTimeMS = solverMS
		raw.Measurement.EndToEndTimeMS = float64(apiElapsedNS) / 1_000_000
		raw.Measurement.ZeroDuration = apiElapsedNS <= 0
		raw.Measurement.TimingValid = apiElapsedNS > 0
		if timeBreakdown.SolverNS <= 0 {
			raw.Measurement.TimingValid = false
			raw.Measurement.TimingIssue = "solver boundary duration is zero; use repeated benchmark timing for solver ranking"
		}
		endMS = raw.Measurement.EndToEndTimeMS
		raw.ExecutionResult.ErrorCode = errorCode
		if raw.ExecutionResult.FailureReason == "" && !found {
			raw.ExecutionResult.FailureReason = classifyFailure(errorCode, raw.Measurement.SystemMetrics, false)
		}
		raw.ExecutionResult.TerminationReason = terminationReason(found, errorCode)
		raw.ExecutionResult.QualityClaims = QualityClaims{}
		if s.Observation.Mode != "minimum" {
			if algorithm == "bridge" {
				raw.Observations.ObservationData = routeResult.Observation
			} else {
				raw.Observations.ObservationData = executeResult.Observation
			}
		}
		if collector != nil {
			m := collector.Metrics()
			raw.Observations.QualityHistory = append([]ultrasound.QualityPoint(nil), m.QualityHistory...)
			raw.Observations.BudgetHistory = append([]ultrasound.BudgetPoint(nil), m.BudgetHistory...)
			raw.Observations.CollectorMetrics = &m
			if effectiveObservationMode == "debug" {
				raw.Observations.DebugSummary = buildDebugSummary(work, raw.ExecutionResult.BudgetLedger, m)
				raw.Observations.DebugSummary.HandoffMetrics = raw.ExecutionResult.HandoffMetrics
				raw.Observations.DebugSummary.BottleneckProfile = raw.ExecutionResult.BottleneckProfile
			}
		}
		if distance != nil {
			d := *distance
			raw.ExecutionResult.PathCost = &d
		}
		if invariantErr := validateBenchmarkRunClaims(raw); invariantErr != nil {
			stopProgress(false, completedRuns)
			return out, fmt.Errorf("scenario %s query %s algorithm %s produced invalid claims: %w", c.ID, query.ID, algorithm, invariantErr)
		}
		raw.RunMetadata.StableDigest = rawRunStableDigest(raw)
		if tracePath != "" && collector != nil {
			raw.References.TracePath = tracePath
			raw.References.TraceManifestPath = filepath.Join(traceBaseDir, "manifest.json")
			if absolute, absErr := filepath.Abs(raw.References.TracePath); absErr == nil {
				raw.References.TracePath = absolute
			}
			if absolute, absErr := filepath.Abs(raw.References.TraceManifestPath); absErr == nil {
				raw.References.TraceManifestPath = absolute
			}
			if err := writeTraceManifest(traceBaseDir, raw, collector.Metrics(), tracePath); err != nil {
				stopProgress(false, completedRuns)
				return out, err
			}
		}

		out.Runs = append(out.Runs, raw)

		completedRuns++
		stopProgress(true, completedRuns)
		if plan.Warmup {
			continue
		}
		key := c.ID + "\x00" + algorithm + "\x00" + query.ID
		acc := groups[key]
		if acc == nil {
			acc = &accumulator{result: ScenarioSummary{ScenarioID: c.ID, Algorithm: algorithm, QueryID: query.ID, TargetKind: raw.AlgorithmConfiguration.TargetKind, ExecutionPath: raw.AlgorithmConfiguration.ExecutionPath, MetricStatistics: map[string]SummaryStatistics{}, FailureReasons: map[string]int{}, Ablation: c.Ablation}, metrics: map[string][]float64{}, failureReasons: map[string]int{}}
			groups[key] = acc
		}
		acc.result.Runs++
		if found {
			acc.found++
			if distance != nil {
				acc.distances = append(acc.distances, *distance)
			}
		}
		if exact {
			acc.exact++
		}
		acc.works = append(acc.works, float64(work.TotalActions))
		acc.solverTimes = append(acc.solverTimes, solverMS)
		acc.endTimes = append(acc.endTimes, endMS)
		acc.metrics["expand_actions"] = append(acc.metrics["expand_actions"], float64(work.ExpandActions))
		acc.metrics["evaluate_actions"] = append(acc.metrics["evaluate_actions"], float64(work.EvaluateActions))
		acc.metrics["relax_actions"] = append(acc.metrics["relax_actions"], float64(work.RelaxActions))
		acc.metrics["enqueue_actions"] = append(acc.metrics["enqueue_actions"], float64(work.EnqueueActions))
		acc.metrics["anchor_ms"] = append(acc.metrics["anchor_ms"], timeBreakdown.AnchorMS)
		acc.metrics["bolts_ms"] = append(acc.metrics["bolts_ms"], timeBreakdown.BoltsMS)
		acc.metrics["fallback_ms"] = append(acc.metrics["fallback_ms"], timeBreakdown.FallbackMS)
		acc.metrics["alloc_bytes"] = append(acc.metrics["alloc_bytes"], float64(raw.Measurement.SystemMetrics.AllocBytes))
		acc.metrics["malloc_count"] = append(acc.metrics["malloc_count"], float64(raw.Measurement.SystemMetrics.MallocCount))
		acc.metrics["gc_count"] = append(acc.metrics["gc_count"], float64(raw.Measurement.SystemMetrics.GCCount))
		acc.metrics["first_path_elapsed_ms"] = appendOptional(acc.metrics["first_path_elapsed_ms"], raw.ExecutionResult.TimeToFirstPathMS)
		acc.metrics["best_path_elapsed_ms"] = appendOptional(acc.metrics["best_path_elapsed_ms"], raw.ExecutionResult.TimeToBestFoundMS)
		acc.metrics["improvement_count"] = append(acc.metrics["improvement_count"], float64(raw.ExecutionResult.ImprovementCount))
		acc.metrics["bridge_overhead_ratio"] = append(acc.metrics["bridge_overhead_ratio"], raw.ExecutionResult.BridgeOverheadRatio)
		acc.metrics["duplicated_work_ratio"] = append(acc.metrics["duplicated_work_ratio"], raw.ExecutionResult.DuplicatedWorkRatio)
		acc.metrics["state_reuse_ratio"] = append(acc.metrics["state_reuse_ratio"], raw.ExecutionResult.StateReuseRatio)
		if raw.ExecutionResult.FailureReason != "" {
			acc.failureReasons[raw.ExecutionResult.FailureReason]++
		}
	}
	enrichHandoffBaselines(out.Runs)
	for _, acc := range groups {
		r := &acc.result
		n := float64(r.Runs)
		r.FoundRate = float64(acc.found) / n
		r.ExactRate = float64(acc.exact) / n
		r.WorkStatistics = summarizeValues(acc.works)
		r.SolverTimeStatistics = summarizeValues(acc.solverTimes)
		r.EndToEndStatistics = summarizeValues(acc.endTimes)
		r.AverageWork = r.WorkStatistics.Mean
		r.AverageSolverTimeMS = r.SolverTimeStatistics.Mean
		r.AverageEndToEndMS = r.EndToEndStatistics.Mean
		r.AverageTimeMS = r.AverageEndToEndMS
		r.FailureReasons = acc.failureReasons
		for name, values := range acc.metrics {
			r.MetricStatistics[name] = summarizeValues(values)
		}
		if len(acc.distances) > 0 {
			st := summarizeValues(acc.distances)
			r.AverageDistance = st.Mean
			min, max := st.Min, st.Max
			r.MinDistance = &min
			r.MaxDistance = &max
		}
		out.ScenarioSummaries = append(out.ScenarioSummaries, *r)
	}
	sort.Slice(out.ScenarioSummaries, func(i, j int) bool {
		if out.ScenarioSummaries[i].ScenarioID == out.ScenarioSummaries[j].ScenarioID {
			if out.ScenarioSummaries[i].Algorithm == out.ScenarioSummaries[j].Algorithm {
				return out.ScenarioSummaries[i].QueryID < out.ScenarioSummaries[j].QueryID
			}
			return out.ScenarioSummaries[i].Algorithm < out.ScenarioSummaries[j].Algorithm
		}
		return out.ScenarioSummaries[i].ScenarioID < out.ScenarioSummaries[j].ScenarioID
	})
	completedAt := time.Now()
	out.RunMetadata.CompletedAt = completedAt.UTC().Format(time.RFC3339Nano)
	out.RunMetadata.DurationMS = float64(completedAt.Sub(started).Microseconds()) / 1000
	if err := writeExecutionArtifacts(executionDirectory, s, out); err != nil {
		return BenchmarkResult{}, err
	}
	return out, nil
}

func appendOptional(values []float64, value *float64) []float64 {
	if value == nil {
		return values
	}
	return append(values, *value)
}

func classifyFailure(code core.ErrorCode, _ core.SystemMetrics, fallbackUsed bool) string {
	switch code {
	case core.ErrDeadlineExceeded, core.ErrCancelled:
		return "timeout"
	case core.ErrBudgetExhausted:
		return "budget_exhausted"
	case core.ErrNoPath:
		if fallbackUsed {
			return "fallback_failure"
		}
		return "disconnected"
	case core.ErrInvalidRequest:
		return "invalid_request"
	default:
		return "no_path"
	}
}

func startHeapSampler(enabled bool, initial uint64) func() uint64 {
	if !enabled {
		return func() uint64 { return 0 }
	}
	done := make(chan struct{})
	result := make(chan uint64, 1)
	go func() {
		peak := initial
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				if m.HeapAlloc > peak {
					peak = m.HeapAlloc
				}
			case <-done:
				result <- peak
				return
			}
		}
	}()
	return func() uint64 { close(done); return <-result }
}

func averageOutDegree(g *core.AdjacencyGraph) float64 {
	if g == nil || g.NodeCount() == 0 {
		return 0
	}
	return float64(g.EdgeCount()) / float64(g.NodeCount())
}

func edgeDensity(g *core.AdjacencyGraph) float64 {
	if g == nil || g.NodeCount() < 2 {
		return 0
	}
	nodes := float64(g.NodeCount())
	denominator := nodes * (nodes - 1)
	if !g.Directed() {
		denominator /= 2
	}
	if denominator == 0 {
		return 0
	}
	return float64(g.EdgeCount()) / denominator
}

func queryStableHash(queryID, strategy string, source, target uint32, seed int64) string {
	payload := struct {
		QueryID   string `json:"query_id"`
		Strategy  string `json:"query_selection_method"`
		Source    uint32 `json:"source"`
		Target    uint32 `json:"target"`
		QuerySeed int64  `json:"query_seed"`
	}{queryID, strategy, source, target, seed}
	b, _ := json.Marshal(payload)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func terminationReason(found bool, code core.ErrorCode) string {
	if found {
		return "path_returned"
	}
	switch code {
	case core.ErrDeadlineExceeded, core.ErrCancelled:
		return "timeout"
	case core.ErrBudgetExhausted:
		return "budget_exhausted"
	case core.ErrNoPath:
		return "unreachable"
	case core.ErrInvalidRequest:
		return "invalid_request"
	case "":
		return "completed_without_path"
	default:
		return "error"
	}
}

func rawRunStableDigest(raw BenchmarkRun) string {
	payload := struct {
		ScenarioID string           `json:"scenario_id"`
		Algorithm  string           `json:"algorithm"`
		QueryID    string           `json:"query_id"`
		Seed       int64            `json:"seed"`
		Found      bool             `json:"path_found"`
		Exact      bool             `json:"optimality_proven"`
		Distance   *float64         `json:"path_cost,omitempty"`
		Work       core.WorkMetrics `json:"work"`
		ErrorCode  core.ErrorCode   `json:"error_code,omitempty"`
	}{
		raw.RunMetadata.ScenarioID,
		raw.RunMetadata.AlgorithmID,
		raw.RunMetadata.QueryID,
		raw.RunMetadata.ExecutionSeed,
		raw.ExecutionResult.PathFound,
		raw.ExecutionResult.OptimalityProven,
		raw.ExecutionResult.PathCost,
		raw.Measurement.Work,
		raw.ExecutionResult.ErrorCode,
	}
	b, _ := json.Marshal(payload)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func writeTraceManifest(dir string, raw BenchmarkRun, metrics ultrasound.CollectorMetrics, tracePath string) error {
	b, err := os.ReadFile(tracePath)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(b)
	manifest := map[string]any{
		"schema_version":       "bridge.trace.v1",
		"run_id":               raw.RunMetadata.RunID,
		"created_at":           time.Now().UTC().Format(time.RFC3339Nano),
		"ultrasound_mode":      raw.ScenarioDefinition.ObservationMode,
		"sample_rate":          1.0,
		"trace_complete":       !metrics.Truncated && metrics.DroppedEvents == 0,
		"first_sequence":       metrics.FirstSequence,
		"last_sequence":        metrics.LastSequence,
		"event_count":          metrics.EventCount,
		"dropped_event_count":  metrics.DroppedEvents,
		"truncated":            metrics.Truncated,
		"observer_overhead_ns": metrics.ObservationNS,
		"sink_write_ns":        metrics.SinkWriteNS,
		"stable_digest":        raw.RunMetadata.StableDigest,
		"trace_sha256":         hex.EncodeToString(sum[:]),
		"trace_file":           filepath.Base(tracePath),
	}
	return writeJSONFile(filepath.Join(dir, "manifest.json"), manifest, true)
}

func fileSHA256(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func formatProgressDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSeconds := int(d.Round(time.Second) / time.Second)
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func progressMaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func startRunProgressReporter(
	started time.Time,
	runStarted time.Time,
	runName string,
	runOrdinal int,
	totalRuns int,
	reporter ProgressReporter,
) func(finished bool, completedRuns int) {
	if reporter == nil {
		return func(bool, int) {}
	}

	report := func(finished bool, completedRuns int) {
		totalElapsed := time.Since(started)
		runElapsed := time.Since(runStarted)
		startedRuns := runOrdinal
		if finished {
			startedRuns = completedRuns
		}
		avgRun := totalElapsed / time.Duration(progressMaxInt(1, startedRuns))
		remaining := totalRuns - completedRuns
		if !finished {
			remaining = totalRuns - runOrdinal
		}
		reporter.ReportRunProgress(RunProgress{
			RunName:      runName,
			Current:      runOrdinal,
			Completed:    completedRuns,
			Total:        totalRuns,
			RunElapsed:   runElapsed,
			TotalElapsed: totalElapsed,
			ETA:          avgRun * time.Duration(progressMaxInt(0, remaining)),
			Finished:     finished,
		})
	}

	report(false, runOrdinal-1)

	ticker := time.NewTicker(time.Second)
	done := make(chan struct{})
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				report(false, runOrdinal-1)
			case <-done:
				return
			}
		}
	}()

	return func(finished bool, completedRuns int) {
		close(done)
		report(finished, completedRuns)
	}
}

func writeJSONFile(path string, value any, overwrite bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	flags := os.O_WRONLY | os.O_CREATE
	if overwrite {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func graphToInput(g core.Graph) gate.GraphInput {
	nodes := make([]gate.GraphNode, g.NodeCount())
	for i := range nodes {
		id := uint32(i)
		nodes[i] = gate.GraphNode{ID: id}
		if p, ok := g.Position(core.NodeID(i)); ok {
			x, y := p.X, p.Y
			nodes[i].X = &x
			nodes[i].Y = &y
		}
	}
	edges := []gate.GraphEdge{}
	for i := 0; i < g.NodeCount(); i++ {
		for _, e := range g.EdgesFrom(core.NodeID(i)) {
			if !g.Directed() && uint32(i) > uint32(e.To) {
				continue
			}
			edges = append(edges, gate.GraphEdge{From: uint32(i), To: uint32(e.To), Weight: e.Weight})
		}
	}
	return gate.GraphInput{Type: "inline", Directed: g.Directed(), Nodes: nodes, Edges: edges}
}

func maxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

// validateBenchmarkRunClaims rejects internally contradictory benchmark data.
// Invalid proof or timing claims must fail the run rather than silently enter
// aggregate research output.
func validateBenchmarkRunClaims(run BenchmarkRun) error {
	result := run.ExecutionResult
	measurement := run.Measurement
	if result.OptimalityProven && !result.PathFound {
		return fmt.Errorf("optimality_proven requires path_found")
	}
	if result.OptimalityProven && !result.SearchCompleted {
		return fmt.Errorf("optimality_proven requires search_completed")
	}
	if result.PathFound && !result.ReachabilityProven {
		return fmt.Errorf("path_found requires reachability_proven")
	}
	if result.ErrorCode == core.ErrNoPath && (!result.SearchCompleted || !result.ReachabilityProven || result.PathFound) {
		return fmt.Errorf("NO_PATH requires completed unreachable proof")
	}
	if result.ErrorCode == core.ErrBudgetExhausted && (result.SearchCompleted || result.ReachabilityProven || result.OptimalityProven) {
		return fmt.Errorf("budget exhaustion cannot produce completion or proof claims")
	}
	if measurement.TimingValid && (measurement.EndToEndTimeNS <= 0 || measurement.SolverTimeNS <= 0) {
		return fmt.Errorf("timing_valid requires positive end-to-end and solver durations")
	}
	return nil
}

func validateSafeID(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	for _, r := range value {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return fmt.Errorf("%s contains unsafe character %q", field, r)
		}
	}
	return nil
}

func newExecutionID() (string, error) {
	var random [10]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%013d-%s", time.Now().UTC().UnixMilli(), hex.EncodeToString(random[:])), nil
}

func durationMillisecondsPointer(value string) *float64 {
	if value == "" {
		return nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return nil
	}
	ms := float64(d) / float64(time.Millisecond)
	return &ms
}

func writeExecutionArtifacts(directory string, scenario BenchmarkScenario, result BenchmarkResult) error {
	if err := writeJSONFile(filepath.Join(directory, "result.json"), result, false); err != nil {
		return err
	}
	if err := writeJSONFile(filepath.Join(directory, "scenario.json"), scenario, false); err != nil {
		return err
	}
	if err := writeJSONFile(filepath.Join(directory, "environment.json"), result.Environment, false); err != nil {
		return err
	}
	manifest := map[string]any{
		"schema_version":          "bridge.benchmark.execution.v1",
		"execution_id":            result.ExecutionID,
		"suite_id":                result.SuiteID,
		"scenario_schema_version": scenario.SchemaVersion,
		"observation_mode":        scenario.Observation.Mode,
		"started_at":              result.RunMetadata.StartedAt,
		"completed_at":            result.RunMetadata.CompletedAt,
		"output_directory":        result.OutputDirectory,
		"run_order":               result.Execution.RunOrder,
	}
	if err := writeJSONFile(filepath.Join(directory, "manifest.json"), manifest, false); err != nil {
		return err
	}
	if err := writeJSONLines(filepath.Join(directory, "runs.jsonl"), result.Runs); err != nil {
		return err
	}
	if err := writeSummaryCSV(filepath.Join(directory, "summary.csv"), result); err != nil {
		return err
	}
	return writeHandoffCSV(filepath.Join(directory, "handoffs.csv"), result.Runs)
}

func writeJSONLines(path string, values []BenchmarkRun) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, value := range values {
		if err := enc.Encode(value); err != nil {
			return err
		}
	}
	return nil
}

func writeSummaryCSV(path string, result BenchmarkResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := fmt.Fprintln(f, "suite_id,scenario_id,algorithm,query_id,runs,path_found_rate,optimality_proven_rate,mean_path_cost,mean_work_actions,mean_solver_time_ms,mean_end_to_end_time_ms"); err != nil {
		return err
	}
	for _, row := range result.ScenarioSummaries {
		if _, err := fmt.Fprintf(f, "%s,%s,%s,%s,%d,%.9g,%.9g,%.17g,%.17g,%.17g,%.17g\n", result.SuiteID, row.ScenarioID, row.Algorithm, row.QueryID, row.Runs, row.FoundRate, row.ExactRate, row.AverageDistance, row.AverageWork, row.AverageSolverTimeMS, row.AverageEndToEndMS); err != nil {
			return err
		}
	}
	return nil
}

func buildDebugSummary(work core.WorkMetrics, ledger *core.BudgetLedger, metrics ultrasound.CollectorMetrics) *DebugSummary {
	out := &DebugSummary{
		ActionCounts: map[string]uint64{
			"select": work.SelectActions, "expand": work.ExpandActions, "evaluate": work.EvaluateActions,
			"relax": work.RelaxActions, "enqueue": work.EnqueueActions, "reject": work.RejectActions,
			"backtrack": work.BacktrackActions, "connect": work.ConnectActions, "candidate": work.CandidateActions,
			"repair": work.RepairActions, "bound": work.BoundActions, "terminate": work.TerminateActions,
			"hypothesis": work.HypothesisActions, "evidence": work.EvidenceActions, "handoff": work.HandoffActions,
			"schedule": work.ScheduleActions,
		},
		WorkByComponent: map[string]uint64{}, BudgetGrantedByPurpose: map[string]uint64{}, BudgetUsedByPurpose: map[string]uint64{},
		CandidateUpdateCount: maxUint64(metrics.DebugSummary.CandidateUpdateCount, work.CandidateActions), FallbackCount: metrics.DebugSummary.FallbackCount,
		CertificationCount: metrics.DebugSummary.CertificationCount, StateReuseAppliedCount: metrics.DebugSummary.StateReuseAppliedCount,
		MaxFrontierSize: metrics.DebugSummary.MaxFrontierSize, ComponentEventCounts: metrics.DebugSummary.ComponentEventCounts,
		ObservationOverheadNS: metrics.ObservationNS, TraceSinkWriteNS: metrics.SinkWriteNS,
		DroppedEvents: metrics.DroppedEvents, Truncated: metrics.Truncated,
	}
	if ledger != nil {
		for k, v := range ledger.ByComponent {
			out.WorkByComponent[string(k)] = v
		}
		for _, e := range ledger.Entries {
			if e.Granted != nil {
				out.BudgetGrantedByPurpose[e.Purpose] += *e.Granted
			}
			out.BudgetUsedByPurpose[e.Purpose] += e.Used
			id := strings.ToLower(e.TaskID)
			if strings.Contains(id, "certif") {
				out.CertificationCount++
			}
			if strings.Contains(id, "fallback") || strings.Contains(id, "emergency") {
				out.FallbackCount++
			}
		}
	}
	return out
}

func enrichHandoffBaselines(runs []BenchmarkRun) {
	type key struct {
		scenario, graph, query string
		seed                   int64
		rep                    int
	}
	baseline := map[key]BenchmarkRun{}
	for _, run := range runs {
		if run.RunMetadata.WarmupRun || run.RunMetadata.AlgorithmID != "weighted_astar" {
			continue
		}
		baseline[key{run.RunMetadata.ScenarioID, run.RunMetadata.GraphInstanceID, run.RunMetadata.QueryID, run.RunMetadata.ExecutionSeed, run.RunMetadata.RepetitionIndex}] = run
	}
	for i := range runs {
		h := runs[i].ExecutionResult.HandoffMetrics
		if h == nil {
			continue
		}
		b, ok := baseline[key{runs[i].RunMetadata.ScenarioID, runs[i].RunMetadata.GraphInstanceID, runs[i].RunMetadata.QueryID, runs[i].RunMetadata.ExecutionSeed, runs[i].RunMetadata.RepetitionIndex}]
		if !ok {
			continue
		}
		for j := range h.Records {
			bw := b.Measurement.Work.TotalActions
			bt := b.Measurement.SolverTimeNS
			h.Records[j].BoltsStandaloneWork = &bw
			dw := int64(h.Records[j].AnchorWorkAtHandoff+h.Records[j].BoltsWork) - int64(bw)
			h.Records[j].AdditionalWorkVsBoltsStandalone = &dw
			h.Records[j].BoltsStandaloneTimeNS = &bt
			dt := h.Records[j].BoltsTimeNS - bt
			h.Records[j].AdditionalTimeNSVsBoltsStandalone = &dt
		}
	}
}

func writeHandoffCSV(path string, runs []BenchmarkRun) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = fmt.Fprintln(f, "scenario_id,algorithm,run_id,handoff_sequence,reason,anchor_work_at_handoff,bolts_work,bolts_time_ms,available_state_units,transferred_state_units,reused_state_units,pre_handoff_waste_work,bolts_standalone_work,additional_work_vs_bolts_standalone,bolts_standalone_time_ms,additional_time_ms_vs_bolts_standalone,dominant_work_component,dominant_time_component"); err != nil {
		return err
	}
	for _, run := range runs {
		h := run.ExecutionResult.HandoffMetrics
		if h == nil {
			continue
		}
		for _, r := range h.Records {
			bw, dw, bt, dt := "", "", "", ""
			if r.BoltsStandaloneWork != nil {
				bw = fmt.Sprint(*r.BoltsStandaloneWork)
			}
			if r.AdditionalWorkVsBoltsStandalone != nil {
				dw = fmt.Sprint(*r.AdditionalWorkVsBoltsStandalone)
			}
			if r.BoltsStandaloneTimeNS != nil {
				bt = fmt.Sprintf("%.6f", float64(*r.BoltsStandaloneTimeNS)/1e6)
			}
			if r.AdditionalTimeNSVsBoltsStandalone != nil {
				dt = fmt.Sprintf("%.6f", float64(*r.AdditionalTimeNSVsBoltsStandalone)/1e6)
			}
			dwcomp, dtcomp := "", ""
			if run.ExecutionResult.BottleneckProfile != nil {
				dwcomp = run.ExecutionResult.BottleneckProfile.DominantWorkComponent
				dtcomp = run.ExecutionResult.BottleneckProfile.DominantTimeComponent
			}
			if _, err = fmt.Fprintf(f, "%s,%s,%s,%d,%s,%d,%d,%.6f,%d,%d,%d,%d,%s,%s,%s,%s,%s,%s\n", run.RunMetadata.ScenarioID, run.RunMetadata.AlgorithmID, run.RunMetadata.RunID, r.Sequence, r.Reason, r.AnchorWorkAtHandoff, r.BoltsWork, float64(r.BoltsTimeNS)/1e6, r.AvailableStateUnits, r.TransferredStateUnits, r.ReusedStateUnits, r.PreHandoffWasteWork, bw, dw, bt, dt, dwcomp, dtcomp); err != nil {
				return err
			}
		}
	}
	return nil
}
