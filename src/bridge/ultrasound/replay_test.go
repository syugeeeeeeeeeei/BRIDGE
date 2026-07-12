package ultrasound

import (
	"testing"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
)

func TestReplayAppliesStateDeltas(t *testing.T) {
	s := ReplayState{ExpandedNodes: map[uint32]bool{}, FrontierNodes: map[uint32]bool{}, EvaluatedEdges: map[string]bool{}, Parents: map[uint32]uint32{}, Distances: map[uint32]float64{}}
	events := []bearing.Event{
		{SchemaVersion: bearing.TraceSchemaVersion, RunID: "r", Sequence: 1, Kind: "frontier_enqueued", Phase: "p", Attributes: map[string]any{"node": uint32(1)}},
		{Sequence: 2, Kind: "node_expanded", Phase: "p", Attributes: map[string]any{"node": uint32(1), "distance": 2.0}},
		{Sequence: 3, Kind: "edge_evaluated", Phase: "p", Attributes: map[string]any{"from": uint32(1), "to": uint32(2)}},
		{Sequence: 4, Kind: "relaxation", Phase: "p", Attributes: map[string]any{"from": uint32(1), "to": uint32(2), "new_distance": 3.0, "accepted": true}},
	}
	for _, e := range events {
		if err := ApplyEvent(&s, e); err != nil {
			t.Fatal(err)
		}
	}
	if !s.ExpandedNodes[1] || s.FrontierNodes[1] || !s.EvaluatedEdges["1>2"] || s.Parents[2] != 1 || s.Distances[2] != 3 {
		t.Fatalf("unexpected replay state: %+v", s)
	}
}
