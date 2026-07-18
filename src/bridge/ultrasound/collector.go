package ultrasound

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
)

type DebugSummary struct {
	CandidateUpdateCount   uint64            `json:"candidate_update_count"`
	FallbackCount          uint64            `json:"fallback_count"`
	CertificationCount     uint64            `json:"certification_count"`
	StateReuseAppliedCount uint64            `json:"state_reuse_applied_count"`
	MaxFrontierSize        uint64            `json:"max_frontier_size"`
	ComponentEventCounts   map[string]uint64 `json:"component_event_counts,omitempty"`
}

type SpanMetric struct {
	RunID        string `json:"run_id,omitempty"`
	SpanID       string `json:"span_id"`
	ParentSpanID string `json:"parent_span_id,omitempty"`
	Component    string `json:"component"`
	Operation    string `json:"operation"`
	StartedNS    int64  `json:"started_ns"`
	CompletedNS  int64  `json:"completed_ns"`
	DurationNS   int64  `json:"duration_ns"`
	Failed       bool   `json:"failed,omitempty"`
}

type SpanSummary struct {
	Completed      []SpanMetric `json:"completed,omitempty"`
	Incomplete     uint64       `json:"incomplete"`
	DuplicateStart uint64       `json:"duplicate_start"`
	OrphanComplete uint64       `json:"orphan_complete"`
}

type CollectorMetrics struct {
	EventCount     uint64         `json:"event_count"`
	DroppedEvents  uint64         `json:"dropped_events"`
	Truncated      bool           `json:"truncated"`
	FirstSequence  uint64         `json:"first_sequence,omitempty"`
	LastSequence   uint64         `json:"last_sequence,omitempty"`
	ObservationNS  int64          `json:"observation_ns"`
	SinkWriteNS    int64          `json:"sink_write_ns"`
	Summary        TraceSummary   `json:"summary"`
	QualityHistory []QualityPoint `json:"quality_history,omitempty"`
	BudgetHistory  []BudgetPoint  `json:"budget_history,omitempty"`
	DebugSummary   DebugSummary   `json:"debug_summary,omitempty"`
	Spans          SpanSummary    `json:"spans,omitempty"`
}

type spanKey struct {
	runID  string
	spanID string
}

type Collector struct {
	mu          sync.Mutex
	mode        string
	sink        EventSink
	start, last time.Time
	seq         uint64
	maxEvents   uint64
	metrics     CollectorMetrics
	err         error
	closed      bool
	openSpans   map[spanKey]SpanMetric
}

func NewCollector(mode string, sink EventSink) *Collector {
	return NewCollectorWithLimit(mode, sink, 0)
}
func NewCollectorWithLimit(mode string, sink EventSink, maxEvents uint64) *Collector {
	if sink == nil {
		sink = DiscardSink{}
	}
	now := time.Now()
	return &Collector{
		mode: mode, sink: sink, start: now, last: now, maxEvents: maxEvents,
		metrics:   CollectorMetrics{Summary: TraceSummary{KindCounts: map[string]uint64{}, PhaseCounts: map[string]uint64{}}, DebugSummary: DebugSummary{ComponentEventCounts: map[string]uint64{}}, Spans: SpanSummary{Completed: make([]SpanMetric, 0, 16)}},
		openSpans: map[spanKey]SpanMetric{},
	}
}

