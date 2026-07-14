package truss

import "sync"

type BudgetManager struct {
	mu                    sync.Mutex
	limit, used, reserved uint64
	unlimited             bool
}

func NewBudgetManager(limit *uint64) *BudgetManager {
	b := &BudgetManager{unlimited: limit == nil}
	if limit != nil {
		b.limit = *limit
	}
	return b
}
func (b *BudgetManager) Reserve(n uint64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.unlimited && b.used+b.reserved+n > b.limit {
		return false
	}
	b.reserved += n
	return true
}
func (b *BudgetManager) Grant(n uint64) uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.unlimited {
		return n
	}
	avail := b.limit - b.used
	if n > avail {
		n = avail
	}
	return n
}
func (b *BudgetManager) Consume(n uint64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.unlimited && b.used+n > b.limit {
		return false
	}
	b.used += n
	if b.reserved >= n {
		b.reserved -= n
	}
	return true
}
func (b *BudgetManager) Used() uint64 { b.mu.Lock(); defer b.mu.Unlock(); return b.used }
func (b *BudgetManager) Remaining() *uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.unlimited {
		return nil
	}
	v := b.limit - b.used
	return &v
}
