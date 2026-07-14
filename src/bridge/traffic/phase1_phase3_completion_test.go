package traffic

import (
	"context"
	"testing"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
)

func oneRunScenario(mode string) BenchmarkScenario {
	s := validScenario()
	s.Execution.Repetitions = 1
	s.Execution.Seeds = []int64{7}
	s.Algorithms = []string{"bridge"}
	s.Observation = ObservationSpec{Mode: mode}
	return s
}

func TestObservationModesPreserveStableDigestAndSummaryIsCollectedWithoutTrace(t *testing.T) {
	modes := []string{"minimum", "debug", "trace"}
	var expected string
	for _, mode := range modes {
		result, err := RunScenario(context.Background(), oneRunScenario(mode))
		if err != nil {
			t.Fatalf("%s: %v", mode, err)
		}
		if len(result.Runs) != 1 {
			t.Fatalf("%s runs=%d", mode, len(result.Runs))
		}
		run := result.Runs[0]
		if expected == "" {
			expected = run.RunMetadata.StableDigest
		}
		if run.RunMetadata.StableDigest != expected {
			t.Fatalf("mode %s changed stable digest: %s != %s", mode, run.RunMetadata.StableDigest, expected)
		}
		if mode == "minimum" && run.Observations.ObservationData != nil {
			t.Fatalf("off must not return observation")
		}
		observation, _ := run.Observations.ObservationData.(*gate.ObservationResult)
		if mode != "minimum" && (observation == nil || observation.EventCount == 0) {
			t.Fatalf("%s did not collect observation", mode)
		}
	}
}

func TestRawRunsRecomputeQuerySummary(t *testing.T) {
	s := oneRunScenario("minimum")
	s.Execution.Repetitions = 3
	result, err := RunScenario(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.ScenarioSummaries) != 1 {
		t.Fatalf("summaries=%d", len(result.ScenarioSummaries))
	}
	values := []float64{}
	for _, run := range result.Runs {
		if !run.RunMetadata.WarmupRun {
			values = append(values, float64(run.Measurement.Work.TotalActions))
		}
	}
	got := summarizeValues(values)
	if got != result.ScenarioSummaries[0].WorkStatistics {
		t.Fatalf("raw recomputation mismatch: got=%+v stored=%+v", got, result.ScenarioSummaries[0].WorkStatistics)
	}
}

func TestExecutionManifestMatchesRawRunOrder(t *testing.T) {
	s := oneRunScenario("minimum")
	s.Execution.Repetitions = 2
	s.Execution.RandomizeOrder = true
	result, err := RunScenario(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Execution.RunOrder) != len(result.Runs) {
		t.Fatalf("manifest=%d runs=%d", len(result.Execution.RunOrder), len(result.Runs))
	}
	for i := range result.Runs {
		if result.Runs[i].RunMetadata.RunOrdinal != i+1 || result.Execution.RunOrder[i] != result.Runs[i].RunMetadata.RunID {
			t.Fatalf("order mismatch at %d", i)
		}
	}
}
