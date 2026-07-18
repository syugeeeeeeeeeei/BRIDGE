package gate

import (
	"fmt"
)

func resolveObservationMode(opts RouteOptions, fallback ObservationMode) (ObservationMode, error) {
	obsMode := opts.Observation.Mode
	if obsMode == "" {
		obsMode = fallback
	}
	if obsMode == "" {
		obsMode = ObservationMinimum
	}
	switch obsMode {
	case ObservationMinimum, ObservationDebug, ObservationTrace:
		return obsMode, nil
	default:
		return "", &PublicError{Code: "INVALID_OBSERVATION", Message: fmt.Sprintf("unsupported observation mode %q", obsMode)}
	}
}

func observationResultFromReporter(reporter ObservationReporter) *ObservationResult {
	if reporter == nil {
		return nil
	}
	out := &ObservationResult{Mode: ObservationMode(reporter.ObservationMode()), EventCount: reporter.ObservationEventCount(), DroppedEvents: reporter.ObservationDroppedEvents(), Truncated: reporter.ObservationTruncated()}
	if r, ok := reporter.(interface{ ObservationOverheadNS() int64 }); ok {
		out.OverheadNS = r.ObservationOverheadNS()
	}
	if r, ok := reporter.(interface{ ObservationSinkWriteNS() int64 }); ok {
		out.SinkWriteNS = r.ObservationSinkWriteNS()
	}
	if r, ok := reporter.(interface{ ObservationSummary() any }); ok {
		out.Summary = r.ObservationSummary()
	}
	if r, ok := reporter.(interface{ ObservationSpans() any }); ok {
		out.Spans = r.ObservationSpans()
	}
	return out
}

func telemetryFloat(m map[string]any, k string, fallback float64) float64 {
	if m == nil {
		return fallback
	}
	switch v := m[k].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case uint64:
		return float64(v)
	default:
		return fallback
	}
}
