package traffic

import (
	"context"
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
	"time"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/ultrasound"
)

const BenchmarkSchemaV1 = "bridge.benchmark.v2"
const BenchmarkResultSchemaV1 = "bridge.benchmark.artifact.v2"

type BenchmarkScenario struct {
	SchemaVersion string          `json:"schema_version" yaml:"schema_version"`
	Suite         SuiteSpec       `json:"suite" yaml:"suite"`
	Execution     ExecutionSpec   `json:"execution" yaml:"execution"`
	Algorithms    []string        `json:"algorithms" yaml:"algorithms"`
	Observation   ObservationSpec `json:"observation_config,omitempty" yaml:"observation_config,omitempty"`
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
	Jobs           int     `json:"jobs,omitempty" yaml:"jobs,omitempty"`
	Timeout        string  `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	RandomizeOrder bool    `json:"randomize_order,omitempty" yaml:"randomize_order,omitempty"`
}
type ObservationSpec struct {
	Mode       string  `json:"level,omitempty" yaml:"level,omitempty"`
	SampleRate float64 `json:"sample_rate,omitempty" yaml:"sample_rate,omitempty"`
}
type OutputSpec struct {
	OutputDir          string            `json:"output_dir,omitempty" yaml:"output_dir,omitempty"`
	ArtifactID         string            `json:"artifact_id,omitempty" yaml:"artifact_id,omitempty"`
	SaveRawResults     bool              `json:"save_raw_results,omitempty" yaml:"save_raw_results,omitempty"`
	SaveGraphSnapshot  bool              `json:"save_graph_snapshot,omitempty" yaml:"save_graph_snapshot,omitempty"`
	SaveTrace          bool              `json:"save_trace,omitempty" yaml:"save_trace,omitempty"`
	CaptureEnvironment bool              `json:"capture_environment,omitempty" yaml:"capture_environment,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
type ScenarioCase struct {
	ID        string        `json:"id" yaml:"id"`
	Graph     GeneratorSpec `json:"graph" yaml:"graph"`
	Endpoints EndpointSpec  `json:"endpoints,omitempty" yaml:"endpoints,omitempty"`
	Queries   []QuerySpec   `json:"queries,omitempty" yaml:"queries,omitempty"`
	Route     RouteSpec     `json:"route,omitempty" yaml:"route,omitempty"`
	Budget    BudgetSpec    `json:"budget,omitempty" yaml:"budget,omitempty"`
	Ablation  AblationSpec  `json:"ablation,omitempty" yaml:"ablation,omitempty"`
}

type QuerySpec struct {
	ID       string  `json:"id" yaml:"id"`
	Strategy string  `json:"query_selection_method,omitempty" yaml:"query_selection_method,omitempty"`
	Source   *uint32 `json:"source,omitempty" yaml:"source,omitempty"`
	Target   *uint32 `json:"target,omitempty" yaml:"target,omitempty"`
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
type EndpointSpec struct {
	Strategy string  `json:"query_selection_method" yaml:"query_selection_method"`
	Source   *uint32 `json:"source,omitempty" yaml:"source,omitempty"`
	Target   *uint32 `json:"target,omitempty" yaml:"target,omitempty"`
}
type RouteSpec struct {
	Mode    core.RouteMode `json:"route_mode,omitempty" yaml:"route_mode,omitempty"`
	Workers int            `json:"logical_worker_count,omitempty" yaml:"logical_worker_count,omitempty"`
}
type AblationSpec = core.AblationOptions

type BudgetSpec struct {
	TotalWork *uint64  `json:"total_work,omitempty" yaml:"total_work,omitempty"`
	TimeoutMS *float64 `json:"timeout_ms,omitempty" yaml:"timeout_ms,omitempty"`
}
type BenchmarkResult struct {
	SchemaVersion      string               `json:"schema_version"`
	TerminologyVersion string               `json:"terminology_version"`
	SuiteID            string               `json:"suite_id"`
	ArtifactID         string               `json:"artifact_id,omitempty"`
	RunMetadata        ArtifactRunMetadata  `json:"run_metadata"`
	Execution          ExecutionManifest    `json:"execution"`
	Environment        *EnvironmentMetadata `json:"environment,omitempty"`
	Runs               []BenchmarkRun       `json:"runs"`
	ScenarioSummaries  []ScenarioSummary    `json:"scenario_summaries"`
	Failures           []string             `json:"failures,omitempty"`
}

type ArtifactRunMetadata struct {
	ScenarioSchemaVersion string            `json:"scenario_schema_version"`
	StartedAt             string            `json:"started_at"`
	CompletedAt           string            `json:"completed_at,omitempty"`
	DurationMS            float64           `json:"duration_ms"`
	ExecutionSucceeded    bool              `json:"execution_succeeded"`
	ObservationMode       string            `json:"observation_mode"`
	ObservationSampleRate float64           `json:"observation_sample_rate,omitempty"`
	OutputMetadata        map[string]string `json:"output_metadata,omitempty"`
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
	if s.Execution.Repetitions == 0 {
		s.Execution.Repetitions = 1
	}
	if len(s.Execution.Seeds) == 0 {
		s.Execution.Seeds = []int64{1}
	}
	if s.Execution.Jobs == 0 {
		s.Execution.Jobs = 1
	}
	if s.Observation.SampleRate == 0 {
		s.Observation.SampleRate = 1
	}
	if len(s.Algorithms) == 0 {
		s.Algorithms = []string{"bridge"}
	}
	if s.Observation.Mode == "" {
		s.Observation.Mode = "off"
	}
	for i := range s.Scenarios {
		if s.Scenarios[i].Graph.Generator == "" {
			s.Scenarios[i].Graph.Generator = "grid"
		}
		if s.Scenarios[i].Graph.Topology == "" {
			s.Scenarios[i].Graph.Topology = "open"
		}
		if len(s.Scenarios[i].Queries) == 0 && s.Scenarios[i].Endpoints.Strategy == "" {
			s.Scenarios[i].Endpoints.Strategy = "generator_default_endpoints"
		}
		for q := range s.Scenarios[i].Queries {
			if s.Scenarios[i].Queries[q].Strategy == "" {
				s.Scenarios[i].Queries[q].Strategy = "generator_default_endpoints"
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
	if s.Execution.Jobs != 1 {
		return errors.New("execution.jobs must be 1 in v0.12.1")
	}
	if s.Execution.Timeout != "" {
		if _, err := time.ParseDuration(s.Execution.Timeout); err != nil {
			return fmt.Errorf("execution.timeout: %w", err)
		}
	}
	switch s.Observation.Mode {
	case "off", "aggregate", "trace":
	default:
		return fmt.Errorf("observation_config.level must be one of off, aggregate, trace")
	}
	if s.Observation.SampleRate <= 0 || s.Observation.SampleRate > 1 {
		return errors.New("observation.sample_rate must be > 0 and <= 1")
	}
	if s.Output.OutputDir == "" && (s.Output.SaveRawResults || s.Output.SaveGraphSnapshot || s.Output.SaveTrace) {
		return errors.New("output.output_dir is required when output.save_raw_results, output.save_graph_snapshot, or output.save_trace is enabled")
	}
	if s.Output.SaveTrace && s.Observation.Mode == "off" {
		return errors.New("observation_config.level must not be off when output.save_trace is enabled")
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
			if q.Strategy != "generator_default_endpoints" && q.Strategy != "explicit_endpoints" {
				return fmt.Errorf("scenario %q query %q: unsupported query_selection_method", c.ID, q.ID)
			}
			if q.Strategy == "explicit_endpoints" && (q.Source == nil || q.Target == nil) {
				return fmt.Errorf("scenario %q query %q: explicit_endpoints requires source and target", c.ID, q.ID)
			}
			if q.Strategy == "explicit_endpoints" && (int(*q.Source) >= nodeCount || int(*q.Target) >= nodeCount) {
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
		if c.Budget.TotalWork != nil && *c.Budget.TotalWork == 0 {
			return fmt.Errorf("scenario %q: budget.total_work must be > 0", c.ID)
		}
		if c.Budget.TimeoutMS != nil && *c.Budget.TimeoutMS <= 0 {
			return fmt.Errorf("scenario %q: budget.timeout_ms must be > 0", c.ID)
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
	TraceDir         string
	Overwrite        bool
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
		ArtifactID:         s.Output.ArtifactID,
		RunMetadata: ArtifactRunMetadata{
			ScenarioSchemaVersion: s.SchemaVersion,
			StartedAt:             started.UTC().Format(time.RFC3339Nano),
			ExecutionSucceeded:    true,
			ObservationMode:       s.Observation.Mode,
			ObservationSampleRate: s.Observation.SampleRate,
			OutputMetadata:        s.Output.Metadata,
		},
	}
	out.Execution = ExecutionManifest{Randomized: s.Execution.RandomizeOrder, RunOrder: []string{}}
	if s.Execution.RandomizeOrder {
		out.Execution.ShuffleSeed = s.Execution.Seeds[0]
		out.Execution.ShuffleAlgorithm = "math/rand-v1"
	}
	if s.Output.CaptureEnvironment {
		out.Environment = captureEnvironment()
	}
	plans := expandRunPlans(s)
	completedRuns := 0
	var timeout time.Duration
	if s.Execution.Timeout != "" {
		timeout, _ = time.ParseDuration(s.Execution.Timeout)
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
		if query.Strategy == "explicit_endpoints" {
			source, target = *query.Source, *query.Target
		} else {
			source, target = uint32(defaultSource), uint32(defaultTarget)
		}
		requestID := fmt.Sprintf("%s-%s-%d-%s-%d", c.ID, algorithm, seed, query.ID, rep)
		req := gate.RouteRequest{SchemaVersion: gate.RouteRequestSchemaV1, RequestID: requestID, Graph: graphToInput(g), Route: gate.RouteInput{Source: source, Target: target, Mode: c.Route.Mode, Workers: c.Route.Workers, Seed: uint64(seed)}, Budget: gate.BudgetInput{TotalWork: c.Budget.TotalWork, TimeoutMS: c.Budget.TimeoutMS}, Observation: gate.ObservationInput{Mode: gate.ObservationMode(s.Observation.Mode), SampleRate: &s.Observation.SampleRate}, Ablation: c.Ablation}
		runCtx := ctx
		cancel := func() {}
		if timeout > 0 {
			runCtx, cancel = context.WithTimeout(ctx, timeout)
		}
		traceBaseDir := opts.TraceDir
		if traceBaseDir == "" && s.Output.SaveTrace {
			traceBaseDir = filepath.Join(s.Output.OutputDir, c.ID, runDirName)
		}
		var collector *ultrasound.Collector
		var tracePath string
		if s.Observation.Mode != "off" {
			var sink ultrasound.EventSink = ultrasound.DiscardSink{}
			if traceBaseDir != "" && s.Observation.Mode == "trace" {
				tracePath = filepath.Join(traceBaseDir, "trace.jsonl")
				fs, sinkErr := ultrasound.NewFileSink(tracePath, opts.Overwrite)
				if sinkErr != nil {
					cancel()
					stopProgress(false, completedRuns)
					return out, sinkErr
				}
				sink = fs
			}
			collector = ultrasound.NewCollectorConfigured(s.Observation.Mode, sink, 0, s.Observation.SampleRate, uint64(seed)^uint64(rep))
		}
		obs := gate.ObservationOptions{Mode: gate.ObservationMode(s.Observation.Mode)}
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
		if algorithm == "bridge" {
			routeResult, err = router.Route(runCtx, req, gate.RouteOptions{Observation: obs})
		} else {
			executeResult, err = router.ExecuteOnce(runCtx, gate.ExecuteRequest{SchemaVersion: gate.ExecuteRequestSchemaV1, RequestID: requestID, Target: gate.ExecuteTargetInput{ID: algorithm}, Graph: req.Graph, Route: req.Route, Budget: req.Budget, Observation: req.Observation, Ablation: c.Ablation}, gate.RouteOptions{Observation: obs})
		}
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
				QuerySelectionMethod:   query.Strategy,
				ObservationMode:        s.Observation.Mode,
				ObservationSampleRate:  s.Observation.SampleRate,
				OutputMetadata:         s.Output.Metadata,
			},
			GraphProfile: graphMeta,
			QueryProfile: QueryProfile{
				QueryID:              query.ID,
				QuerySeed:            seed,
				QueryHash:            queryStableHash(query.ID, query.Strategy, source, target, seed),
				QuerySelectionMethod: query.Strategy,
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
		raw.ExecutionResult.SearchCompleted = found || errorCode == ""
		raw.ExecutionResult.ReachabilityProven = found || errorCode == core.ErrNoPath
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
		raw.Measurement.SolverTimeMS = solverMS
		raw.Measurement.EndToEndTimeMS = endMS
		raw.Measurement.ZeroDuration = endMS == 0 || solverMS == 0
		raw.ExecutionResult.ErrorCode = errorCode
		if raw.ExecutionResult.FailureReason == "" && !found {
			raw.ExecutionResult.FailureReason = classifyFailure(errorCode, raw.Measurement.SystemMetrics, false)
		}
		raw.ExecutionResult.TerminationReason = terminationReason(found, errorCode)
		raw.ExecutionResult.QualityClaims = QualityClaims{}
		if s.Observation.Mode != "off" {
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
		}
		if distance != nil {
			d := *distance
			raw.ExecutionResult.PathCost = &d
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
			if err := writeTraceManifest(traceBaseDir, raw, s.Observation.SampleRate, collector.Metrics(), tracePath); err != nil {
				stopProgress(false, completedRuns)
				return out, err
			}
		}
		if s.Output.OutputDir != "" && (s.Output.SaveRawResults || s.Output.SaveGraphSnapshot) {
			runDir := filepath.Join(s.Output.OutputDir, c.ID, runDirName)
			if s.Output.SaveGraphSnapshot {
				graphPath := filepath.Join(runDir, "graph.json")
				graphSnapshot := graphToInput(g)
				if err := writeJSONFile(graphPath, graphSnapshot, opts.Overwrite); err != nil {
					stopProgress(false, completedRuns)
					return out, err
				}
				raw.References.GraphSnapshotPath = graphPath
				if absolute, absErr := filepath.Abs(raw.References.GraphSnapshotPath); absErr == nil {
					raw.References.GraphSnapshotPath = absolute
				}
				graphSHA, shaErr := fileSHA256(raw.References.GraphSnapshotPath)
				if shaErr != nil {
					stopProgress(false, completedRuns)
					return out, shaErr
				}
				raw.References.GraphSnapshotSHA256 = graphSHA
			}
		}
		out.Runs = append(out.Runs, raw)
		if s.Output.OutputDir != "" && s.Output.SaveRawResults {
			runDir := filepath.Join(s.Output.OutputDir, c.ID, runDirName)
			if err := writeJSONFile(filepath.Join(runDir, "raw-result.json"), raw, opts.Overwrite); err != nil {
				stopProgress(false, completedRuns)
				return out, err
			}
		}
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
	if s.Output.OutputDir != "" {
		if err := writeJSONFile(filepath.Join(s.Output.OutputDir, "result.json"), out, opts.Overwrite); err != nil {
			return BenchmarkResult{}, err
		}
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

func writeTraceManifest(dir string, raw BenchmarkRun, sampleRate float64, metrics ultrasound.CollectorMetrics, tracePath string) error {
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
		"sample_rate":          sampleRate,
		"sampling_algorithm":   "fnv1a-seed-ordinal-kind-v1",
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
