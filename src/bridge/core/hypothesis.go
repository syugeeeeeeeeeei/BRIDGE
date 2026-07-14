package core

// HypothesisState is the lifecycle of an ANCHOR partial-graph hypothesis.
type HypothesisState string

const (
	HypothesisRunnable HypothesisState = "runnable"
	HypothesisFrozen   HypothesisState = "frozen"
	HypothesisPruned   HypothesisState = "pruned"
	HypothesisFinished HypothesisState = "finished"
)

type Region struct {
	Nodes   []NodeID `json:"nodes"`
	Version uint64   `json:"version"`
}

func (r Region) Contains(n NodeID) bool {
	for _, v := range r.Nodes {
		if v == n {
			return true
		}
	}
	return false
}
func (r Region) Overlaps(o Region) bool {
	for _, v := range r.Nodes {
		if o.Contains(v) {
			return true
		}
	}
	return false
}
func (r Region) Expanded(nodes ...NodeID) Region {
	out := Region{Nodes: append([]NodeID{}, r.Nodes...), Version: r.Version + 1}
	for _, n := range nodes {
		if !out.Contains(n) {
			out.Nodes = append(out.Nodes, n)
		}
	}
	return out
}

type Checkpoint struct {
	Node         NodeID  `json:"node"`
	Cost         float64 `json:"cost"`
	HypothesisID string  `json:"hypothesis_id"`
}
type Hypothesis struct {
	ID          string          `json:"id"`
	Kind        string          `json:"kind"`
	Region      Region          `json:"region"`
	State       HypothesisState `json:"state"`
	WorkUsed    uint64          `json:"work_used"`
	Candidate   *RouteResult    `json:"candidate,omitempty"`
	LowerBound  *float64        `json:"lower_bound,omitempty"`
	Checkpoints []Checkpoint    `json:"checkpoints,omitempty"`
}
