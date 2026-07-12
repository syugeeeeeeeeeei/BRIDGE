package healthy

import (
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

type ReconstructionResult struct {
	Work       core.WorkMetrics `json:"work"`
	Verifiable bool             `json:"verifiable"`
	Errors     []string         `json:"errors,omitempty"`
}

// ReconstructWork rebuilds Work from unsampled, non-truncated profile Action Events.
// The caller must pass the trace completeness facts from its manifest.
func ReconstructWork(events []bearing.Event, sampleRate float64, truncated bool, dropped uint64) ReconstructionResult {
	out := ReconstructionResult{Verifiable: true}
	if sampleRate != 1 {
		out.Verifiable = false
		out.Errors = append(out.Errors, "sample_rate must be 1.0")
	}
	if truncated {
		out.Verifiable = false
		out.Errors = append(out.Errors, "trace is truncated")
	}
	if dropped > 0 {
		out.Verifiable = false
		out.Errors = append(out.Errors, fmt.Sprintf("trace dropped %d events", dropped))
	}
	if !out.Verifiable {
		return out
	}
	var lastLogical, lastScheduled uint64
	for _, e := range events {
		if e.Kind != "action" {
			continue
		}
		if !core.IsWorkAction(e.Action) {
			out.Verifiable = false
			out.Errors = append(out.Errors, "unknown action event: "+e.Action)
			continue
		}
		out.Work.AddAction(e.Action)
		if e.LogicalStep > lastLogical {
			lastLogical = e.LogicalStep
		}
		if e.ScheduledStep > lastScheduled {
			lastScheduled = e.ScheduledStep
		}
	}
	out.Work.LogicalSteps = lastLogical
	out.Work.ScheduledSteps = lastScheduled
	if !out.Work.Valid() {
		out.Verifiable = false
		out.Errors = append(out.Errors, out.Work.ValidationErrors()...)
	}
	return out
}
