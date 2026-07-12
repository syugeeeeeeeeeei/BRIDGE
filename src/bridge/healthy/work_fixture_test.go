package healthy

import (
	"context"
	"testing"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
)

func lineGraphInput() gate.GraphInput {
	return gate.GraphInput{Type: "inline", Directed: false,
		Nodes: []gate.GraphNode{{ID: 0}, {ID: 1}, {ID: 2}, {ID: 3}},
		Edges: []gate.GraphEdge{{From: 0, To: 1, Weight: 1}, {From: 1, To: 2, Weight: 1}, {From: 2, To: 3, Weight: 1}}}
}

func TestManualLineGraphWorkFixtures(t *testing.T) {
	want := map[string]core.WorkMetrics{
		"anchor":                 {TotalActions: 26, SelectActions: 4, ExpandActions: 4, EvaluateActions: 5, RelaxActions: 5, EnqueueActions: 4, RejectActions: 2, CandidateActions: 1, TerminateActions: 1, LogicalSteps: 26, ScheduledSteps: 26, WorkerCount: 1},
		"dijkstra":               {TotalActions: 25, SelectActions: 4, ExpandActions: 4, EvaluateActions: 5, RelaxActions: 5, EnqueueActions: 4, RejectActions: 2, TerminateActions: 1, LogicalSteps: 25, ScheduledSteps: 25, WorkerCount: 1},
		"bidirectional_dijkstra": {TotalActions: 22, SelectActions: 3, ExpandActions: 3, EvaluateActions: 4, RelaxActions: 4, EnqueueActions: 5, RejectActions: 1, ConnectActions: 1, TerminateActions: 1, LogicalSteps: 22, ScheduledSteps: 22, WorkerCount: 1},
		"astar":                  {TotalActions: 25, SelectActions: 4, ExpandActions: 4, EvaluateActions: 5, RelaxActions: 5, EnqueueActions: 4, RejectActions: 2, TerminateActions: 1, LogicalSteps: 25, ScheduledSteps: 25, WorkerCount: 1},
		"weighted_astar":         {TotalActions: 25, SelectActions: 4, ExpandActions: 4, EvaluateActions: 5, RelaxActions: 5, EnqueueActions: 4, RejectActions: 2, TerminateActions: 1, LogicalSteps: 25, ScheduledSteps: 25, WorkerCount: 1},
		"reachability":           {TotalActions: 20, SelectActions: 4, ExpandActions: 4, EvaluateActions: 5, EnqueueActions: 4, RejectActions: 2, TerminateActions: 1, LogicalSteps: 20, ScheduledSteps: 20, WorkerCount: 1},
	}
	r := gate.NewRouter()

	bridge, err := r.Route(context.Background(), gate.RouteRequest{SchemaVersion: gate.RouteRequestSchemaV1, Graph: lineGraphInput(), Route: gate.RouteInput{Source: 0, Target: 3, Mode: core.ModeBalanced, Workers: 1}, Observation: gate.ObservationInput{Mode: gate.ObservationOff}}, gate.RouteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if bridge.Work != want["anchor"] {
		t.Fatalf("bridge work mismatch: got=%+v want=%+v", bridge.Work, want["anchor"])
	}
	if bridge.BudgetLedger == nil {
		t.Fatal("bridge budget ledger missing")
	}

	for algorithm, expected := range want {
		got, err := r.ExecuteOnce(context.Background(), gate.ExecuteRequest{SchemaVersion: gate.ExecuteRequestSchemaV1, Target: gate.ExecuteTargetInput{ID: algorithm}, Graph: lineGraphInput(), Route: gate.RouteInput{Source: 0, Target: 3, Mode: core.ModeBalanced, Workers: 1}, Observation: gate.ObservationInput{Mode: gate.ObservationOff}}, gate.RouteOptions{})
		if err != nil {
			t.Fatalf("%s: %v", algorithm, err)
		}
		if got.Work != expected {
			t.Fatalf("%s work mismatch\n got=%+v\nwant=%+v", algorithm, got.Work, expected)
		}
	}
}
