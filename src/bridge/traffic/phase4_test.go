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
		Execution:     ExecutionSpec{Repetitions: 1, Seeds: []int64{1}, Jobs: 1},
		Algorithms:    []string{"bridge"},
		Observation:   ObservationSpec{Mode: "summary", SampleRate: 1},
		Scenarios:     []ScenarioCase{{ID: "budget", Graph: GeneratorSpec{Generator: "grid", Nodes: 25, Topology: "open"}, Endpoints: EndpointSpec{Strategy: "generator_default_endpoints"}, Route: RouteSpec{Mode: core.ModeQuality, Workers: 1}, Budget: BudgetSpec{TotalWork: &budget}, Ablation: AblationSpec{DisableFallback: true, DisableCertification: true}}},
	}
	got, err := RunScenario(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.RawRuns) != 1 {
		t.Fatalf("raw runs=%d", len(got.RawRuns))
	}
	r := got.RawRuns[0]
	if !r.Ablation.DisableFallback || !r.Ablation.DisableCertification {
		t.Fatalf("ablation not preserved: %+v", r.Ablation)
	}
	if !r.Found && r.FailureReason == "" {
		t.Fatal("missing failure reason")
	}
	if len(got.Cases) != 1 {
		t.Fatalf("cases=%d", len(got.Cases))
	}
	if _, ok := got.Cases[0].MetricStatistics["improvement_count"]; !ok {
		t.Fatal("missing improvement statistics")
	}
}

