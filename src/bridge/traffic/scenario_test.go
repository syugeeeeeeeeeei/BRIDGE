package traffic

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunScenario(t *testing.T) {
	s := BenchmarkScenario{SchemaVersion: BenchmarkSchemaV1, Suite: SuiteSpec{ID: "test"}, Execution: ExecutionSpec{Repetitions: 1, Seeds: []int64{1}}, Algorithms: []string{"bridge", "anchor", "dijkstra"}, Scenarios: []ScenarioCase{{ID: "grid", Graph: GeneratorSpec{Generator: "grid", Nodes: 16, Topology: "open"}, Queries: []QuerySpec{{ID: "default", Selection: QuerySelectionSpec{Method: "generator_default"}}}}}}
	r, err := RunScenario(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.ScenarioSummaries) != 3 || r.ScenarioSummaries[0].Runs != 1 || r.ScenarioSummaries[0].FoundRate != 1 {
		t.Fatalf("unexpected result: %+v", r)
	}
	paths := map[string]string{}
	for _, c := range r.ScenarioSummaries {
		paths[c.Algorithm] = c.ExecutionPath
		if c.AverageEndToEndMS < c.AverageSolverTimeMS {
			t.Fatalf("timing order must hold: %+v", c)
		}
	}
	if paths["bridge"] != "route" || paths["anchor"] != "execute_once" || paths["dijkstra"] != "execute_once" {
		t.Fatalf("unexpected execution paths: %+v", paths)
	}
}

func TestRunScenarioRandomGeometric(t *testing.T) {
	s := BenchmarkScenario{
		SchemaVersion: BenchmarkSchemaV1,
		Suite:         SuiteSpec{ID: "random-geometric"},
		Execution:     ExecutionSpec{Repetitions: 1, Seeds: []int64{1}},
		Algorithms:    []string{"bridge", "astar"},
		Scenarios: []ScenarioCase{{
			ID:      "geo",
			Graph:   GeneratorSpec{Generator: "random_geometric", Nodes: 40, K: 6},
			Queries: []QuerySpec{{ID: "default", Selection: QuerySelectionSpec{Method: "generator_default"}}},
		}},
	}
	r, err := RunScenario(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.ScenarioSummaries) != 2 {
		t.Fatalf("unexpected result: %+v", r)
	}
}

func TestRunScenarioReportsProgress(t *testing.T) {
	s := BenchmarkScenario{
		SchemaVersion: BenchmarkSchemaV1,
		Suite:         SuiteSpec{ID: "progress"},
		Execution:     ExecutionSpec{Repetitions: 1, Seeds: []int64{1}},
		Algorithms:    []string{"bridge", "astar"},
		Scenarios: []ScenarioCase{{
			ID:      "grid",
			Graph:   GeneratorSpec{Generator: "grid", Nodes: 16, Topology: "open"},
			Queries: []QuerySpec{{ID: "default", Selection: QuerySelectionSpec{Method: "generator_default"}}},
		}},
	}
	var lines []string
	_, err := RunScenarioWithOptions(context.Background(), s, RunScenarioOptions{
		ProgressReporter: ProgressReporterFunc(func(progress RunProgress) {
			lines = append(lines, progress.RunName)
			if progress.Total != 2 {
				t.Fatalf("unexpected progress: %+v", progress)
			}
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) < 2 {
		t.Fatalf("expected progress events, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "grid/") {
		t.Fatalf("unexpected run name: %q", lines[0])
	}
}

func TestRunScenarioWritesOutputArtifacts(t *testing.T) {
	dir := t.TempDir()
	s := BenchmarkScenario{
		SchemaVersion: BenchmarkSchemaV1,
		Suite:         SuiteSpec{ID: "artifacts"},
		Execution:     ExecutionSpec{Repetitions: 1, Seeds: []int64{1}},
		Algorithms:    []string{"bridge"},
		Observation:   ObservationSpec{Mode: "minimum"},
		Output:        OutputSpec{Directory: dir},
		Scenarios: []ScenarioCase{{
			ID:      "grid",
			Graph:   GeneratorSpec{Generator: "grid", Nodes: 16, Topology: "open"},
			Queries: []QuerySpec{{ID: "default", Selection: QuerySelectionSpec{Method: "generator_default"}}},
		}},
	}
	r, err := RunScenarioWithOptions(context.Background(), s, RunScenarioOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(r.OutputDirectory, "result.json")); err != nil {
		t.Fatal(err)
	}
	if len(r.Runs) != 1 {
		t.Fatalf("runs=%d", len(r.Runs))
	}

}
func TestScenarioRejectsUnsupportedAlgorithm(t *testing.T) {
	s := BenchmarkScenario{SchemaVersion: BenchmarkSchemaV1, Suite: SuiteSpec{ID: "x"}, Execution: ExecutionSpec{Repetitions: 1}, Algorithms: []string{"astar"}, Scenarios: []ScenarioCase{{ID: "x", Graph: GeneratorSpec{Generator: "grid", Nodes: 4, Topology: "open"}, Queries: []QuerySpec{{ID: "default", Selection: QuerySelectionSpec{Method: "generator_default"}}}}}}
	s.ApplyDefaults()
	if err := s.Validate(); err != nil {
		t.Fatalf("astar should now be supported: %v", err)
	}
}

func TestScenarioRejectsUnknownAlgorithm(t *testing.T) {
	s := BenchmarkScenario{SchemaVersion: BenchmarkSchemaV1, Suite: SuiteSpec{ID: "x"}, Execution: ExecutionSpec{Repetitions: 1}, Algorithms: []string{"mystery"}, Scenarios: []ScenarioCase{{ID: "x", Graph: GeneratorSpec{Generator: "grid", Nodes: 4, Topology: "open"}, Queries: []QuerySpec{{ID: "default", Selection: QuerySelectionSpec{Method: "generator_default"}}}}}}
	s.ApplyDefaults()
	if s.Validate() == nil {
		t.Fatal("expected validation error")
	}
}

func TestScenarioRejectsGraphSnapshotOutputWithoutDir(t *testing.T) {
	s := validScenario()
	s.Output.Directory = ""
	if err := s.Validate(); err == nil {
		t.Fatal("expected error")
	}
}
