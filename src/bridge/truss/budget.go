package truss

import "github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"

type Budget struct {
	limit       *uint64
	used        uint64
	byComponent map[core.Component]uint64
	entries     []core.BudgetLedgerEntry
}

func NewBudget(limit *uint64) *Budget {
	return &Budget{limit: limit, byComponent: map[core.Component]uint64{}}
}
func (b *Budget) Remaining() *uint64 {
	if b.limit == nil {
		return nil
	}
	v := uint64(0)
	if b.used < *b.limit {
		v = *b.limit - b.used
	}
	return &v
}
func (b *Budget) Grant(max *uint64) core.WorkBudget {
	rem := b.Remaining()
	if rem == nil {
		return core.WorkBudget{MaxWork: max}
	}
	v := *rem
	if max != nil && *max < v {
		v = *max
	}
	return core.WorkBudget{MaxWork: &v}
}
func (b *Budget) Consume(c core.Component, n uint64) {
	if b.limit != nil && b.used+n > *b.limit {
		n = *b.limit - b.used
	}
	b.used += n
	b.byComponent[c] += n
}
func (b *Budget) Used() uint64                          { return b.used }
func (b *Budget) ComponentUsed(c core.Component) uint64 { return b.byComponent[c] }

func (b *Budget) Record(taskID string, component core.Component, purpose string, grant core.WorkBudget, used uint64) {
	var granted *uint64
	if grant.MaxWork != nil {
		v := *grant.MaxWork
		granted = &v
	}
	b.entries = append(b.entries, core.BudgetLedgerEntry{TaskID: taskID, Component: component, Purpose: purpose, Granted: granted, Used: used})
}

func (b *Budget) Snapshot() core.BudgetLedger {
	by := map[core.Component]uint64{}
	for k, v := range b.byComponent {
		by[k] = v
	}
	entries := append([]core.BudgetLedgerEntry(nil), b.entries...)
	return core.BudgetLedger{Limit: b.limit, Used: b.used, Remaining: b.Remaining(), ByComponent: by, Entries: entries}
}
