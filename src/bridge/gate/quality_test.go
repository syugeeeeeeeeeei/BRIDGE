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
	// Python reference keeps the earlier candidate when certification returns an
	// equal distance. Exact/certified flags therefore remain those of ANCHOR.
	if res.Exact || res.QualityCertified {
		t.Fatalf("Python parity regression: %+v", res)
	}
}
