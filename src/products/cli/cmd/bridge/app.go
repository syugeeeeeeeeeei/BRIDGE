package main

import (
	"encoding/json"
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/buildinfo"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

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
	case "serve":
		return serve(args[1:], stdout, stderr)
	case "scenario":
		return scenario(args[1:], stdout, stderr)
	case "benchmark":
		return benchmark(args[1:], stdout, stderr)
	case "artifact":
		return artifact(args[1:], stdout, stderr)
	case "schema":
		return schemaCommand(args[1:], stdout, stderr)
	case "capabilities":
		return capabilities(args[1:], stdout, stderr)
	case "completion":
		return completion(args[1:], stdout, stderr)
	case "version":
		if len(args) > 1 && args[1] == "--output" && len(args) > 2 && args[2] == "json" {
			_ = json.NewEncoder(stdout).Encode(map[string]any{"version": buildinfo.Version, "commit": buildinfo.Commit, "build_time": buildinfo.BuildTime, "dirty": buildinfo.Dirty, "go_version": buildinfo.GoVersion(), "api_versions": []string{"v1"}})
		} else {
			fmt.Fprintln(stdout, "bridge "+buildinfo.Version)
		}
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
  serve                 Start and manage the HTTP service
  scenario              Generate, validate, and inspect scenarios
  benchmark run         Run a benchmark scenario
  benchmark list        List scenario cases
  artifact              Inspect, validate, and evaluate artifacts
  schema                List and show public schemas
  capabilities          Show supported features and contracts
  completion            Generate shell completion
  version               Show version information
  help                  Show help`)
	return code
}
