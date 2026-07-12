//go:build legacy
// +build legacy

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/ultrasound"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "record":
		record(os.Args[2:])
	case "replay":
		replay(os.Args[2:])
	case "validate":
		validate(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}
func usage() { fmt.Fprintln(os.Stderr, "usage: bridge-ultrasound record|replay|validate [options]") }

func record(args []string) {
	fs := flag.NewFlagSet("record", flag.ExitOnError)
	width := fs.Int("width", 70, "grid width")
	height := fs.Int("height", 70, "grid height")
	seed := fs.Uint64("seed", 1, "deterministic seed")
	mode := fs.String("route-mode", "balanced", "fast|balanced|quality|exact")
	traceMode := fs.String("ultrasound-mode", "trace", "summary|metrics|trace|debug")
	output := fs.String("output", "ultrasound-runs", "run directory root")
	runID := fs.String("run-id", "", "stable run identifier")
	workBudget := fs.Uint64("work-budget", 0, "optional total work budget")
	_ = fs.Parse(args)
	g, err := traffic.Grid(*width, *height, int64(*seed))
	must(err)
	req := core.RouteRequest{Source: 0, Target: core.NodeID((*width)*(*height) - 1), Mode: core.RouteMode(*mode), Workers: 1, Seed: *seed}
	if *workBudget > 0 {
		req.WorkBudget = workBudget
	}
	cfg := ultrasound.RunConfig{OutputDir: *output, RunID: *runID, Mode: ultrasound.Mode(*traceMode), Command: os.Args, Topology: "grid", NodeCount: g.NodeCount(), EdgeCount: g.EdgeCount(), Source: req.Source, Target: req.Target, Seed: *seed, RouteMode: req.Mode}
	recorder, err := ultrasound.NewRecorder(cfg)
	must(err)
	result, routeErr := gate.New(recorder).Route(context.Background(), g, req)
	closeErr := recorder.Close(result)
	must(routeErr)
	must(closeErr)
	fmt.Println(recorder.Dir())
}

func replay(args []string) {
	fs := flag.NewFlagSet("replay", flag.ExitOnError)
	input := fs.String("input", "", "events.jsonl or run directory")
	output := fs.String("output", "", "optional replay-state JSON")
	at := fs.Uint64("sequence", 0, "reserved for future partial replay; 0 replays all")
	_ = fs.Parse(args)
	_ = at
	path := eventPath(*input)
	state, err := ultrasound.ReplayFile(path)
	must(err)
	b, err := json.MarshalIndent(state, "", "  ")
	must(err)
	if *output == "" {
		fmt.Println(string(b))
		return
	}
	must(os.WriteFile(*output, append(b, '\n'), 0o644))
}

func validate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	input := fs.String("input", "", "events.jsonl or run directory")
	_ = fs.Parse(args)
	state, err := ultrasound.ReplayFile(eventPath(*input))
	must(err)
	fmt.Printf("valid schema=%s run_id=%s sequence=%d expanded=%d frontier=%d edges=%d work=%d\n", state.SchemaVersion, state.RunID, state.LastSequence, len(state.ExpandedNodes), len(state.FrontierNodes), len(state.EvaluatedEdges), state.Work)
}
func eventPath(input string) string {
	if input == "" {
		must(fmt.Errorf("--input is required"))
	}
	if strings.HasSuffix(strings.ToLower(input), ".jsonl") {
		return input
	}
	return filepath.Join(input, "events.jsonl")
}
func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var _ = strconv.IntSize
