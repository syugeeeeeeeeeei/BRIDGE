package truss

import (
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bolts"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

type SolverRouter struct{ Registry *bolts.CapabilityRegistry }

func (r SolverRouter) Route(p core.HandoffPurpose) (bolts.Capability, error) {
	if r.Registry == nil {
		r.Registry = bolts.NewCapabilityRegistry()
	}
	c, ok := r.Registry.Get(p)
	if !ok {
		return c, fmt.Errorf("unsupported capability %s", p)
	}
	return c, nil
}
