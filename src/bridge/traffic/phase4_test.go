package traffic

import (
	"context"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"testing"
)

func TestPhase4AblationAndFailureContracts(t *testing.T) {
	budget := uint64(1)
	s := BenchmarkScenario{
		SchemaVersion: BenchmarkSchemaV1,
		Suite:         SuiteSpec{ID: "phase4"},
		Execution:     ExecutionSpec{Repetitions: 1, Seeds: []int64{1}},
		Algorithms:    []string{"bridge"},
		Observation:   ObservationSpec{Mode: "debug"},
		Scenarios:     []ScenarioCase{{ID: "budget", Graph: GeneratorSpec{Generator: "grid", Nodes: 25, Topology: "open"}, Queries: []QuerySpec{{ID: "default", Selection: QuerySelectionSpec{Method: "generator_default"}}}, Route: RouteSpec{Mode: core.ModeQuality, Workers: 1}, Budget: BudgetSpec{WorkLimit: &budget}, Ablation: AblationSpec{DisableFallback: true, DisableCertification: true}}},
	}
	got, err := RunScenario(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Runs) != 1 {
		t.Fatalf("runs=%d", len(got.Runs))
	}
	r := got.Runs[0]
	if !r.ScenarioDefinition.AblationConfiguration.DisableFallback || !r.ScenarioDefinition.AblationConfiguration.DisableCertification {
		t.Fatalf("ablation not preserved: %+v", r.ScenarioDefinition.AblationConfiguration)
	}
	if !r.ExecutionResult.PathFound && r.ExecutionResult.FailureReason == "" {
		t.Fatal("missing failure reason")
	}
	if len(got.ScenarioSummaries) != 1 {
		t.Fatalf("summaries=%d", len(got.ScenarioSummaries))
	}
	if _, ok := got.ScenarioSummaries[0].MetricStatistics["improvement_count"]; !ok {
		t.Fatal("missing improvement statistics")
	}
}
