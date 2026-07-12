package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
)

func main() {
	repetitions := flag.Int("repetitions", 5, "identical executions used to verify deterministic results")
	includeTiming := flag.Bool("include-timing", false, "include non-deterministic wall-clock timing in CSV")
	seed := flag.Int64("seed", 1, "graph generation seed")
	flag.Parse()

	wr := csv.NewWriter(os.Stdout)
	defer wr.Flush()
	header := []string{"nodes", "edges", "seed", "mode", "found", "distance", "total_work", "work_expanded_nodes", "scheduled_steps", "result_sha256", "repeatability_runs"}
	if *includeTiming {
		header = append(header, "time_ms")
	}
	mustWrite(wr, header)

	runner := gate.New(nil)
	for _, n := range []int{20, 50, 100, 500, 1000, 2000, 5000, 10000} {
		graph, err := traffic.Grid(n, 1, *seed)
		if err != nil {
			fatal(err)
		}
		for _, mode := range []core.RouteMode{core.ModeBalanced, core.ModeExact} {
			request := core.RouteRequest{Source: 0, Target: core.NodeID(n - 1), Mode: mode, Workers: 1, Seed: uint64(*seed)}
			result, digest, err := traffic.VerifyRepeatability(context.Background(), runner, graph, request, *repetitions)
			if err != nil {
				fatal(fmt.Errorf("nodes=%d mode=%s: %w", n, mode, err))
			}
			row := []string{
				strconv.Itoa(graph.NodeCount()), strconv.Itoa(graph.EdgeCount()), strconv.FormatInt(*seed, 10), string(mode),
				strconv.FormatBool(result.Found), strconv.FormatFloat(result.Distance, 'g', -1, 64),
				strconv.FormatUint(result.TotalWork(), 10), strconv.FormatUint(result.WorkExpandedNodes, 10), strconv.FormatUint(result.Work.ScheduledSteps, 10), digest, strconv.Itoa(max(2, *repetitions)),
			}
			if *includeTiming {
				row = append(row, strconv.FormatFloat(result.TimeMS, 'f', 3, 64))
			}
			mustWrite(wr, row)
		}
	}
	if err := wr.Error(); err != nil {
		fatal(err)
	}
}

func mustWrite(writer *csv.Writer, row []string) {
	if err := writer.Write(row); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
