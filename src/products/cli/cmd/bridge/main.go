package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/healthy"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/ultrasound"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/internal/yamlmini"
)

const version = "0.14.3"
const (
	exitUsage      = 2
	exitIO         = 3
	exitTimeout    = 4
	exitAcceptance = 5
	exitInternal   = 10
)

type rewritingProgressReporter struct {
	mu         sync.Mutex
	w          io.Writer
	lastWidth  int
	hadUpdates bool
}

func newRewritingProgressReporter(w io.Writer) *rewritingProgressReporter {
	return &rewritingProgressReporter{w: w}
}

func (r *rewritingProgressReporter) ReportRunProgress(progress traffic.RunProgress) {
	if r == nil || r.w == nil {
		return
	}
	line := fmt.Sprintf(
		"[%d/%d] run=%s run_time=%s elapsed=%s eta=%s",
		progress.Current,
		progress.Total,
		progress.RunName,
		formatCLIProgressDuration(progress.RunElapsed),
		formatCLIProgressDuration(progress.TotalElapsed),
		formatCLIProgressDuration(progress.ETA),
	)
	r.mu.Lock()
	defer r.mu.Unlock()
	padding := ""
	if len(line) < r.lastWidth {
		padding = strings.Repeat(" ", r.lastWidth-len(line))
	}
	_, _ = fmt.Fprintf(r.w, "\r%s%s", line, padding)
	r.lastWidth = len(line)
	r.hadUpdates = true
	if progress.Finished {
		_, _ = fmt.Fprint(r.w, "\n")
		r.lastWidth = 0
	}
}

func (r *rewritingProgressReporter) Finish() {
	if r == nil || r.w == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.hadUpdates && r.lastWidth > 0 {
		_, _ = fmt.Fprint(r.w, "\n")
		r.lastWidth = 0
	}
}

func formatCLIProgressDuration(d time.Duration) string {
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

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return printHelp(stderr, exitUsage)
	}
	switch args[0] {
	case "route":
		return route(args[1:], stdin, stdout, stderr)
	case "benchmark":
		return benchmark(args[1:], stdout, stderr)
	case "health":
		return health(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintln(stdout, version)
		return 0
	case "help", "--help", "-h":
		return printHelp(stdout, 0)
	default:
		fmt.Fprintf(stderr, "error: unknown command %q\n", args[0])
		return printHelp(stderr, exitUsage)
	}
}
func printHelp(w io.Writer, code int) int {
	fmt.Fprintln(w, `BRIDGE - budget-managed anytime route search

Usage:
  bridge <command> [options]

Commands:
  route                 Process one route request
  benchmark             Run benchmark scenarios
  benchmark validate    Validate a scenario file
  benchmark list        List scenario cases
  health check          Validate and compare a benchmark artifact
  version               Show version information
  help                  Show help

Use "bridge benchmark validate <scenario>" or "bridge benchmark list <scenario>" for benchmark utilities.`)
	return code
}
func route(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("route", flag.ContinueOnError)
	fs.SetOutput(stderr)
	requestPath := fs.String("request", "", "JSON request file")
	outputPath := fs.String("output", "", "result output file")
	overwrite := fs.Bool("overwrite", false, "overwrite output")
	traceOutput := fs.String("trace-output", "", "JSONL trace output file")
	traceOverwrite := fs.Bool("trace-overwrite", false, "overwrite trace output")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "error: unexpected positional arguments")
		return exitUsage
	}
	data, err := readInput(stdin, *requestPath)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	var req gate.RouteRequest
	if err := gate.DecodeStrictJSON(data, &req); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}
	if err := resolveGraphFile(&req); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	var collector *ultrasound.Collector
	if *traceOutput != "" {
		if req.Observation.Mode == "" || req.Observation.Mode == gate.ObservationOff {
			req.Observation.Mode = gate.ObservationTrace
		}
		fsink, sinkErr := ultrasound.NewFileSink(*traceOutput, *traceOverwrite)
		if sinkErr != nil {
			fmt.Fprintln(stderr, "error:", sinkErr)
			return exitIO
		}
		collector = ultrasound.NewCollector(string(req.Observation.Mode), fsink)
	}
	obs := gate.ObservationOptions{Mode: req.Observation.Mode}
	if collector != nil {
		obs.Observer = collector
		obs.Reporter = collector
	}
	result, err := gate.NewRouter().Route(context.Background(), req, gate.RouteOptions{Observation: obs})
	var observationErr error
	if collector != nil {
		observationErr = collector.Close(context.Background())
	}
	if err != nil {
		var pe *gate.PublicError
		if errors.As(err, &pe) {
			fmt.Fprintf(stderr, "error: %s: %s\n", pe.Code, pe.Message)
			return exitUsage
		}
		fmt.Fprintln(stderr, "error:", err)
		return exitInternal
	}
	out, closeFn, err := outputWriter(stdout, *outputPath, *overwrite)
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
	if observationErr != nil {
		fmt.Fprintln(stderr, "warning: observation failed:", observationErr)
	}
	return 0
}

