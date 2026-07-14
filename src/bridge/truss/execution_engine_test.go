package truss

import (
	"context"
	"testing"
)

func TestExecutionEngineCancellationLeavesNilOutputsWithoutPanic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out := (ExecutionEngine{Workers: 1}).Run(ctx, []TaskFunc{func(context.Context) any { return "unexpected" }})
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0] != nil {
		t.Fatalf("cancelled task executed: %v", out[0])
	}
}
