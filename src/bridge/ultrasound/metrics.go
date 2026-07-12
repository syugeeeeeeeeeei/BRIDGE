package ultrasound

import (
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"sort"
)

// TraceSummary is a read-only aggregate derived from BEARING events.
type TraceSummary struct {
	EventCount     uint64            `json:"event_count"`
	FirstSequence  uint64            `json:"first_sequence"`
	LastSequence   uint64            `json:"last_sequence"`
	KindCounts     map[string]uint64 `json:"kind_counts"`
	PhaseCounts    map[string]uint64 `json:"phase_counts"`
	MaxLogicalStep uint64            `json:"max_logical_step"`
}

// Summarize derives deterministic counts without changing producer state.
func Summarize(events []bearing.Event) TraceSummary {
	out := TraceSummary{KindCounts: map[string]uint64{}, PhaseCounts: map[string]uint64{}}
	for i, e := range events {
		out.EventCount++
		seq := uint64(i + 1)
		if i == 0 {
			out.FirstSequence = seq
		}
		out.LastSequence = seq
		out.KindCounts[e.Kind]++
		out.PhaseCounts[e.Phase]++
		if e.LogicalStep > out.MaxLogicalStep {
			out.MaxLogicalStep = e.LogicalStep
		}
	}
	return out
}

// Validate checks stable event fields required for formal artifacts.
func Validate(events []bearing.Event) error {
	for i, e := range events {
		if e.Kind == "" {
			return fmt.Errorf("event %d has empty kind", i+1)
		}
		if e.Phase == "" {
			return fmt.Errorf("event %d has empty phase", i+1)
		}
	}
	return nil
}

// SortedKeys returns deterministic map keys for report generation.
func SortedKeys(m map[string]uint64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
