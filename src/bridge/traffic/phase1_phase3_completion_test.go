package traffic

import (
	"context"
	"testing"
)

func oneRunScenario(mode string) BenchmarkScenario {
	s := validScenario()
	s.Execution.Repetitions = 1
	s.Execution.Seeds = []int64{7}
	s.Algorithms = []string{"bridge"}
	s.Observation = ObservationSpec{Mode: mode, SampleRate: 1}
	return s
}

func TestObservationModesPreserveStableDigestAndSummaryIsCollectedWithoutTrace(t *testing.T) {
	modes := []string{"off", "summary", "trace", "profile"}
	var expected string
	for _, mode := range modes {
		result, err := RunScenario(context.Background(), oneRunScenario(mode))
		if err != nil {
			t.Fatalf("%s: %v", mode, err)
		}
		if len(result.RawRuns) != 1 {
			t.Fatalf("%s raw runs=%d", mode, len(result.RawRuns))
		}
		run := result.RawRuns[0]
		if expected == "" {
			expected = run.StableDigest
		}
		if run.StableDigest != expected {
			t.Fatalf("mode %s changed stable digest: %s != %s", mode, run.StableDigest, expected)
		}
		if mode == "off" && run.Observation != nil {
			t.Fatalf("off must not return observation")
		}
		if mode != "off" && (run.Observation == nil || run.Observation.EventCount == 0) {
			t.Fatalf("%s did not collect observation", mode)
		}
	}
}

func TestRawRunsRecomputeQuerySummary(t *testing.T) {
	s := oneRunScenario("off")
	s.Execution.Repetitions = 3
	result, err := RunScenario(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Cases) != 1 {
		t.Fatalf("cases=%d", len(result.Cases))
	}
	values := []float64{}
	for _, run := range result.RawRuns {
		if !run.Warmup {
			values = append(values, float64(run.Work.TotalActions))
		}
	}
	got := summarizeValues(values)
	if got != result.Cases[0].WorkStatistics {
		t.Fatalf("raw recomputation mismatch: got=%+v stored=%+v", got, result.Cases[0].WorkStatistics)
	}
}

func TestExecutionManifestMatchesRawRunOrder(t *testing.T) {
	s := oneRunScenario("off")
	s.Execution.Repetitions = 2
	s.Execution.RandomizeOrder = true
	result, err := RunScenario(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Execution.RunOrder) != len(result.RawRuns) {
		t.Fatalf("manifest=%d raw=%d", len(result.Execution.RunOrder), len(result.RawRuns))
	}
	for i := range result.RawRuns {
		if result.RawRuns[i].RunOrdinal != i+1 || result.Execution.RunOrder[i] != result.RawRuns[i].RunID {
			t.Fatalf("order mismatch at %d", i)
		}
	}
}