func (c *Collector) Wants(kind string) bool {
	class := bearing.ClassifyEvent(kind)
	switch c.mode {
	case "trace":
		return true
	case "debug":
		if kind == "state_delta" {
			return true
		}
		return class == bearing.ClassLifecycle || class == bearing.ClassControl || class == bearing.ClassCandidate
	default:
		return class == bearing.ClassLifecycle
	}
}
func (c *Collector) Observe(e bearing.Event) {
	started := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	defer func() { c.metrics.ObservationNS += time.Since(started).Nanoseconds() }()
	if c.err != nil || c.closed {
		return
	}
	if c.mode == "debug" {
		class := bearing.ClassifyEvent(e.Kind)
		if class != bearing.ClassLifecycle && class != bearing.ClassControl && class != bearing.ClassCandidate && e.Kind != string(bearing.KindFrontierSelected) && e.Kind != string(bearing.KindNodeExpanded) {
			return
		}
	} else if !c.Wants(e.Kind) {
		return
	}
	if c.maxEvents > 0 && c.metrics.EventCount >= c.maxEvents {
		c.metrics.DroppedEvents++
		c.metrics.Truncated = true
		return
	}
	now := time.Now()
	c.seq++
	e.SchemaVersion = bearing.TraceSchemaVersion
	e.Sequence = c.seq
	e.ElapsedNS = now.Sub(c.start).Nanoseconds()
	e.DeltaNS = now.Sub(c.last).Nanoseconds()
	c.last = now
	if e.ScheduledStep == 0 {
		e.ScheduledStep = e.LogicalStep
	}
	if c.metrics.FirstSequence == 0 {
		c.metrics.FirstSequence = c.seq
	}
	c.metrics.LastSequence = c.seq

	if c.mode == "trace" {
		writeStarted := time.Now()
		c.err = c.sink.WriteEvent(context.Background(), e)
		c.metrics.SinkWriteNS += time.Since(writeStarted).Nanoseconds()
		if c.err != nil {
			return
		}
	}

	c.metrics.EventCount++
	s := &c.metrics.Summary
	s.EventCount++
	if s.FirstSequence == 0 {
		s.FirstSequence = e.Sequence
	}
	s.LastSequence = e.Sequence
	s.KindCounts[e.Kind]++
	s.PhaseCounts[e.Phase]++
	if e.Component != "" {
		c.metrics.DebugSummary.ComponentEventCounts[e.Component]++
	}
	if e.Kind == string(bearing.KindLifecycle) && e.SpanID != "" {
		c.observeLifecycle(e)
	}
	switch bearing.EventKind(e.Kind) {
	case bearing.KindCandidateSubmitted, bearing.KindIncumbentUpdated:
		c.metrics.DebugSummary.CandidateUpdateCount++
	case bearing.KindFallbackStarted:
		c.metrics.DebugSummary.FallbackCount++
	case bearing.KindCertificationStarted:
		c.metrics.DebugSummary.CertificationCount++
	case bearing.KindStateReuseApplied:
		c.metrics.DebugSummary.StateReuseAppliedCount++
	}
	if n, ok := attrFloat(e.Attributes, "frontier_size"); ok && n >= 0 && uint64(n) > c.metrics.DebugSummary.MaxFrontierSize {
		c.metrics.DebugSummary.MaxFrontierSize = uint64(n)
	}
	if e.LogicalStep > s.MaxLogicalStep {
		s.MaxLogicalStep = e.LogicalStep
	}
	if e.Kind == "incumbent_updated" || e.Kind == "candidate_submitted" {
		if d, ok := attrFloat(e.Attributes, "distance"); ok {
			c.metrics.QualityHistory = append(c.metrics.QualityHistory, QualityPoint{Sequence: e.Sequence, ElapsedNS: e.ElapsedNS, Work: e.WorkAfter, Distance: d})
		}
	}
	if e.Kind == "budget_extended" {
		c.metrics.BudgetHistory = append(c.metrics.BudgetHistory, BudgetPoint{Sequence: e.Sequence, Work: e.WorkAfter, FromExpand: uint64(attrUint32(e.Attributes, "from_expand")), ToExpand: uint64(attrUint32(e.Attributes, "to_expand"))})
	}
}
func (c *Collector) observeLifecycle(e bearing.Event) {
	key := spanKey{runID: e.RunID, spanID: e.SpanID}
	switch bearing.LifecyclePhase(e.Phase) {
	case bearing.LifecycleStarted:
		if _, exists := c.openSpans[key]; exists {
			c.metrics.Spans.DuplicateStart++
			return
		}
		c.openSpans[key] = SpanMetric{RunID: e.RunID, SpanID: e.SpanID, ParentSpanID: e.ParentSpanID, Component: e.Component, Operation: e.Action, StartedNS: e.ElapsedNS}
	case bearing.LifecycleCompleted, bearing.LifecycleFailed:
		span, exists := c.openSpans[key]
		if !exists {
			c.metrics.Spans.OrphanComplete++
			return
		}
		delete(c.openSpans, key)
		span.CompletedNS = e.ElapsedNS
		span.DurationNS = span.CompletedNS - span.StartedNS
		span.Failed = bearing.LifecyclePhase(e.Phase) == bearing.LifecycleFailed
		c.metrics.Spans.Completed = append(c.metrics.Spans.Completed, span)
	}
}

