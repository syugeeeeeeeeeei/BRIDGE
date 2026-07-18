package ultrasound

import (
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"testing"
)

func TestCollectorAggregatesLifecycleSpan(t *testing.T) {
	c := NewCollector("minimum", nil)
	span, finish := bearing.BeginLifecycle(c, "run-1", "task-1", "", "GATE", "graph_build")
	if span == "" {
		t.Fatal("lifecycle span was not created")
	}
	finish(false)
	m := c.Metrics()
	if len(m.Spans.Completed) != 1 {
		t.Fatalf("completed spans=%d", len(m.Spans.Completed))
	}
	got := m.Spans.Completed[0]
	if got.Component != "GATE" || got.Operation != "graph_build" || got.DurationNS < 0 {
		t.Fatalf("unexpected span: %+v", got)
	}
	if m.Spans.Incomplete != 0 || m.Spans.OrphanComplete != 0 || m.Spans.DuplicateStart != 0 {
		t.Fatalf("invalid span summary: %+v", m.Spans)
	}
}

func TestDisabledLifecycleDoesNotEmit(t *testing.T) {
	span, finish := bearing.BeginLifecycle(bearing.NullObserver{}, "", "", "", "GATE", "route")
	if span != "" {
		t.Fatalf("disabled observer created span %q", span)
	}
	finish(false)
}
