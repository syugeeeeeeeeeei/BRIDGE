package bolts

import "github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"

type Capability struct {
	Purpose   core.HandoffPurpose
	Local     bool
	Exact     bool
	Resumable bool
}
type CapabilityRegistry struct {
	items map[core.HandoffPurpose]Capability
}

func NewCapabilityRegistry() *CapabilityRegistry {
	r := &CapabilityRegistry{items: map[core.HandoffPurpose]Capability{}}
	for _, c := range []Capability{{core.ConnectCheckpoints, true, true, false}, {core.EscapeRegion, true, false, false}, {core.RepairSegment, true, false, false}, {core.ProveUnreachable, true, true, false}, {core.TightenBound, true, true, false}, {core.CertifyCandidate, true, true, false}} {
		r.items[c.Purpose] = c
	}
	return r
}
func (r *CapabilityRegistry) Get(p core.HandoffPurpose) (Capability, bool) {
	c, ok := r.items[p]
	return c, ok
}
func (r *CapabilityRegistry) All() []Capability {
	out := make([]Capability, 0, len(r.items))
	for _, c := range r.items {
		out = append(out, c)
	}
	return out
}
