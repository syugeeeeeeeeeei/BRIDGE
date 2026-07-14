package truss

import (
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"testing"
)

func TestArbiterV015Ranking(t *testing.T) {
	one, two := 1.0, 1.2
	exact := core.RouteResult{Found: true, Exact: true, QualityCertified: true, CertifiedRatio: &one, Distance: 10, SolverName: "exact"}
	shorterUnproven := core.RouteResult{Found: true, Distance: 9, SolverName: "unproven"}
	if got := (Arbiter{}).Choose(shorterUnproven, exact); got.SolverName != "exact" {
		t.Fatalf("proof must outrank raw distance: %+v", got)
	}
	certified12 := core.RouteResult{Found: true, QualityCertified: true, CertifiedRatio: &two, Distance: 8, SolverName: "ratio12"}
	certified10 := core.RouteResult{Found: true, QualityCertified: true, CertifiedRatio: &one, Distance: 9, SolverName: "ratio10"}
	if got := (Arbiter{}).Choose(certified12, certified10); got.SolverName != "ratio10" {
		t.Fatalf("quality ratio must outrank distance: %+v", got)
	}
	lowWork := core.RouteResult{Found: true, Distance: 10, SolverName: "a", Work: core.WorkMetrics{TotalActions: 2, SelectActions: 2}}
	highWork := core.RouteResult{Found: true, Distance: 10, SolverName: "b", Work: core.WorkMetrics{TotalActions: 3, SelectActions: 3}}
	if got := (Arbiter{}).Choose(highWork, lowWork); got.SolverName != "a" {
		t.Fatalf("work tie-break failed: %+v", got)
	}
}
