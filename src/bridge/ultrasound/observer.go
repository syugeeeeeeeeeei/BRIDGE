package ultrasound

import (
	"encoding/json"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"io"
	"math"
	"sync"
)

type MemoryObserver struct {
	mu     sync.Mutex
	Events []bearing.Event
	Detail bool
}

func (m *MemoryObserver) Observe(e bearing.Event) {
	m.mu.Lock()
	m.Events = append(m.Events, cloneEvent(e))
	m.mu.Unlock()
}
func (m *MemoryObserver) Wants(string) bool { return m.Detail }

type JSONLObserver struct {
	mu     sync.Mutex
	enc    *json.Encoder
	Detail bool
	Err    error
}

func NewJSONLObserver(w io.Writer) *JSONLObserver {
	return &JSONLObserver{enc: json.NewEncoder(w), Detail: true}
}
func (o *JSONLObserver) Observe(e bearing.Event) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.Err == nil {
		o.Err = o.enc.Encode(cloneEvent(e))
	}
}
func (o *JSONLObserver) Wants(string) bool { return o.Detail }

func cloneEvent(e bearing.Event) bearing.Event {
	if e.Attributes == nil {
		return e
	}
	attrs := make(map[string]any, len(e.Attributes))
	for k, v := range e.Attributes {
		attrs[k] = cloneValue(v)
	}
	e.Attributes = attrs
	return e
}
func cloneValue(v any) any {
	switch x := v.(type) {
	case []uint32:
		out := append([]uint32(nil), x...)
		return out
	case []int:
		out := append([]int(nil), x...)
		return out
	case []string:
		out := append([]string(nil), x...)
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, y := range x {
			out[k] = cloneValue(y)
		}
		return out
	case float64:
		if math.IsInf(x, 1) {
			return "Infinity"
		}
		if math.IsInf(x, -1) {
			return "-Infinity"
		}
		if math.IsNaN(x) {
			return "NaN"
		}
		return x
	default:
		return v
	}
}
