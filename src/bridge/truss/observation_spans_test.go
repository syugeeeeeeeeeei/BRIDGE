package truss

import (
	"context"
	"testing"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/ultrasound"
)

func TestRoutePublishesLowFrequencyInternalLifecycleSpans(t *testing.T) {
	collector := ultrasound.NewCollector("minimum", nil)
	got, err := New(collector).Route(context.Background(), testGraph(), core.RouteRequest{
		Source:  0,
		Target:  3,
		Mode:    core.ModeBalanced,
		Workers: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Found {
		t.Fatalf("route not found: %+v", got)
	}

	metrics := collector.Metrics()
	if metrics.Spans.Incomplete != 0 || metrics.Spans.DuplicateStart != 0 || metrics.Spans.OrphanComplete != 0 {
		t.Fatalf("invalid lifecycle spans: %+v", metrics.Spans)
	}

	operations := map[string]bool{}
	for _, span := range metrics.Spans.Completed {
		if span.Component == "TRUSS" {
			operations[span.Operation] = true
		}
	}
	for _, operation := range []string{
		"request_adaptation",
		"route",
		"deadline_setup",
		"budget_setup",
		"observer_setup",
		"policy_setup",
		"session_creation",
		"adaptive_execution",
		"final_handoff",
		"finalization",
		"result_integration",
	} {
		if !operations[operation] {
			t.Fatalf("missing TRUSS lifecycle operation %q: %+v", operation, operations)
		}
	}
}
