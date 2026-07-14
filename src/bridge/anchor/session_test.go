package anchor

import (
	"context"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"testing"
)

func TestSessionSnapshotResumeEquivalence(t *testing.T) {
	g := grid(12, 12)
	r := core.RouteRequest{Source: 0, Target: 143, Mode: core.ModeBalanced, Workers: 1}
	a, _ := NewSession(g, r, bearing.NullObserver{})
	for !a.Finished() {
		a.Step(context.Background(), 37)
	}
	want := a.Result()
	b, _ := NewSession(g, r, bearing.NullObserver{})
	b.Step(context.Background(), 111)
	snap := b.Snapshot()
	c, _ := Resume(g, snap, bearing.NullObserver{})
	for !c.Finished() {
		c.Step(context.Background(), 37)
	}
	got := c.Result()
	if want.Found != got.Found || want.Distance != got.Distance || want.Work != got.Work || len(want.Path) != len(got.Path) {
		t.Fatalf("resume mismatch want=%+v got=%+v", want, got)
	}
	for i := range want.Path {
		if want.Path[i] != got.Path[i] {
			t.Fatal("path mismatch")
		}
	}
}
func TestStepNeverExceedsGrant(t *testing.T) {
	g := grid(6, 6)
	r := core.RouteRequest{Source: 0, Target: 35, Mode: core.ModeBalanced, Workers: 1}
	s, _ := NewSession(g, r, bearing.NullObserver{})
	for i := 0; i < 100 && !s.Finished(); i++ {
		x := s.Step(context.Background(), 7)
		if x.Consumed > 7 {
			t.Fatalf("consumed=%d", x.Consumed)
		}
	}
}
