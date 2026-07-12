package core

import "testing"

func TestCompatibilityCountersAreDerivedMirrors(t *testing.T) {
	w := WorkMetrics{RelaxActions: 2, ExpandActions: 3, EnqueueActions: 4, ScheduledSteps: 9}
	r := RouteResult{Work: w, WorkRelaxations: 2, WorkExpandedNodes: 3, QueuePushes: 4, ParallelSteps: 9}
	if !r.CompatibilityCountersValid() {
		t.Fatal("expected derived mirrors to match")
	}
	r.QueuePushes++
	if r.CompatibilityCountersValid() {
		t.Fatal("expected independent mirror mutation to fail")
	}
}
