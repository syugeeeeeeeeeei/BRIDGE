package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/ultrasound"
	"io"
	"os"
)

func route(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("route", flag.ContinueOnError)
	fs.SetOutput(stderr)
	requestPath := fs.String("input", "", "JSON request file; use - for stdin")
	fs.StringVar(requestPath, "i", "", "JSON request file; use - for stdin")
	outputPath := fs.String("output", "", "result output file")
	fs.StringVar(outputPath, "o", "", "result output file")
	overwrite := fs.Bool("overwrite", false, "overwrite output")
	traceOutput := fs.String("trace-output", "", "JSONL trace output file")
	traceOverwrite := fs.Bool("trace-overwrite", false, "overwrite trace output")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "error: at most one input file is allowed")
		return exitUsage
	}
	if fs.NArg() == 1 {
		if *requestPath != "" {
			fmt.Fprintln(stderr, "error: input was specified twice")
			return exitUsage
		}
		*requestPath = fs.Arg(0)
	}
	if *requestPath == "-" {
		*requestPath = ""
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
		if req.Observation.Mode == "" || req.Observation.Mode == gate.ObservationMinimum {
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
