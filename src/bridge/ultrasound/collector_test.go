package ultrasound

import (
	"context"
	"errors"
	"testing"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
)

func TestCollectorSequenceAndTruncation(t *testing.T) {
	sink := &MemorySink{}
	c := NewCollectorWithLimit("trace", sink, 2)
	c.Observe(bearing.Event{Phase: "route", Kind: "control"})
	c.Observe(bearing.Event{Phase: "route", Kind: "candidate"})
	c.Observe(bearing.Event{Phase: "route", Kind: "detail"})
	if err := c.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := sink.Events()
	if len(got) != 2 {
		t.Fatalf("events=%d", len(got))
	}
	if got[0].Sequence != 1 || got[1].Sequence != 2 {
		t.Fatalf("sequence=%v,%v", got[0].Sequence, got[1].Sequence)
	}
	m := c.Metrics()
	if !m.Truncated || m.DroppedEvents != 1 {
		t.Fatalf("metrics=%+v", m)
	}
}

func TestMultiSinkClosesAll(t *testing.T) {
	a, b := false, false
	s := MultiSink{Sinks: []EventSink{closeSink{fn: func() { a = true }, err: errors.New("x")}, closeSink{fn: func() { b = true }}}}
	if err := s.Close(context.Background()); err == nil {
		t.Fatal("expected error")
	}
	if !a || !b {
		t.Fatalf("closed a=%t b=%t", a, b)
	}
}

type closeSink struct {
	fn  func()
	err error
}

func (s closeSink) WriteEvent(context.Context, bearing.Event) error { return nil }
func (s closeSink) Close(context.Context) error {
	if s.fn != nil {
		s.fn()
	}
	return s.err
}

func TestObservationModesSeparateAggregateAndTrace(t *testing.T) {
	events := []bearing.Event{
		{Phase: "search", Kind: "search_started"},
		{Phase: "search", Kind: "incumbent_updated", Attributes: map[string]any{"distance": 2.0}},
		{Phase: "search", Kind: "node_expanded"},
		{Phase: "search", Kind: "action"},
	}
	aggregateSink := &MemorySink{}
	aggregate := NewCollector("debug", aggregateSink)
	traceSink := &MemorySink{}
	trace := NewCollector("trace", traceSink)
	for _, e := range events {
		aggregate.Observe(e)
		trace.Observe(e)
	}
	_ = aggregate.Close(context.Background())
	_ = trace.Close(context.Background())
	if len(aggregateSink.Events()) != 0 {
		t.Fatal("aggregate must not write trace events")
	}
	if aggregate.Metrics().EventCount != 3 {
		t.Fatalf("aggregate count=%d", aggregate.Metrics().EventCount)
	}
	if len(traceSink.Events()) != 4 {
		t.Fatalf("trace events=%d", len(traceSink.Events()))
	}
	if trace.Metrics().EventCount != 4 {
		t.Fatalf("trace count=%d", trace.Metrics().EventCount)
	}
}

func TestDebugCollectorDoesNotRetainActionStreamOrWriteSink(t *testing.T) {
	sink := &MemorySink{}
	c := NewCollector("debug", sink)
	for i := 0; i < 10000; i++ {
		c.Observe(bearing.Event{Phase: "search", Kind: "action", Action: "expand", LogicalStep: uint64(i + 1)})
	}
	c.Observe(bearing.Event{Component: "BOLTS", Phase: "search", Kind: "search_started"})
	c.Observe(bearing.Event{Component: "BOLTS", Phase: "search", Kind: "search_finished"})
	if err := c.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	m := c.Metrics()
	if m.EventCount != 2 {
		t.Fatalf("event_count=%d", m.EventCount)
	}
	if m.SinkWriteNS != 0 {
		t.Fatalf("sink_write_ns=%d", m.SinkWriteNS)
	}
	if len(sink.Events()) != 0 {
		t.Fatalf("debug wrote %d trace events", len(sink.Events()))
	}
	if m.Summary.KindCounts["action"] != 0 {
		t.Fatalf("action events were retained")
	}
}

func TestDebugCollectorAggregatesFrontierWithoutTraceStorage(t *testing.T) {
	c := NewCollector("debug", &MemorySink{})
	c.Observe(bearing.Event{Component: "BOLTS", Phase: "dijkstra", Kind: "node_expanded", Attributes: map[string]any{"frontier_size": 7}})
	c.Observe(bearing.Event{Component: "BOLTS", Phase: "dijkstra", Kind: "frontier_selected", Attributes: map[string]any{"frontier_size": 3}})
	m := c.Metrics()
	if m.DebugSummary.MaxFrontierSize != 7 {
		t.Fatalf("max_frontier_size=%d", m.DebugSummary.MaxFrontierSize)
	}
	if m.DebugSummary.ComponentEventCounts["BOLTS"] != 2 {
		t.Fatalf("component events=%v", m.DebugSummary.ComponentEventCounts)
	}
}
