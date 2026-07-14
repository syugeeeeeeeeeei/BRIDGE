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
	// v0.15.0 ranks proof strength before equal distance and Work.
	if !res.Exact || !res.QualityCertified {
		t.Fatalf("certified candidate was not preferred: %+v", res)
	}
}
