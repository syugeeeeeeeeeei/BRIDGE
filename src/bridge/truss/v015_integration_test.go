package truss

import (
	"context"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"reflect"
	"testing"
)

func TestV015OnlineIntegratedRoute(t *testing.T) {
	g := testGraph()
	r := core.RouteRequest{Source: 0, Target: 3, Mode: core.ModeQuality, Workers: 2, Seed: 17}
	a, err := New(nil).Route(context.Background(), g, r)
	if err != nil {
		t.Fatal(err)
	}
	b, err := New(nil).Route(context.Background(), g, r)
	if err != nil {
		t.Fatal(err)
	}
	if !a.Found || a.TerminationStatus != core.TerminationFound {
		t.Fatalf("route not found: %+v", a)
	}
	if a.Exact || a.QualityCertified {
		t.Fatalf("quality-mode weighted route made unsupported proof claim: %+v", a)
	}
	if a.Telemetry["hypothesis_count"] != 1 {
		t.Fatalf("single adaptive session not used: %+v", a.Telemetry)
	}
	if a.Telemetry["architecture"] != "TRUSS-single-adaptive-session/BOLTS-conditional-handoff" {
		t.Fatalf("unexpected architecture: %+v", a.Telemetry)
	}
	if a.BudgetLedger == nil || a.BudgetLedger.Used != a.Work.TotalActions {
		t.Fatalf("ledger mismatch: %+v", a.BudgetLedger)
	}
	if !reflect.DeepEqual(a.Path, b.Path) || a.Distance != b.Distance || a.TerminationStatus != b.TerminationStatus || a.Work != b.Work {
		t.Fatalf("deterministic result mismatch\na=%+v\nb=%+v", a, b)
	}
}

func TestV015BudgetNeverExceeded(t *testing.T) {
	g := testGraph()
	for _, limit := range []uint64{1, 2, 3, 7, 16, 32, 64} {
		got, err := New(nil).Route(context.Background(), g, core.RouteRequest{Source: 0, Target: 3, Mode: core.ModeBalanced, Workers: 2, WorkBudget: &limit})
		if err != nil {
			t.Fatalf("limit=%d: %v", limit, err)
		}
		if got.TotalWork() > limit {
			t.Fatalf("limit=%d used=%d", limit, got.TotalWork())
		}
	}
}
