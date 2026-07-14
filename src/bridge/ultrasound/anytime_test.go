package ultrasound

import (
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"testing"
)

func TestAnytimeAndReuse(t *testing.T) {
	ev := []bearing.Event{{Kind: "bound", WorkAfter: 5, Attributes: map[string]any{"upper_bound": 10.0}}, {Kind: "bound", WorkAfter: 7, Attributes: map[string]any{"lower_bound": 5.0}}, {Kind: "action", Attributes: map[string]any{"reused": true}}, {Kind: "action", Attributes: map[string]any{"duplicate": true}}}
	c := AnytimeCurve(ev)
	if len(c) != 2 || c[1].Ratio == nil || *c[1].Ratio != 2 {
		t.Fatalf("bad curve %+v", c)
	}
	m := ComputeReuse(ev)
	if m.StateReuseRatio != .5 || m.DuplicateWorkRatio != .5 {
		t.Fatalf("bad reuse %+v", m)
	}
}
