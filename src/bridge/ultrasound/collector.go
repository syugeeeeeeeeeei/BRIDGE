package ultrasound

import (
	"context"
	"encoding/binary"
	"hash/fnv"
	"sync"
	"time"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
)

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
}

type Collector struct {
	mu          sync.Mutex
	mode        string
	sink        EventSink
	start, last time.Time
	seq         uint64
	maxEvents   uint64
	sampleRate  float64
	sampleSeed  uint64
	seen        uint64
	metrics     CollectorMetrics
	events      []bearing.Event
	err         error
	closed      bool
}

func NewCollector(mode string, sink EventSink) *Collector {
	return NewCollectorConfigured(mode, sink, 0, 1, 0)
}
func NewCollectorWithLimit(mode string, sink EventSink, maxEvents uint64) *Collector {
	return NewCollectorConfigured(mode, sink, maxEvents, 1, 0)
}
func NewCollectorConfigured(mode string, sink EventSink, maxEvents uint64, sampleRate float64, sampleSeed uint64) *Collector {
	if sink == nil {
		sink = DiscardSink{}
	}
	if sampleRate <= 0 || sampleRate > 1 {
		sampleRate = 1
	}
	now := time.Now()
	return &Collector{mode: mode, sink: sink, start: now, last: now, maxEvents: maxEvents, sampleRate: sampleRate, sampleSeed: sampleSeed}
}
func (c *Collector) Wants(kind string) bool {
	class := bearing.ClassifyEvent(kind)
	switch c.mode {
	case "profile":
		return true
	case "trace":
		return class != bearing.ClassProfile
	case "summary":
		return class == bearing.ClassControl || class == bearing.ClassCandidate
	default:
		return false
	}
}
func (c *Collector) Observe(e bearing.Event) {
	started := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	defer func() { c.metrics.ObservationNS += time.Since(started).Nanoseconds() }()
	if c.err != nil || c.closed || !c.Wants(e.Kind) {
		return
	}
	c.seen++
	if !c.sampled(c.seen, e) {
		c.metrics.DroppedEvents++
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
	c.events = append(c.events, cloneEvent(e))
	// summary intentionally performs no trace I/O.
	if c.mode == "trace" || c.mode == "profile" {
		writeStarted := time.Now()
		c.err = c.sink.WriteEvent(context.Background(), e)
		c.metrics.SinkWriteNS += time.Since(writeStarted).Nanoseconds()
	}
	if c.err == nil {
		c.metrics.EventCount++
		c.metrics.Summary = Summarize(c.events)
		if e.Kind == "incumbent_updated" || e.Kind == "candidate_submitted" {
			if d, ok := attrFloat(e.Attributes, "distance"); ok {
				c.metrics.QualityHistory = append(c.metrics.QualityHistory, QualityPoint{Sequence: e.Sequence, ElapsedNS: e.ElapsedNS, Work: e.WorkAfter, Distance: d})
			}
		}
		if e.Kind == "budget_extended" {
			c.metrics.BudgetHistory = append(c.metrics.BudgetHistory, BudgetPoint{Sequence: e.Sequence, Work: e.WorkAfter, FromExpand: uint64(attrUint32(e.Attributes, "from_expand")), ToExpand: uint64(attrUint32(e.Attributes, "to_expand"))})
		}
	}
}
func (c *Collector) Metrics() CollectorMetrics        { c.mu.Lock(); defer c.mu.Unlock(); return c.metrics }
func (c *Collector) ObservationMode() string          { return c.mode }
func (c *Collector) ObservationEventCount() uint64    { return c.Metrics().EventCount }
func (c *Collector) ObservationDroppedEvents() uint64 { return c.Metrics().DroppedEvents }
func (c *Collector) ObservationTruncated() bool       { return c.Metrics().Truncated }
func (c *Collector) ObservationOverheadNS() int64     { return c.Metrics().ObservationNS }
func (c *Collector) ObservationSinkWriteNS() int64    { return c.Metrics().SinkWriteNS }
func (c *Collector) TraceSummary() TraceSummary       { return c.Metrics().Summary }
func (c *Collector) ObservationSummary() any          { return c.Metrics().Summary }
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
	if mode != "trace" && mode != "profile" {
		return err
	}
	closeErr := c.sink.Close(ctx)
	if err != nil {
		return err
	}
	return closeErr
}

func (c *Collector) sampled(ordinal uint64, e bearing.Event) bool {
	if c.sampleRate >= 1 {
		return true
	}
	h := fnv.New64a()
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], c.sampleSeed)
	_, _ = h.Write(b[:])
	binary.LittleEndian.PutUint64(b[:], ordinal)
	_, _ = h.Write(b[:])
	_, _ = h.Write([]byte(e.Kind))
	threshold := uint64(c.sampleRate * float64(^uint64(0)))
	return h.Sum64() <= threshold
}
