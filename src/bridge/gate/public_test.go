package gate

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/ultrasound"
)

func validPublicRequest() RouteRequest {
	return RouteRequest{
		SchemaVersion: RouteRequestSchemaV1,
		RequestID:     "test-1",
		Graph:         GraphInput{Type: "inline", Nodes: []GraphNode{{ID: 0}, {ID: 1}, {ID: 2}}, Edges: []GraphEdge{{From: 0, To: 1, Weight: 1}, {From: 1, To: 2, Weight: 2}}},
		Route:         RouteInput{Source: 0, Target: 2},
	}
}
func TestPublicRouterRoute(t *testing.T) {
	got, err := NewRouter().Route(context.Background(), validPublicRequest(), RouteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Found || got.Status != "found" || got.Distance == nil || *got.Distance != 3 {
		t.Fatalf("unexpected result: %+v", got)
	}
	if got.SchemaVersion != RouteResultSchemaV1 {
		t.Fatalf("schema: %s", got.SchemaVersion)
	}
}
func TestPublicRouterObservationMemory(t *testing.T) {
	sink := &ultrasound.MemorySink{}
	_, err := NewRouter().Route(context.Background(), validPublicRequest(), RouteOptions{Observation: ObservationOptions{Mode: ObservationTrace, Observer: ultrasound.NewCollector("trace", sink)}})
	if err != nil {
		t.Fatal(err)
	}
	if len(sink.Events()) == 0 {
		t.Fatal("expected trace events")
	}
}
func TestStrictJSONRejectsUnknownAndDuplicate(t *testing.T) {
	var req RouteRequest
	if err := DecodeStrictJSON([]byte(`{"schema_version":"bridge.route.v1","schema_version":"bridge.route.v1"}`), &req); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate not rejected: %v", err)
	}
	if err := DecodeStrictJSON([]byte(`{"schema_version":"bridge.route.v1","graph":{"type":"inline"},"route":{"source":0,"target":0},"unknown":1}`), &req); err == nil {
		t.Fatal("unknown field not rejected")
	}
}
func TestPublicRouterRejectsNonPositiveWeight(t *testing.T) {
	req := validPublicRequest()
	req.Graph.Edges[0].Weight = 0
	if _, err := NewRouter().Route(context.Background(), req, RouteOptions{}); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestObservationDoesNotChangeRouteDecision(t *testing.T) {
	req := RouteRequest{
		SchemaVersion: RouteRequestSchemaV1,
		Graph:         GraphInput{Type: "inline", Nodes: []GraphNode{{ID: 0}, {ID: 1}, {ID: 2}}, Edges: []GraphEdge{{From: 0, To: 1, Weight: 1}, {From: 1, To: 2, Weight: 2}, {From: 0, To: 2, Weight: 10}}},
		Route:         RouteInput{Source: 0, Target: 2, Mode: core.ModeBalanced, Seed: 1},
	}
	off, err := NewRouter().Route(context.Background(), req, RouteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	sink := &ultrasound.MemorySink{}
	trace, err := NewRouter().Route(context.Background(), req, RouteOptions{Observation: ObservationOptions{Mode: ObservationTrace, Observer: ultrasound.NewCollector("trace", sink)}})
	if err != nil {
		t.Fatal(err)
	}
	if off.Status != trace.Status || off.Found != trace.Found || off.Exact != trace.Exact || off.SolverName != trace.SolverName || off.Work != trace.Work {
		t.Fatalf("off=%+v trace=%+v", off, trace)
	}
	if (off.Distance == nil) != (trace.Distance == nil) || (off.Distance != nil && *off.Distance != *trace.Distance) {
		t.Fatalf("distance differs")
	}
	if !slices.Equal(off.Path, trace.Path) {
		t.Fatalf("path off=%v trace=%v", off.Path, trace.Path)
	}
	if len(sink.Events()) == 0 {
		t.Fatal("expected trace events")
	}
}
