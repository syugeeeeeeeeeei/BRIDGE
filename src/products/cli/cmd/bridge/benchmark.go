package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/healthy"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/ultrasound"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/internal/yamlmini"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func benchmark(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "error: benchmark subcommand is required")
		return exitUsage
	}
	switch args[0] {
	case "run":
		return benchmarkRun(args[1:], stdout, stderr)
	case "list":
		return benchmarkList(args[1:], stdout, stderr)
	case "help", "--help", "-h":
		printBenchmarkHelp(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "error: unknown benchmark subcommand %q\n", args[0])
		return exitUsage
	}
}
func printBenchmarkHelp(w io.Writer) {
	fmt.Fprintln(w, `Usage:
  bridge benchmark run <scenario.yaml|json>
  bridge benchmark list <scenario.yaml|json>`)
}
func benchmarkRun(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "error: exactly one scenario file is required")
		return exitUsage
	}
	scenarioPath := args[0]
	s, err := loadScenario(scenarioPath)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}
	s.ApplyDefaults()
	if err := s.Validate(); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}
	progress := newRewritingProgressReporter(stderr)
	result, err := traffic.RunScenarioWithOptions(context.Background(), s, traffic.RunScenarioOptions{
		ScenarioPath:     scenarioPath,
		ProgressReporter: progress,
	})
	progress.Finish()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return exitTimeout
		}
		return exitInternal
	}
	processCollector := ultrasound.NewCollector("minimum", nil)
	processRunID := result.ExecutionID
	processSpan, finishProcess := bearing.BeginLifecycle(processCollector, processRunID, "benchmark-process", "", "TRAFFIC", "benchmark_post_processing")
	profile := healthy.DefaultProfile("bridge", "dijkstra")
	_, finishHealthy := bearing.BeginLifecycle(processCollector, processRunID, "benchmark-process", processSpan, "TRAFFIC", "healthy_evaluation")
	healthResult, healthErr := healthy.Analyze(context.Background(), result, profile)
	finishHealthy(healthErr != nil)
	if healthErr != nil {
		fmt.Fprintln(stderr, "error: HEALTHY:", healthErr)
		return exitInternal
	}
	healthPath := filepath.Join(result.OutputDirectory, "healthy.json")
	healthFile, healthErr := os.OpenFile(healthPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if healthErr != nil {
		fmt.Fprintln(stderr, "error:", healthErr)
		return exitIO
	}
	_, finishHealthEncoding := bearing.BeginLifecycle(processCollector, processRunID, "benchmark-process", processSpan, "TRAFFIC", "healthy_json_encoding")
	healthEncoder := json.NewEncoder(healthFile)
	healthEncoder.SetIndent("", "  ")
	healthErr = healthEncoder.Encode(healthResult)
	closeErr := healthFile.Close()
	finishHealthEncoding(healthErr != nil || closeErr != nil)
	if healthErr != nil {
		fmt.Fprintln(stderr, "error:", healthErr)
		return exitIO
	}
	if closeErr != nil {
		fmt.Fprintln(stderr, "error:", closeErr)
		return exitIO
	}
	_, finishIntegrity := bearing.BeginLifecycle(processCollector, processRunID, "benchmark-process", processSpan, "TRAFFIC", "artifact_integrity")
	if err := traffic.FinalizeExecutionManifest(result.OutputDirectory); err != nil {
		finishIntegrity(true)
		finishProcess(true)
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	finishIntegrity(false)
	_, finishArchive := bearing.BeginLifecycle(processCollector, processRunID, "benchmark-process", processSpan, "TRAFFIC", "zip_compression")
	if err := writeBenchmarkArchive(result.OutputDirectory, result.ExecutionID); err != nil {
		finishArchive(true)
		finishProcess(true)
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	finishArchive(false)
	finishProcess(false)
	_ = processCollector.Close(context.Background())
	processMetricsPath := filepath.Join(result.OutputDirectory, result.ExecutionID+".operation-metrics.json")
	if err := writeJSONMetricFile(processMetricsPath, processCollector.Metrics()); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	archivePath := filepath.Join(result.OutputDirectory, result.ExecutionID+".zip")
	fmt.Fprintln(stderr, "artifacts:", result.OutputDirectory)
	fmt.Fprintln(stderr, "archive:", archivePath)
	if err := writeBenchmark(stdout, "console", result); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}
	return 0
}

func writeJSONMetricFile(path string, value any) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	encodeErr := enc.Encode(value)
	closeErr := f.Close()
	if encodeErr != nil {
		return encodeErr
	}
	return closeErr
}

