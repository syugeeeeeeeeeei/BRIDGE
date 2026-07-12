package bearing

const TraceSchemaVersion = "bridge.trace.v1"

// Event is the stable, replay-oriented envelope shared by BRIDGE components.
// Attributes contain event-specific immutable state deltas.
type Event struct {
	SchemaVersion string         `json:"schema_version,omitempty"`
	RunID         string         `json:"run_id,omitempty"`
	TaskID        string         `json:"task_id,omitempty"`
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
