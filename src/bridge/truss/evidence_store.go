package truss

import (
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"sort"
	"sync"
)

type EvidenceStore struct {
	mu    sync.RWMutex
	items map[string]core.Evidence
}

func NewEvidenceStore() *EvidenceStore { return &EvidenceStore{items: map[string]core.Evidence{}} }
func (s *EvidenceStore) Put(e core.Evidence) error {
	if err := e.Validate(); err != nil {
		return err
	}
	if e.Proof == core.ProofEmpirical && (e.ID == "exact" || e.ID == "unreachable") {
		return fmt.Errorf("empirical evidence cannot be promoted")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[e.ID] = e
	return nil
}
func (s *EvidenceStore) Get(id string) (core.Evidence, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.items[id]
	return e, ok
}

// MergeEpoch validates and commits evidence in a deterministic order. The
// operation is all-or-nothing: invalid evidence leaves the store unchanged.
func (s *EvidenceStore) MergeEpoch(items []core.Evidence) error {
	ordered := append([]core.Evidence(nil), items...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].HypothesisID != ordered[j].HypothesisID {
			return ordered[i].HypothesisID < ordered[j].HypothesisID
		}
		if ordered[i].Solver != ordered[j].Solver {
			return ordered[i].Solver < ordered[j].Solver
		}
		return ordered[i].ID < ordered[j].ID
	})
	for _, e := range ordered {
		if err := e.Validate(); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range ordered {
		s.items[e.ID] = e
	}
	return nil
}

func (s *EvidenceStore) Snapshot() []core.Evidence {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]core.Evidence, 0, len(s.items))
	for _, e := range s.items {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
