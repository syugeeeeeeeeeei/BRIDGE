package gate

import (
	"context"
	"testing"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

func TestQualityPreservesEarlierEqualDistanceCandidate(t *testing.T) {
	res, err := New(nil).Route(context.Background(), graph(), core.RouteRequest{Source: 0, Target: 3, Mode: core.ModeQuality, Workers: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Found || res.Distance != 2 {
		t.Fatalf("unexpected quality result: %+v", res)
	}
	// Quality mode uses a weighted ANCHOR session. Finishing that session does
	// not prove global optimality unless an independent certification pass ran.
	if res.Exact || res.QualityCertified {
		t.Fatalf("weighted search must not claim exact certification: %+v", res)
	}
}
