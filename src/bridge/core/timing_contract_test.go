package core

import "testing"

func TestTimeBreakdownNanosecondsArePrimary(t *testing.T) {
	b := TimeBreakdown{TotalNS: 1250, SolverNS: 1000, TrussNS: 1100, GateNS: 1250}
	if b.TotalNS <= 0 || b.SolverNS <= 0 {
		t.Fatal("primary nanosecond durations must be positive")
	}
	if b.GateNS < b.TrussNS || b.TrussNS < b.SolverNS {
		t.Fatalf("invalid timing boundary order: %+v", b)
	}
}
