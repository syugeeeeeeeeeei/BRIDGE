package anchor

import (
	"context"
	"testing"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/ultrasound"
)

func grid(w, h int) *core.AdjacencyGraph {
	g := core.NewAdjacencyGraph(w*h, false)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			n := core.NodeID(y*w + x)
			_ = g.SetPosition(n, core.Point{X: float64(x), Y: float64(y)})
			if x+1 < w {
				_ = g.AddEdge(n, n+1, 1)
			}
			if y+1 < h {
				_ = g.AddEdge(n, n+core.NodeID(w), 1)
			}
		}
	}
	return g
}
func TestAnalyzeGeometricGrid(t *testing.T) {
	g := grid(10, 10)
	got := Analyze(g)
	if got != "geometric_corridor" {
		t.Fatalf("strategy=%s", got)
	}
}
func TestAnchorHardBudgetAndValidPath(t *testing.T) {
	g := grid(20, 20)
	b := uint64(180)
	r := core.RouteRequest{Source: 0, Target: 399, Mode: core.ModeBalanced, Workers: 1, AnchorStrategy: "geometric_corridor"}
	res := Solver{}.Solve(context.Background(), g, r, core.WorkBudget{MaxWork: &b}, bearing.NullObserver{})
	if res.TotalWork() > b {
		t.Fatalf("work %d > %d", res.TotalWork(), b)
	}
	if !res.Work.Valid() {
		t.Fatalf("invalid work metrics: %+v", res.Work)
	}
	if res.Found {
		if d := core.PathDistance(g, res.Path); d != res.Distance {
			t.Fatalf("distance %f != %f", d, res.Distance)
		}
	}
}
func TestObserverNonInterference(t *testing.T) {
	g := grid(8, 8)
	b := uint64(64)
	r := core.RouteRequest{Source: 0, Target: 63, Mode: core.ModeBalanced, Workers: 1, AnchorStrategy: "geometric_corridor"}
	a := Solver{}.Solve(context.Background(), g, r, core.WorkBudget{MaxWork: &b}, bearing.NullObserver{})
	c := &countObserver{}
	z := Solver{}.Solve(context.Background(), g, r, core.WorkBudget{MaxWork: &b}, c)
	if a.Found != z.Found || a.Distance != z.Distance || a.WorkExpandedNodes != z.WorkExpandedNodes {
		t.Fatalf("observer changed result: %#v %#v", a, z)
	}
	if c.n == 0 {
		t.Fatal("expected events")
	}
}

type countObserver struct{ n int }

func (c *countObserver) Observe(bearing.Event) { c.n++ }

func TestObservationModesDoNotChangeStableOutcome(t *testing.T) {
	g := grid(10, 10)
	b := uint64(300)
	r := core.RouteRequest{Source: 0, Target: 99, Mode: core.ModeBalanced, Workers: 1, AnchorStrategy: "geometric_corridor"}
	base := Solver{}.Solve(context.Background(), g, r, core.WorkBudget{MaxWork: &b}, bearing.NullObserver{})
	for _, mode := range []string{"summary", "trace", "profile"} {
		c := ultrasound.NewCollector(mode, &ultrasound.MemorySink{})
		got := Solver{}.Solve(context.Background(), g, r, core.WorkBudget{MaxWork: &b}, c)
		if base.Found != got.Found || base.Distance != got.Distance || base.Work != got.Work || len(base.Path) != len(got.Path) {
			t.Fatalf("mode %s changed outcome: base=%+v got=%+v", mode, base, got)
		}
		for i := range base.Path {
			if base.Path[i] != got.Path[i] {
				t.Fatalf("mode %s changed path", mode)
			}
		}
	}
}
