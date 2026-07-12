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

func TestObservationModesSeparateSummaryTraceAndProfile(t *testing.T) {
	events := []bearing.Event{
		{Phase: "search", Kind: "search_started"},
		{Phase: "search", Kind: "incumbent_updated", Attributes: map[string]any{"distance": 2.0}},
		{Phase: "search", Kind: "node_expanded"},
		{Phase: "search", Kind: "action"},
	}
	summarySink := &MemorySink{}
	summary := NewCollector("summary", summarySink)
	traceSink := &MemorySink{}
	trace := NewCollector("trace", traceSink)
	profileSink := &MemorySink{}
	profile := NewCollector("profile", profileSink)
	for _, e := range events {
		summary.Observe(e)
		trace.Observe(e)
		profile.Observe(e)
	}
	_ = summary.Close(context.Background())
	_ = trace.Close(context.Background())
	_ = profile.Close(context.Background())
	if len(summarySink.Events()) != 0 {
		t.Fatal("summary must not write trace events")
	}
	if summary.Metrics().EventCount != 2 {
		t.Fatalf("summary count=%d", summary.Metrics().EventCount)
	}
	if len(traceSink.Events()) != 3 {
		t.Fatalf("trace events=%d", len(traceSink.Events()))
	}
	if len(profileSink.Events()) != 4 {
		t.Fatalf("profile events=%d", len(profileSink.Events()))
	}
	if profile.Metrics().ObservationNS <= 0 {
		t.Fatal("profile overhead was not recorded")
	}
}
