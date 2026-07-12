package traffic

import (
	"context"
	"testing"
)

func TestResearchScenarioExpandsQueriesWarmupsAndRawRuns(t *testing.T) {
	source0, target0 := uint32(0), uint32(8)
	source1, target1 := uint32(1), uint32(7)
	s := BenchmarkScenario{
		SchemaVersion: BenchmarkSchemaV1,
		Suite:         SuiteSpec{ID: "research-contract"},
		Execution:     ExecutionSpec{Repetitions: 2, WarmupRuns: 1, Seeds: []int64{11, 12}, Jobs: 1, RandomizeOrder: true},
		Algorithms:    []string{"dijkstra"},
		Observation:   ObservationSpec{Mode: "off"},
		Scenarios:     []ScenarioCase{{ID: "grid", Graph: GeneratorSpec{Generator: "grid", Width: 3, Height: 3}, Queries: []QuerySpec{{ID: "q0", Strategy: "explicit", Source: &source0, Target: &target0}, {ID: "q1", Strategy: "explicit", Source: &source1, Target: &target1}}}},
	}
	result, err := RunScenario(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(result.RawRuns), 12; got != want {
		t.Fatalf("raw runs=%d want=%d", got, want)
	}
	warmups := 0
	ids := map[string]bool{}
	for _, run := range result.RawRuns {
		if run.Warmup {
			warmups++
		}
		if run.RunID == "" || run.GraphInstanceID == "" || run.QueryID == "" {
			t.Fatalf("incomplete run identity: %+v", run)
		}
		if ids[run.RunID] {
			t.Fatalf("duplicate run id %q", run.RunID)
		}
		ids[run.RunID] = true
	}
	if warmups != 4 {
		t.Fatalf("warmups=%d want=4", warmups)
	}
	if len(result.Cases) != 2 || result.Cases[0].Runs != 4 || result.Cases[1].Runs != 4 {
		t.Fatalf("cases=%+v", result.Cases)
	}
	if result.Cases[0].WorkStatistics.Count != 4 || result.Cases[1].WorkStatistics.Count != 4 {
		t.Fatalf("stats=%+v", result.Cases[0].WorkStatistics)
	}
}

func TestSummaryStatisticsAreRecomputable(t *testing.T) {
	got := summarizeValues([]float64{1, 2, 3, 4})
	if got.Count != 4 || got.Mean != 2.5 || got.P50 != 2.5 || got.Min != 1 || got.Max != 4 {
		t.Fatalf("stats=%+v", got)
	}
	if got.CI95Lower >= got.Mean || got.CI95Upper <= got.Mean {
		t.Fatalf("invalid confidence interval: %+v", got)
	}
}

func TestScenarioRejectsLegacyObservationModes(t *testing.T) {
	for _, mode := range []string{"metrics", "debug"} {
		s := validScenario()
		s.Observation.Mode = mode
		if err := s.Validate(); err == nil {
			t.Fatalf("mode %q should be rejected", mode)
		}
	}
	for _, mode := range []string{"off", "summary", "trace", "profile"} {
		s := validScenario()
		s.Observation.Mode = mode
		if err := s.Validate(); err != nil {
			t.Fatalf("mode %q: %v", mode, err)
		}
	}
}
