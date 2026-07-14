package healthy

import (
	"context"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
	"testing"
)

func TestValidatePathAndWork(t *testing.T) {
	g := core.NewAdjacencyGraph(3, false)
	_ = g.AddEdge(0, 1, 2)
	_ = g.AddEdge(1, 2, 3)
	d := 5.0
	r := traffic.BenchmarkRun{
		QueryProfile:    traffic.QueryProfile{Source: 0, Target: 2},
		ExecutionResult: traffic.ExecutionResult{PathFound: true, Path: []core.NodeID{0, 1, 2}, PathCost: &d},
		Measurement:     traffic.Measurement{Work: core.WorkMetrics{TotalActions: 1, ExpandActions: 1, ScheduledSteps: 1, LogicalSteps: 1}},
	}
	v := ValidatePath(g, r, ValidationPolicy{DistanceAbsoluteTolerance: 1e-9, DistanceRelativeTolerance: 1e-9})
	if !v.PathValid {
		t.Fatalf("%+v", v)
	}
	if !ValidateWork(r.Measurement.Work, nil).StructuralValid {
		t.Fatal("work should be valid")
	}
}
func TestAnalyzeBenchmark(t *testing.T) {
	s := traffic.BenchmarkScenario{SchemaVersion: traffic.BenchmarkSchemaV1, Suite: traffic.SuiteSpec{ID: "h"}, Execution: traffic.ExecutionSpec{Repetitions: 1, Seeds: []int64{1}}, Algorithms: []string{"dijkstra", "anchor"}, Observation: traffic.ObservationSpec{Mode: "minimum"}, Scenarios: []traffic.ScenarioCase{{ID: "g", Graph: traffic.GeneratorSpec{Generator: "grid", Nodes: 9, Topology: "open"}, Queries: []traffic.QuerySpec{{ID: "q", Selection: traffic.QuerySelectionSpec{Method: "generator_default"}}}, Route: traffic.RouteSpec{Mode: core.ModeBalanced, Workers: 1}}}}
	a, err := traffic.RunScenario(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	p := DefaultProfile("anchor", "dijkstra")
	h, err := Analyze(context.Background(), a, p)
	if err != nil {
		t.Fatal(err)
	}
	if h.Summary.Runs != 2 {
		t.Fatalf("%+v", h.Summary)
	}
}
