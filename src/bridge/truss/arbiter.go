package truss

import (
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"math"
)

type Arbiter struct{}

// Choose applies the v0.15 candidate order:
// found/valid candidate -> proof strength -> certified quality ratio ->
// distance -> Work. Path structural validity is enforced at the component
// boundary before candidates reach the Arbiter.
func (Arbiter) Choose(a, b core.RouteResult) core.RouteResult {
	if betterCandidate(b, a) {
		return b
	}
	return a
}

func betterCandidate(a, b core.RouteResult) bool {
	if a.Found != b.Found {
		return a.Found
	}
	if !a.Found {
		return terminationRank(a.TerminationStatus) > terminationRank(b.TerminationStatus)
	}
	if a.Exact != b.Exact {
		return a.Exact
	}
	if a.QualityCertified != b.QualityCertified {
		return a.QualityCertified
	}
	ar, br := certifiedRatio(a), certifiedRatio(b)
	if ar != br {
		return ar < br
	}
	if a.Distance != b.Distance {
		return a.Distance < b.Distance
	}
	if a.TotalWork() != b.TotalWork() {
		return a.TotalWork() < b.TotalWork()
	}
	return a.SolverName < b.SolverName
}

func certifiedRatio(r core.RouteResult) float64 {
	if r.CertifiedRatio == nil || math.IsNaN(*r.CertifiedRatio) || *r.CertifiedRatio < 1 {
		return math.Inf(1)
	}
	return *r.CertifiedRatio
}

func terminationRank(s core.TerminationStatus) int {
	switch s {
	case core.TerminationUnreachable:
		return 3
	case core.TerminationUnknownBudget:
		return 2
	case core.TerminationCancelled, core.TerminationDeadline:
		return 1
	default:
		return 0
	}
}
