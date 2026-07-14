package truss

import (
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"reflect"
	"testing"
)

func TestDeterministicEpochMerge(t *testing.T) {
	a := NewEvidenceStore()
	b := NewEvidenceStore()
	items := []core.Evidence{{ID: "z", Solver: "b", HypothesisID: "h2", Proof: core.ProofEmpirical}, {ID: "a", Solver: "a", HypothesisID: "h1", Scope: core.Region{Nodes: []core.NodeID{0}}, GeneratedWork: 1, Proof: core.ProofExact}}
	if err := a.MergeEpoch(items); err != nil {
		t.Fatal(err)
	}
	if err := b.MergeEpoch([]core.Evidence{items[1], items[0]}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.Snapshot(), b.Snapshot()) {
		t.Fatal("merge order changed result")
	}
}
