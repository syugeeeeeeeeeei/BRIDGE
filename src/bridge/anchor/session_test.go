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

func TestWeightedSessionDoesNotClaimOptimalityOnFrontierExhaustion(t *testing.T) {
	g := grid(5, 5)
	r := core.RouteRequest{Source: 0, Target: 24, Mode: core.ModeBalanced, Workers: 1}
	s, err := NewHypothesisSession(g, r, bearing.NullObserver{}, "weighted", "test", 1.8)
	if err != nil {
		t.Fatal(err)
	}
	for !s.Finished() {
		s.Step(context.Background(), 1024)
	}
	got := s.Result()
	if !got.Found || !got.SearchCompleted {
		t.Fatalf("weighted session did not complete: %+v", got)
	}
	if got.Exact || got.QualityCertified || got.CertifiedRatio != nil {
		t.Fatalf("weighted session made unsupported proof claim: %+v", got)
	}
}

func TestAdmissibleSessionMayClaimOptimalityOnFrontierExhaustion(t *testing.T) {
	g := grid(5, 5)
	r := core.RouteRequest{Source: 0, Target: 24, Mode: core.ModeExact, Workers: 1}
	s, err := NewHypothesisSession(g, r, bearing.NullObserver{}, "exact", "test", 1.0)
	if err != nil {
		t.Fatal(err)
	}
	for !s.Finished() {
		s.Step(context.Background(), 1024)
	}
	got := s.Result()
	if !got.Exact || !got.QualityCertified || got.CertifiedRatio == nil || *got.CertifiedRatio != 1 {
		t.Fatalf("admissible completed session did not retain exact proof: %+v", got)
	}
}

type allocationProbeObserver struct{ enabled bool }

func (o allocationProbeObserver) Observe(bearing.Event) {}
func (o allocationProbeObserver) Wants(kind string) bool {
	return o.enabled && kind == "state_delta"
}

func TestDisabledObservationAvoidsStateDeltaAllocation(t *testing.T) {
	g := grid(24, 24)
	r := core.RouteRequest{Source: 0, Target: 575, Mode: core.ModeBalanced, Workers: 1}
	run := func(o bearing.Observer) {
		s, err := NewSession(g, r, o)
		if err != nil {
			t.Fatal(err)
		}
		for !s.Finished() {
			s.Step(context.Background(), 1<<20)
		}
		if !s.Result().Found {
			t.Fatal("route not found")
		}
	}
	disabled := testing.AllocsPerRun(5, func() { run(allocationProbeObserver{enabled: false}) })
	enabled := testing.AllocsPerRun(5, func() { run(allocationProbeObserver{enabled: true}) })
	if disabled*2 >= enabled {
		t.Fatalf("disabled observation still pays state-delta allocation cost: disabled=%0.0f enabled=%0.0f", disabled, enabled)
	}
}
