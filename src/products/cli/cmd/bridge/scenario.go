package main

import (
	"flag"
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/products/cli/templates"
	"io"
	"os"
)

func scenario(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "error: scenario subcommand is required")
		return exitUsage
	}
	switch args[0] {
	case "init":
		return scenarioInit(args[1:], stdout, stderr)
	case "validate":
		return benchmarkValidate(args[1:], stdout, stderr)
	case "inspect":
		return scenarioInspect(args[1:], stdout, stderr)
	case "list-presets":
		for _, p := range templates.Presets() {
			fmt.Fprintln(stdout, p)
		}
		return 0
	case "help", "--help", "-h":
		fmt.Fprintln(stdout, "Usage:\n  bridge scenario init [--preset name]\n  bridge scenario validate <file>\n  bridge scenario inspect <file>\n  bridge scenario list-presets")
		return 0
	default:
		fmt.Fprintf(stderr, "error: unknown scenario subcommand %q\n", args[0])
		return exitUsage
	}
}
func scenarioInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("scenario init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	preset := fs.String("preset", "minimal", "scenario preset")
	out := fs.String("output", "scenario.yaml", "output path; use - for stdout")
	fs.StringVar(out, "o", "scenario.yaml", "output path")
	levelText := fs.String("comments", "full", "full, summary, or none")
	overwrite := fs.Bool("overwrite", false, "overwrite existing file")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	level, err := templates.ParseLevel(*levelText)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}
	text, err := templates.Scenario(*preset, level)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}
	tmp, err := os.CreateTemp("", "bridge-scenario-*.yaml")
	if err != nil {
		return exitIO
	}
	name := tmp.Name()
	_, _ = tmp.WriteString(text)
	_ = tmp.Close()
	_, err = loadScenario(name)
	_ = os.Remove(name)
	if err != nil {
		fmt.Fprintln(stderr, "error: generated template is invalid:", err)
		return exitInternal
	}
	return writeTextOutput(stdout, stderr, *out, *overwrite, text)
}
func scenarioInspect(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "error: exactly one scenario file is required")
		return exitUsage
	}
	s, err := loadScenario(args[0])
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}
	s.ApplyDefaults()
	if err = s.Validate(); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}
	runs := len(s.Scenarios) * len(s.Algorithms) * s.Execution.Repetitions * len(s.Execution.Seeds)
	fmt.Fprintf(stdout, "Scenario: %s\nCases: %d\nAlgorithms: %d\nSeeds: %d\nRepetitions: %d\nEstimated executions: %d\nObservation: %s\n", s.Suite.ID, len(s.Scenarios), len(s.Algorithms), len(s.Execution.Seeds), s.Execution.Repetitions, runs, s.Observation.Mode)
	return 0
}
