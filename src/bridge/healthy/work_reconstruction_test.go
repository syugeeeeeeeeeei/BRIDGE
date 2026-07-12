package healthy

import (
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"testing"
)

func TestReconstructWork(t *testing.T) {
	events := []bearing.Event{{Kind: "action", Action: "expand", LogicalStep: 1, ScheduledStep: 1}, {Kind: "action", Action: "evaluate", LogicalStep: 1, ScheduledStep: 2}}
	r := ReconstructWork(events, 1, false, 0)
	if !r.Verifiable || r.Work.TotalActions != 2 || r.Work.ExpandActions != 1 || r.Work.EvaluateActions != 1 {
		t.Fatalf("%+v", r)
	}
}
func TestReconstructWorkRejectsIncompleteTrace(t *testing.T) {
	r := ReconstructWork(nil, .5, false, 0)
	if r.Verifiable {
		t.Fatal("sampled trace must not be verifiable")
	}
}
