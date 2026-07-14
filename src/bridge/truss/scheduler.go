package truss

import (
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"sort"
	"sync"
)

type ScheduledTask struct {
	ID         string
	ParentID   string
	Epoch      uint64
	Priority   int
	Capability core.HandoffPurpose
	Runnable   bool
}
type Scheduler interface {
	Enqueue(ScheduledTask)
	Next(epoch uint64, limit int) []ScheduledTask
}
type FairScheduler struct {
	mu sync.Mutex
	q  []ScheduledTask
}

func (s *FairScheduler) Enqueue(t ScheduledTask) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.q = append(s.q, t)
}
func (s *FairScheduler) Next(epoch uint64, limit int) []ScheduledTask {
	s.mu.Lock()
	defer s.mu.Unlock()
	sort.SliceStable(s.q, func(i, j int) bool {
		if s.q[i].Priority != s.q[j].Priority {
			return s.q[i].Priority > s.q[j].Priority
		}
		return s.q[i].ID < s.q[j].ID
	})
	if limit <= 0 || limit > len(s.q) {
		limit = len(s.q)
	}
	out := append([]ScheduledTask{}, s.q[:limit]...)
	s.q = append([]ScheduledTask{}, s.q[limit:]...)
	return out
}
