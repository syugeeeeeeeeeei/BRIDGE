package traffic

import (
	"context"
	"testing"
)

func TestRawRunContainsPhaseAndSystemMetrics(t *testing.T) {
	s := BenchmarkScenario{SchemaVersion: BenchmarkSchemaV1, Suite: SuiteSpec{ID: "phase2"}, Execution: ExecutionSpec{Repetitions: 1, Seeds: []int64{1}}, Algorithms: []string{"bridge"}, Scenarios: []ScenarioCase{{ID: "g", Graph: GeneratorSpec{Generator: "grid", Nodes: 16}, Endpoints: EndpointSpec{Strategy: "generator_default_endpoints"}, Route: RouteSpec{Workers: 1}}}}
	s.ApplyDefaults()
	got, err := RunScenario(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.RawRuns) != 1 {
		t.Fatalf("raw runs=%d", len(got.RawRuns))
	}
	r := got.RawRuns[0]
	if (r.TimeBreakdown.TotalMS <= 0 && r.EndToEndTimeMS <= 0 && r.TimeBreakdown.GateMS <= 0) || r.TimeBreakdown.SolverMS < 0 {
		t.Fatalf("invalid timing: %+v", r.TimeBreakdown)
	}
	if r.SystemMetrics.MallocCount == 0 {
		t.Fatalf("expected runtime allocation metrics: %+v", r.SystemMetrics)
	}
	if r.SystemMetrics.HeapAllocBoundaryMax < r.SystemMetrics.HeapAllocBefore || r.SystemMetrics.HeapAllocBoundaryMax < r.SystemMetrics.HeapAllocAfter {
		t.Fatalf("invalid peak: %+v", r.SystemMetrics)
	}
}

