package ultrasound

import (
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"testing"
)

func TestSummarizeDeterministic(t *testing.T) {
	events := []bearing.Event{{Kind: "search_started", Phase: "x"}, {Kind: "node_expanded", Phase: "x", LogicalStep: 3}, {Kind: "search_finished", Phase: "x", LogicalStep: 5}}
	if err := Validate(events); err != nil {
		t.Fatal(err)
	}
	a, b := Summarize(events), Summarize(events)
	if a.EventCount != 3 || a.MaxLogicalStep != 5 || a.KindCounts["node_expanded"] != 1 {
		t.Fatalf("unexpected summary: %+v", a)
	}
	if a.LastSequence != b.LastSequence {
		t.Fatal("non deterministic")
	}
}