func splitScenarioArgs(args []string) (string, []string, error) {
	var scenario string
	flagArgs := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			flagArgs = append(flagArgs, a)
			if (a == "--format" || a == "-format" || a == "--output" || a == "-output" || a == "--trace-dir" || a == "-trace-dir") && i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
			continue
		}
		if scenario != "" {
			return "", nil, errors.New("exactly one scenario file is required")
		}
		scenario = a
	}
	if scenario == "" {
		return "", nil, errors.New("exactly one scenario file is required")
	}
	return scenario, flagArgs, nil
}

func benchmarkValidate(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "error: exactly one scenario file is required")
		return exitUsage
	}
	s, err := loadScenario(args[0])
	if err == nil {
		s.ApplyDefaults()
		err = s.Validate()
	}
	if err != nil {
		fmt.Fprintln(stderr, "invalid:", err)
		return exitUsage
	}
	fmt.Fprintf(stdout, "valid: %s (%d scenarios)\n", s.Suite.ID, len(s.Scenarios))
	return 0
}
func benchmarkList(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "error: exactly one scenario file is required")
		return exitUsage
	}
	s, err := loadScenario(args[0])
	if err == nil {
		s.ApplyDefaults()
		err = s.Validate()
	}
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}
	for _, c := range s.Scenarios {
		fmt.Fprintf(stdout, "%s\t%s\t%d requested_nodes\t%s\n", c.ID, c.Graph.Generator, c.Graph.Nodes, c.Queries[0].Selection.Method)
	}
	return 0
}

func loadScenario(path string) (traffic.BenchmarkScenario, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return traffic.BenchmarkScenario{}, err
	}
	payload := b
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".yaml" || ext == ".yml" {
		payload, err = yamlmini.ToJSON(b)
		if err != nil {
			return traffic.BenchmarkScenario{}, err
		}
	}
	var s traffic.BenchmarkScenario
	if err := decodeStrict(payload, &s); err != nil {
		return s, err
	}
	return s, nil
}
func decodeStrict(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("multiple JSON values are not allowed")
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return errors.New("multiple JSON values are not allowed")
	}
	return nil
}
func writeBenchmark(w io.Writer, format string, r traffic.BenchmarkResult) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	case "jsonl":
		enc := json.NewEncoder(w)
		for _, c := range r.ScenarioSummaries {
			row := struct {
				SchemaVersion string `json:"schema_version"`
				SuiteID       string `json:"suite_id"`
				traffic.ScenarioSummary
			}{r.SchemaVersion, r.SuiteID, c}
			if err := enc.Encode(row); err != nil {
				return err
			}
		}
		return nil
	case "csv":
		cw := csv.NewWriter(w)
		defer cw.Flush()
		if err := cw.Write([]string{"suite_id", "scenario_id", "algorithm", "target_kind", "execution_path", "runs", "path_found_rate", "optimality_proven_rate", "mean_path_cost", "mean_work_actions", "mean_solver_time_ms", "mean_end_to_end_time_ms"}); err != nil {
			return err
		}
		for _, c := range r.ScenarioSummaries {
			if err := cw.Write([]string{r.SuiteID, c.ScenarioID, c.Algorithm, c.TargetKind, c.ExecutionPath, strconv.Itoa(c.Runs), fmt.Sprintf("%.6f", c.FoundRate), fmt.Sprintf("%.6f", c.ExactRate), fmt.Sprintf("%.9g", c.AverageDistance), fmt.Sprintf("%.3f", c.AverageWork), fmt.Sprintf("%.3f", c.AverageSolverTimeMS), fmt.Sprintf("%.3f", c.AverageEndToEndMS)}); err != nil {
				return err
			}
		}
		return cw.Error()
	case "console":
		fmt.Fprintf(w, "Suite: %s\nExecution Succeeded: %t\n\n", r.SuiteID, r.RunMetadata.ExecutionSucceeded)
		fmt.Fprintln(w, "SCENARIO\tALGORITHM\tPATH\tRUNS\tPATH FOUND\tOPTIMALITY\tMEAN WORK\tSOLVER MS\tEND-TO-END MS")
		for _, c := range r.ScenarioSummaries {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%.3f\t%.3f\t%.1f\t%.3f\t%.3f\n", c.ScenarioID, c.Algorithm, c.ExecutionPath, c.Runs, c.FoundRate, c.ExactRate, c.AverageWork, c.AverageSolverTimeMS, c.AverageEndToEndMS)
		}
		for _, f := range r.Failures {
			fmt.Fprintln(w, "FAIL:", f)
		}
		return nil
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