func resolveGraphFile(req *gate.RouteRequest) error {
	if req.Graph.Type != "file" {
		return nil
	}
	if req.Graph.Path == "" {
		return errors.New("graph.path is required for file graph")
	}
	b, err := os.ReadFile(req.Graph.Path)
	if err != nil {
		return err
	}
	var graph gate.GraphInput
	if err := gate.DecodeStrictJSON(b, &graph); err != nil {
		return fmt.Errorf("decode graph file: %w", err)
	}
	if graph.Type != "inline" {
		return errors.New("graph file must contain graph.type=inline")
	}
	req.Graph = graph
	return nil
}

func benchmark(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "error: benchmark command is required")
		return exitUsage
	}
	switch args[0] {
	case "validate":
		return benchmarkValidate(args[1:], stdout, stderr)
	case "list":
		return benchmarkList(args[1:], stdout, stderr)
	case "help", "--help", "-h":
		printBenchmarkHelp(stdout)
		return 0
	default:
		return benchmarkRun(args, stdout, stderr)
	}
}
func printBenchmarkHelp(w io.Writer) {
	fmt.Fprintln(w, `Usage:
  bridge benchmark <scenario.yaml|json>
  bridge benchmark validate <scenario.yaml|json>
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
		Overwrite:        true,
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
	if err := writeBenchmark(stdout, "console", result); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}
	if !result.Passed {
		return exitAcceptance
	}
	return 0
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
		fmt.Fprintf(stdout, "%s\t%s\t%d requested_nodes\t%s\n", c.ID, c.Graph.Generator, c.Graph.Nodes, c.Endpoints.Strategy)
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
		for _, c := range r.Cases {
			row := struct {
				SchemaVersion string `json:"schema_version"`
				SuiteID       string `json:"suite_id"`
				Passed        bool   `json:"passed"`
				traffic.CaseResult
			}{r.SchemaVersion, r.SuiteID, r.Passed, c}
			if err := enc.Encode(row); err != nil {
				return err
			}
		}
		return nil
	case "csv":
		cw := csv.NewWriter(w)
		defer cw.Flush()
		if err := cw.Write([]string{"suite_id", "scenario_id", "algorithm", "target_kind", "execution_path", "runs", "path_found_rate", "optimality_proven_rate", "mean_path_cost", "mean_work_actions", "mean_solver_time_ms", "mean_end_to_end_time_ms", "passed"}); err != nil {
			return err
		}
		for _, c := range r.Cases {
			if err := cw.Write([]string{r.SuiteID, c.ScenarioID, c.Algorithm, c.TargetKind, c.ExecutionPath, strconv.Itoa(c.Runs), fmt.Sprintf("%.6f", c.FoundRate), fmt.Sprintf("%.6f", c.ExactRate), fmt.Sprintf("%.9g", c.AverageDistance), fmt.Sprintf("%.3f", c.AverageWork), fmt.Sprintf("%.3f", c.AverageSolverTimeMS), fmt.Sprintf("%.3f", c.AverageEndToEndMS), strconv.FormatBool(r.Passed)}); err != nil {
				return err
			}
		}
		return cw.Error()
	case "console":
		fmt.Fprintf(w, "Suite: %s\nPassed: %t\n\n", r.SuiteID, r.Passed)
		fmt.Fprintln(w, "SCENARIO\tALGORITHM\tPATH\tRUNS\tPATH FOUND\tOPTIMALITY\tMEAN WORK\tSOLVER MS\tEND-TO-END MS")
		for _, c := range r.Cases {
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
				return nil, errors.New("both stdin and --request were specified")
			}
		}
		return os.ReadFile(path)
	}
	if f, ok := stdin.(*os.File); ok {
		if st, _ := f.Stat(); st != nil && (st.Mode()&os.ModeCharDevice) != 0 {
			return nil, errors.New("route request is required on stdin or --request")
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

func health(args []string, stdout, stderr io.Writer) int {
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
