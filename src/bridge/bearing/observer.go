package bearing

import (
	"strconv"
	"sync/atomic"
)

const TraceSchemaVersion = "bridge.trace.v1"

// Event is the stable, replay-oriented envelope shared by BRIDGE components.
// Attributes contain event-specific immutable state deltas.
type Event struct {
	SchemaVersion string         `json:"schema_version,omitempty"`
	RunID         string         `json:"run_id,omitempty"`
	TaskID        string         `json:"task_id,omitempty"`
	SpanID        string         `json:"span_id,omitempty"`
	ParentSpanID  string         `json:"parent_span_id,omitempty"`
	Sequence      uint64         `json:"sequence,omitempty"`
	LogicalStep   uint64         `json:"logical_step,omitempty"`
	ScheduledStep uint64         `json:"scheduled_step,omitempty"`
	ElapsedNS     int64          `json:"elapsed_ns,omitempty"`
	DeltaNS       int64          `json:"delta_ns,omitempty"`
	Component     string         `json:"component,omitempty"`
	Lane          string         `json:"lane,omitempty"`
	Phase         string         `json:"phase"`
	Kind          string         `json:"kind"`
	Action        string         `json:"action,omitempty"`
	Subject       string         `json:"subject,omitempty"`
	WorkBefore    uint64         `json:"work_before,omitempty"`
	WorkAfter     uint64         `json:"work_after,omitempty"`
	Attributes    map[string]any `json:"attributes,omitempty"`
}

type Observer interface{ Observe(Event) }

// DetailObserver allows producers to avoid constructing high-volume state-delta
// events unless the observer explicitly requested them.
type DetailObserver interface {
	Wants(kind string) bool
}

func Wants(o Observer, kind string) bool {
	if o == nil {
		return false
	}
	if d, ok := o.(DetailObserver); ok {
		return d.Wants(kind)
	}
	return false
}

type NullObserver struct{}

func (NullObserver) Observe(Event)     {}
func (NullObserver) Wants(string) bool { return false }

type SafeObserver struct{ Inner Observer }

func (s SafeObserver) Observe(e Event) {
	if s.Inner == nil {
		return
	}
	func() { defer func() { _ = recover() }(); s.Inner.Observe(e) }()
}
func (s SafeObserver) Wants(kind string) bool {
	if s.Inner == nil {
		return false
	}
	return Wants(s.Inner, kind)
}

// LifecyclePhase identifies a low-frequency operation boundary.
type LifecyclePhase string

const (
	LifecycleStarted   LifecyclePhase = "started"
	LifecycleCompleted LifecyclePhase = "completed"
	LifecycleFailed    LifecyclePhase = "failed"
)

// EmitLifecycle emits a typed, allocation-free lifecycle boundary when the
// observer requested lifecycle events. Timestamping and duration aggregation
// remain observer responsibilities.
func EmitLifecycle(o Observer, runID, taskID, spanID, parentSpanID, component, operation string, phase LifecyclePhase) {
	if o == nil || !Wants(o, string(KindLifecycle)) {
		return
	}
	o.Observe(Event{
		RunID: runID, TaskID: taskID, SpanID: spanID, ParentSpanID: parentSpanID,
		Component: component, Phase: string(phase), Kind: string(KindLifecycle),
		Action: operation,
	})
}

var lifecycleSequence atomic.Uint64

// LifecycleSpan is a low-allocation handle for one lifecycle boundary.
// The zero value is disabled and Finish is idempotent.
type LifecycleSpan struct {
	observer     Observer
	runID        string
	taskID       string
	spanID       string
	parentSpanID string
	component    string
	operation    string
	active       bool
	finished     bool
}

// StartLifecycle emits a started boundary and returns a stack-friendly span handle.
func StartLifecycle(o Observer, runID, taskID, parentSpanID, component, operation string) LifecycleSpan {
	if o == nil || !Wants(o, string(KindLifecycle)) {
		return LifecycleSpan{}
	}
	spanID := component + "-" + strconv.FormatUint(lifecycleSequence.Add(1), 10)
	EmitLifecycle(o, runID, taskID, spanID, parentSpanID, component, operation, LifecycleStarted)
	return LifecycleSpan{observer: o, runID: runID, taskID: taskID, spanID: spanID, parentSpanID: parentSpanID, component: component, operation: operation, active: true}
}

func (s *LifecycleSpan) ID() string {
	if s == nil || !s.active {
		return ""
	}
	return s.spanID
}

func (s *LifecycleSpan) Finish(failed bool) {
	if s == nil || !s.active || s.finished {
		return
	}
	s.finished = true
	phase := LifecycleCompleted
	if failed {
		phase = LifecycleFailed
	}
	EmitLifecycle(s.observer, s.runID, s.taskID, s.spanID, s.parentSpanID, s.component, s.operation, phase)
}

var noopLifecycleFinish = func(bool) {}

// BeginLifecycle emits a started boundary and returns a completion function.
// The disabled path performs no timestamp lookup, span generation, or payload allocation.
func BeginLifecycle(o Observer, runID, taskID, parentSpanID, component, operation string) (string, func(failed bool)) {
	if o == nil || !Wants(o, string(KindLifecycle)) {
		return "", noopLifecycleFinish
	}
	spanID := component + "-" + strconv.FormatUint(lifecycleSequence.Add(1), 10)
	EmitLifecycle(o, runID, taskID, spanID, parentSpanID, component, operation, LifecycleStarted)
	var finished atomic.Bool
	return spanID, func(failed bool) {
		if !finished.CompareAndSwap(false, true) {
			return
		}
		phase := LifecycleCompleted
		if failed {
			phase = LifecycleFailed
		}
		EmitLifecycle(o, runID, taskID, spanID, parentSpanID, component, operation, phase)
	}
}
