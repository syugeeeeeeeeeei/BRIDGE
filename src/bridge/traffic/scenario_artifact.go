package traffic

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/ultrasound"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

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

// ArtifactFileIntegrity records the immutable size and digest of one artifact file.
type ArtifactFileIntegrity struct {
	Path   string `json:"path"`
	Size   int64  `json:"size_bytes"`
	SHA256 string `json:"sha256"`
}

// FinalizeExecutionManifest seals all regular artifact files except manifest.json itself.
// It must be called after healthy.json is written and before the archive is created.
func FinalizeExecutionManifest(directory string) error {
	manifestPath := filepath.Join(directory, "manifest.json")
	var manifest map[string]any
	b, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &manifest); err != nil {
		return err
	}
	entries := make([]ArtifactFileIntegrity, 0, 8)
	err = filepath.Walk(directory, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || path == manifestPath || strings.HasSuffix(strings.ToLower(info.Name()), ".zip") {
			return nil
		}
		rel, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(payload)
		entries = append(entries, ArtifactFileIntegrity{Path: filepath.ToSlash(rel), Size: info.Size(), SHA256: hex.EncodeToString(sum[:])})
		return nil
	})
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	manifest["artifact_files"] = entries
	manifest["integrity_algorithm"] = "sha256"
	return writeJSONFile(manifestPath, manifest, true)
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
