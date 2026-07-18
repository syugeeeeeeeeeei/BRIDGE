package traffic

import (
	"context"
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/ultrasound"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"
)

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
		runOrdinal := completedRuns + 1
		c, algorithm, seed, rep, query := plan.Scenario, plan.Algorithm, plan.Seed, plan.Repetition, plan.Query
		graphSeed := seed
		graphInstanceID := fmt.Sprintf("%s/seed-%d", c.ID, graphSeed)
		runID := fmt.Sprintf("%s/%s/%s/%s/rep-%d", c.ID, algorithm, graphInstanceID, query.ID, rep)
		if plan.Warmup {
			runID += "/warmup"
		}
		runName := runID
		runDirectoryName := fmt.Sprintf("run-%06d", runOrdinal)
		runStarted := time.Now()
		stopProgress := startRunProgressReporter(started, runStarted, runName, completedRuns+1, len(plans), opts.ProgressReporter)
		effectiveObservationMode := s.Observation.Mode
		// Warmups stabilize code and data paths but do not retain detailed observation data.
		if plan.Warmup {
			effectiveObservationMode = "minimum"
		}
		traceBaseDir := ""
		if effectiveObservationMode == "trace" {
			traceBaseDir = filepath.Join(executionDirectory, "traces", runDirectoryName)
		}
		var collector *ultrasound.Collector
		var tracePath string
		var graphSnapshotPath string
		if effectiveObservationMode != "off" {
			var sink ultrasound.EventSink = ultrasound.DiscardSink{}
			if traceBaseDir != "" {
				tracePath = filepath.Join(traceBaseDir, "trace.jsonl")
				fs, sinkErr := ultrasound.NewFileSink(tracePath, false)
				if sinkErr != nil {
					stopProgress(false, completedRuns)
					return out, sinkErr
				}
				sink = fs
			}
			collector = ultrasound.NewCollector(effectiveObservationMode, sink)
		}
		var observer bearing.Observer
		if collector != nil {
			observer = collector
		}
		// Graph generation is benchmark setup, not part of the measured benchmark run.
		// Keep it as an independent reference span so setup cost remains observable
		// without contaminating route-oriented elapsed-time analysis.
		_, finishGraphSpan := bearing.BeginLifecycle(observer, runID, "benchmark-setup", "", "TRAFFIC", "graph_generation")
		g, defaultSource, defaultTarget, err := BuildScenarioGraphAndEndpoints(c.Graph, graphSeed)
		finishGraphSpan(err != nil)
		if err != nil {
			stopProgress(false, completedRuns)
			return out, err
		}
		runSpan, finishRunSpan := bearing.BeginLifecycle(observer, runID, "benchmark-run", "", "TRAFFIC", "benchmark_run")
		_, finishQuerySpan := bearing.BeginLifecycle(observer, runID, "benchmark-run", runSpan, "TRAFFIC", "query_generation")
		var source, target uint32
		if query.Selection.Method == "explicit" {
			source, target = *query.Selection.Source, *query.Selection.Target
		} else {
			source, target = uint32(defaultSource), uint32(defaultTarget)
		}
		finishQuerySpan(false)
		requestID := fmt.Sprintf("%s-%s-%d-%s-%d", c.ID, algorithm, seed, query.ID, rep)
		_, finishConversionSpan := bearing.BeginLifecycle(observer, runID, "benchmark-run", runSpan, "TRAFFIC", "graph_conversion")
		graphInput := graphToInput(g)
		finishConversionSpan(false)
		req := gate.RouteRequest{SchemaVersion: gate.RouteRequestSchemaV1, RequestID: requestID, Graph: graphInput, Route: gate.RouteInput{Source: source, Target: target, Mode: c.Route.Mode, Workers: c.Route.Workers, Seed: uint64(seed), HandoffWorkThreshold: c.Route.HandoffWorkThreshold}, Budget: gate.BudgetInput{TotalWork: c.Budget.WorkLimit, TimeoutMS: durationMillisecondsPointer(c.Budget.SearchTimeLimit)}, Observation: gate.ObservationInput{Mode: gate.ObservationMode(effectiveObservationMode)}, Ablation: c.Ablation}
		runCtx := ctx
		cancel := func() {}
		if timeout > 0 {
			runCtx, cancel = context.WithTimeout(ctx, timeout)
		}
		if traceBaseDir != "" {
			graphSnapshotPath = filepath.Join(traceBaseDir, "graph.json")
			graphSnapshot := struct {
				SchemaVersion string          `json:"schema_version"`
				Graph         gate.GraphInput `json:"graph"`
				Source        uint32          `json:"source"`
				Target        uint32          `json:"target"`
			}{"bridge.graph.snapshot.v1", req.Graph, source, target}
			if err := writeJSONFile(graphSnapshotPath, graphSnapshot, true); err != nil {
				finishRunSpan(true)
				cancel()
				stopProgress(false, completedRuns)
				return out, err
			}
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
		finishRunSpan(err != nil)
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
				RunOrdinal:         runOrdinal,
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
		if graphSnapshotPath != "" {
			raw.References.GraphSnapshotPath = graphSnapshotPath
			if digest, digestErr := fileSHA256(graphSnapshotPath); digestErr == nil {
				raw.References.GraphSnapshotSHA256 = digest
			} else {
				stopProgress(false, completedRuns)
				return out, digestErr
			}
			if relative, relErr := filepath.Rel(executionDirectory, raw.References.GraphSnapshotPath); relErr == nil {
				raw.References.GraphSnapshotPath = filepath.ToSlash(relative)
			}
		}
		if tracePath != "" && collector != nil {
			raw.References.TracePath = tracePath
			raw.References.TraceManifestPath = filepath.Join(traceBaseDir, "manifest.json")
			if relative, relErr := filepath.Rel(executionDirectory, raw.References.TracePath); relErr == nil {
				raw.References.TracePath = filepath.ToSlash(relative)
			}
			if relative, relErr := filepath.Rel(executionDirectory, raw.References.TraceManifestPath); relErr == nil {
				raw.References.TraceManifestPath = filepath.ToSlash(relative)
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
