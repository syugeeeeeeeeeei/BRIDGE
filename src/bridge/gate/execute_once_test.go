package gate

import (
	"context"
	"testing"
)

func TestPublicRouterExecuteOnce(t *testing.T) {
	got, err := NewRouter().ExecuteOnce(context.Background(), ExecuteRequest{
		SchemaVersion: ExecuteRequestSchemaV1,
		RequestID:     "exec-1",
		Target:        ExecuteTargetInput{ID: "dijkstra"},
		Graph:         validPublicRequest().Graph,
		Route:         validPublicRequest().Route,
	}, RouteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Found || got.TargetID != "dijkstra" || got.ExecutionPath != "execute_once" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestPublicRouterExecuteOnceRejectsUnknownTarget(t *testing.T) {
	_, err := NewRouter().ExecuteOnce(context.Background(), ExecuteRequest{
		SchemaVersion: ExecuteRequestSchemaV1,
		Target:        ExecuteTargetInput{ID: "not-a-solver"},
		Graph:         validPublicRequest().Graph,
		Route:         validPublicRequest().Route,
	}, RouteOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
}
