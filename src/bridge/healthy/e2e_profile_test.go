package healthy

import (
	"context"
	"testing"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
)

func TestTraceWorkValidationEndToEnd(t *testing.T) {
	dir := t.TempDir()
	scenario := traffic.BenchmarkScenario{
		SchemaVersion: traffic.BenchmarkSchemaV1,
		Suite:         traffic.SuiteSpec{ID: "healthy-profile-e2e"},
		Execution:     traffic.ExecutionSpec{Repetitions: 1, Seeds: []int64{7}},
		Algorithms:    []string{"bridge", "anchor", "dijkstra", "bidirectional_dijkstra", "astar", "weighted_astar", "reachability"},
		Observation:   traffic.ObservationSpec{Mode: "trace"},
		Output:        traffic.OutputSpec{Directory: dir},
		Scenarios:     []traffic.ScenarioCase{{ID: "line", Graph: traffic.GeneratorSpec{Generator: "grid", Width: 4, Height: 1, Topology: "open"}, Queries: []traffic.QuerySpec{{ID: "q", Selection: traffic.QuerySelectionSpec{Method: "explicit", Source: u32(0), Target: u32(3)}}}, Route: traffic.RouteSpec{Mode: core.ModeBalanced, Workers: 1}, Budget: traffic.BudgetSpec{WorkLimit: u64(10000)}}},
	}
	artifact, err := traffic.RunScenarioWithOptions(context.Background(), scenario, traffic.RunScenarioOptions{})
	if err != nil {
		t.Fatal(err)
	}
	profile := DefaultProfile("bridge", "dijkstra")
	profile.Validation.RequireWorkTrace = true
	profile.Validation.RequireBudgetLedger = true
	result, err := Analyze(context.Background(), artifact, profile)
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.InvalidRuns != 0 {
		t.Fatalf("unexpected invalid runs: %+v", result.RunValidations)
	}
	for _, rv := range result.RunValidations {
		if !rv.Work.TraceVerifiable || !rv.Work.TraceValid {
			t.Fatalf("%s trace work invalid: %+v", rv.RunID, rv.Work)
		}
		if rv.Algorithm == "bridge" && (!rv.Work.LedgerVerifiable || !rv.Work.LedgerValid) {
			t.Fatalf("bridge ledger invalid: %+v", rv.Work)
		}
	}
}
func u32(v uint32) *uint32 { return &v }
func u64(v uint64) *uint64 { return &v }
