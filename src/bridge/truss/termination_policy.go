package truss

import (
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"math"
)

type TerminationPolicy struct{ MaxSuboptimality float64 }

func (p TerminationPolicy) Decide(found bool, upper, lower float64, complete, budget, cancelled bool) core.TerminationStatus {
	if cancelled {
		return core.TerminationCancelled
	}
	if complete && !found {
		return core.TerminationUnreachable
	}
	if found {
		q := p.MaxSuboptimality
		if q < 1 {
			q = 1
		}
		if lower > 0 && !math.IsInf(lower, 0) && upper/lower <= q {
			return core.TerminationFound
		}
		if complete {
			return core.TerminationFound
		}
	}
	if budget {
		return core.TerminationUnknownBudget
	}
	return core.TerminationRunning
}