func (c *Collector) Metrics() CollectorMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	m := cloneMetrics(c.metrics)
	m.Spans.Incomplete = uint64(len(c.openSpans))
	return m
}
func cloneMetrics(m CollectorMetrics) CollectorMetrics {
	m.Summary.KindCounts = cloneCounts(m.Summary.KindCounts)
	m.Summary.PhaseCounts = cloneCounts(m.Summary.PhaseCounts)
	m.QualityHistory = append([]QualityPoint(nil), m.QualityHistory...)
	m.BudgetHistory = append([]BudgetPoint(nil), m.BudgetHistory...)
	m.DebugSummary.ComponentEventCounts = cloneCounts(m.DebugSummary.ComponentEventCounts)
	m.Spans.Completed = append([]SpanMetric(nil), m.Spans.Completed...)
	return m
}
func cloneCounts(in map[string]uint64) map[string]uint64 {
	out := make(map[string]uint64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// Reset starts a new observation run while preserving the collector configuration.
// A collector is otherwise a one-run object; callers that intentionally reuse it MUST call Reset.
func (c *Collector) Reset() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		return fmt.Errorf("collector must be closed before reset")
	}
	now := time.Now()
	c.start, c.last = now, now
	c.seq = 0
	c.metrics = CollectorMetrics{Summary: TraceSummary{KindCounts: map[string]uint64{}, PhaseCounts: map[string]uint64{}}, DebugSummary: DebugSummary{ComponentEventCounts: map[string]uint64{}}, Spans: SpanSummary{Completed: make([]SpanMetric, 0, 16)}}
	c.err = nil
	c.closed = false
	c.openSpans = map[spanKey]SpanMetric{}
	return nil
}

func (c *Collector) ObservationMode() string          { return c.mode }
func (c *Collector) ObservationEventCount() uint64    { return c.Metrics().EventCount }
func (c *Collector) ObservationDroppedEvents() uint64 { return c.Metrics().DroppedEvents }
func (c *Collector) ObservationTruncated() bool       { return c.Metrics().Truncated }
func (c *Collector) ObservationOverheadNS() int64     { return c.Metrics().ObservationNS }
func (c *Collector) ObservationSinkWriteNS() int64    { return c.Metrics().SinkWriteNS }
func (c *Collector) TraceSummary() TraceSummary       { return c.Metrics().Summary }
func (c *Collector) ObservationSummary() any          { return c.Metrics().Summary }
func (c *Collector) ObservationSpans() any            { return c.Metrics().Spans }
func (c *Collector) Err() error                       { c.mu.Lock(); defer c.mu.Unlock(); return c.err }
func (c *Collector) Close(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		err := c.err
		c.mu.Unlock()
		return err
	}
	c.closed = true
	err := c.err
	mode := c.mode
	c.mu.Unlock()
	if mode != "trace" {
		return err
	}
	closeErr := c.sink.Close(ctx)
	if err != nil {
		return err
	}
	return closeErr
}
