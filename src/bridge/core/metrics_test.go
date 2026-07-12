package core

import (
	"encoding/json"
	"testing"
)

func TestResearchMetricContractsSerialize(t *testing.T) {
	result := RouteResult{TimeBreakdown: TimeBreakdown{TotalMS: 2, SolverMS: 1.5, AnchorMS: 1.5}}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) == 0 {
		t.Fatal("empty payload")
	}
	if !result.Work.Valid() {
		t.Fatal("zero work metrics must be valid")
	}
}
