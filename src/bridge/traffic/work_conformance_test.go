package traffic

import (
	"context"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"testing"
)

func TestAllRegisteredAlgorithmsProduceConservedWork(t *testing.T) {
	algorithms := []string{"bridge", "anchor", "dijkstra", "bidirectional_dijkstra", "astar", "weighted_astar", "reachability"}
	for _, a := range algorithms {
		t.Run(a, func(t *testing.T) {
			s := BenchmarkScenario{SchemaVersion: BenchmarkSchemaV1, Suite: SuiteSpec{ID: "work-" + a}, Execution: ExecutionSpec{Repetitions: 1, Seeds: []int64{7}, Jobs: 1}, Algorithms: []string{a}, Observation: ObservationSpec{Mode: "off", SampleRate: 1}, Scenarios: []ScenarioCase{{ID: "line", Graph: GeneratorSpec{Generator: "grid", Nodes: 9, Topology: "open"}, Queries: []QuerySpec{{ID: "q", Strategy: "opposite-corners"}}, Route: RouteSpec{Mode: core.ModeBalanced, Workers: 1}}}}
			r, err := RunScenario(context.Background(), s)
			if err != nil {
				t.Fatal(err)
			}
			if len(r.RawRuns) != 1 {
				t.Fatalf("runs=%d", len(r.RawRuns))
			}
			w := r.RawRuns[0].Work
			if !w.Valid() {
				t.Fatalf("invalid Work: %+v errors=%v", w, w.ValidationErrors())
			}
		})
	}
}
func TestUnsupportedAblationsAreRejected(t *testing.T) {
	s := BenchmarkScenario{SchemaVersion: BenchmarkSchemaV1, Suite: SuiteSpec{ID: "x"}, Execution: ExecutionSpec{Repetitions: 1, Seeds: []int64{1}, Jobs: 1}, Algorithms: []string{"bridge"}, Observation: ObservationSpec{Mode: "off", SampleRate: 1}, Scenarios: []ScenarioCase{{ID: "x", Graph: GeneratorSpec{Generator: "grid", Nodes: 4, Topology: "open"}, Ablation: AblationSpec{DisableStateReuse: true}}}}
	if err := s.Validate(); err == nil {
		t.Fatal("unsupported ablation must be rejected")
	}
}
