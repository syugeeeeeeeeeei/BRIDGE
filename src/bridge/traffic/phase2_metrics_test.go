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
	if len(got.Runs) != 1 {
		t.Fatalf("runs=%d", len(got.Runs))
	}
	r := got.Runs[0]
	if r.Measurement.TimeBreakdown.SolverMS < 0 || r.Measurement.EndToEndTimeMS < 0 || r.Measurement.TimeBreakdown.GateMS < 0 {
		t.Fatalf("invalid timing: %+v", r.Measurement.TimeBreakdown)
	}
	if r.Measurement.ZeroDuration && (r.Measurement.EndToEndTimeMS != 0 || r.Measurement.SolverTimeMS != 0) {
		t.Fatalf("zero_duration flag mismatch: %+v", r.Measurement)
	}
	if r.Measurement.SystemMetrics.MallocCount == 0 {
		t.Fatalf("expected runtime allocation metrics: %+v", r.Measurement.SystemMetrics)
	}
	if r.Measurement.SystemMetrics.HeapAllocBoundaryMax < r.Measurement.SystemMetrics.HeapAllocBefore || r.Measurement.SystemMetrics.HeapAllocBoundaryMax < r.Measurement.SystemMetrics.HeapAllocAfter {
		t.Fatalf("invalid peak: %+v", r.Measurement.SystemMetrics)
	}
}
