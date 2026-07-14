package truss

import (
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"sync"
)

// sequencedObserver rebases per-solver Action counters onto one route-global
// sequence so HEALTHY can reconstruct Work across online epochs and handoffs.
type sequencedObserver struct {
	mu    sync.Mutex
	inner bearing.Observer
	step  uint64
}

func newSequencedObserver(inner bearing.Observer) *sequencedObserver {
	if inner == nil {
		inner = bearing.NullObserver{}
	}
	return &sequencedObserver{inner: inner}
}

func (o *sequencedObserver) Observe(e bearing.Event) {
	o.mu.Lock()
	if e.Kind == "action" {
		before := o.step
		o.step++
		e.LogicalStep = o.step
		e.ScheduledStep = o.step
		e.WorkBefore = before
		e.WorkAfter = o.step
	}
	o.inner.Observe(e)
	o.mu.Unlock()
}

func (o *sequencedObserver) Wants(kind string) bool { return bearing.Wants(o.inner, kind) }
