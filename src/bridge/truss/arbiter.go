package truss

import "github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"

type Arbiter struct{}

func (Arbiter) Choose(a, b core.RouteResult) core.RouteResult {
	if !a.Found {
		return b
	}
	if !b.Found {
		return a
	}
	if b.Distance < a.Distance {
		return b
	}
	return a
}