func writeBenchmarkArchive(outputDir, artifactID string) error {
	if outputDir == "" {
		return nil
	}
	archiveBase := artifactID
	if archiveBase == "" {
		archiveBase = filepath.Base(filepath.Clean(outputDir))
	}
	if archiveBase == "" || archiveBase == "." || archiveBase == string(os.PathSeparator) {
		return fmt.Errorf("could not determine archive name for output_dir %q", outputDir)
	}
	archivePath := filepath.Join(outputDir, archiveBase+".zip")
	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	defer writer.Close()

	return filepath.Walk(outputDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == outputDir {
			return nil
		}
		if path == archivePath {
			return nil
		}
		rel, err := filepath.Rel(outputDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if info.IsDir() {
			_, err := writer.Create(rel + "/")
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = rel
		header.Method = zip.Deflate
		entry, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(entry, src)
		closeErr := src.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}
func outputWriter(stdout io.Writer, path string, overwrite bool) (io.Writer, func(), error) {
	if path == "" {
		return stdout, func() {}, nil
	}
	flags := os.O_WRONLY | os.O_CREATE
	if overwrite {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	f, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return nil, func() {}, err
	}
	return f, func() { _ = f.Close() }, nil
}
func readInput(stdin io.Reader, path string) ([]byte, error) {
	if path != "" {
		if f, ok := stdin.(*os.File); ok {
			if st, _ := f.Stat(); st != nil && (st.Mode()&os.ModeCharDevice) == 0 {
				return nil, errors.New("both stdin and an input file were specified")
			}
		}
		return os.ReadFile(path)
	}
	if f, ok := stdin.(*os.File); ok {
		if st, _ := f.Stat(); st != nil && (st.Mode()&os.ModeCharDevice) != 0 {
			return nil, errors.New("route request is required as a positional input file or stdin")
		}
	}
	b, err := io.ReadAll(io.LimitReader(stdin, 16<<20))
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return nil, errors.New("route request is empty")
	}
	return b, nil
}

func evaluateBenchmarkResult(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "check" {
		fmt.Fprintln(stderr, "error: expected health check")
		return exitUsage
	}
	fs := flag.NewFlagSet("health check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	profilePath := fs.String("profile", "", "health profile JSON file")
	output := fs.String("output", "", "health result output file")
	overwrite := fs.Bool("overwrite", false, "overwrite output")
	if err := fs.Parse(args[1:]); err != nil {
		return exitUsage
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "error: benchmark result path is required")
		return exitUsage
	}
	b, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	var artifact traffic.BenchmarkResult
	if err := json.Unmarshal(b, &artifact); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}
	var profile healthy.HealthProfile
	if *profilePath != "" {
		pb, e := os.ReadFile(*profilePath)
		if e != nil {
			fmt.Fprintln(stderr, "error:", e)
			return exitIO
		}
		if e = json.Unmarshal(pb, &profile); e != nil {
			fmt.Fprintln(stderr, "error:", e)
			return exitUsage
		}
	} else {
		profile = healthy.DefaultProfile("bridge", "dijkstra")
	}
	result, err := healthy.Analyze(context.Background(), artifact, profile)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}
	out, closeFn, err := outputWriter(stdout, *output, *overwrite)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	defer closeFn()
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	if result.RegressionEvaluation.Status == healthy.StatusFail || result.RegressionEvaluation.Status == healthy.StatusInvalid {
		return exitAcceptance
	}
	return 0
}
