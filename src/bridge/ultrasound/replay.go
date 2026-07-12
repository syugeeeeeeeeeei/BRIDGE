package ultrasound

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
)

type QualityPoint struct {
	Sequence  uint64  `json:"sequence"`
	ElapsedNS int64   `json:"elapsed_ns"`
	Work      uint64  `json:"work"`
	Distance  float64 `json:"path_cost"`
}
type BudgetPoint struct {
	Sequence   uint64 `json:"sequence"`
	Work       uint64 `json:"work"`
	FromExpand uint64 `json:"from_expand"`
	ToExpand   uint64 `json:"to_expand"`
}

type ReplayState struct {
	SchemaVersion      string             `json:"schema_version"`
	RunID              string             `json:"run_id"`
	LastSequence       uint64             `json:"last_sequence"`
	LastElapsedNS      int64              `json:"last_elapsed_ns"`
	CurrentComponent   string             `json:"current_component,omitempty"`
	CurrentPhase       string             `json:"current_phase,omitempty"`
	ExpandedNodes      map[uint32]bool    `json:"expanded_nodes"`
	FrontierNodes      map[uint32]bool    `json:"frontier_nodes"`
	EvaluatedEdges     map[string]bool    `json:"evaluated_edges"`
	Parents            map[uint32]uint32  `json:"parents"`
	Distances          map[uint32]float64 `json:"distances"`
	CandidatePaths     [][]uint32         `json:"candidate_paths"`
	QualityHistory     []QualityPoint     `json:"quality_history"`
	BudgetHistory      []BudgetPoint      `json:"budget_history"`
	FallbackCount      uint64             `json:"fallback_count"`
	CertificationCount uint64             `json:"certification_count"`
	Work               uint64             `json:"work"`
	Finished           bool               `json:"finished"`
}

func ReplayFile(path string) (ReplayState, error) {
	f, err := os.Open(path)
	if err != nil {
		return ReplayState{}, err
	}
	defer f.Close()
	state := ReplayState{ExpandedNodes: map[uint32]bool{}, FrontierNodes: map[uint32]bool{}, EvaluatedEdges: map[string]bool{}, Parents: map[uint32]uint32{}, Distances: map[uint32]float64{}}
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for s.Scan() {
		var e bearing.Event
		if err := json.Unmarshal(s.Bytes(), &e); err != nil {
			return state, err
		}
		if err := ApplyEvent(&state, e); err != nil {
			return state, err
		}
	}
	if err := s.Err(); err != nil {
		return state, err
	}
	return state, nil
}

func ApplyEvent(s *ReplayState, e bearing.Event) error {
	if e.Sequence != 0 && s.LastSequence != 0 && e.Sequence != s.LastSequence+1 {
		return fmt.Errorf("trace sequence gap: got %d after %d", e.Sequence, s.LastSequence)
	}
	if e.SchemaVersion != "" {
		s.SchemaVersion = e.SchemaVersion
	}
	if e.RunID != "" {
		s.RunID = e.RunID
	}
	s.LastSequence = e.Sequence
	s.LastElapsedNS = e.ElapsedNS
	s.CurrentComponent = e.Component
	s.CurrentPhase = e.Phase
	if e.WorkAfter > s.Work {
		s.Work = e.WorkAfter
	}
	node := attrUint32(e.Attributes, "node")
	switch e.Kind {
	case "frontier_enqueued":
		s.FrontierNodes[node] = true
	case "frontier_selected":
		delete(s.FrontierNodes, node)
	case "node_expanded":
		s.ExpandedNodes[node] = true
		delete(s.FrontierNodes, node)
		if d, ok := attrFloat(e.Attributes, "distance"); ok {
			s.Distances[node] = d
		}
	case "edge_evaluated":
		from := attrUint32(e.Attributes, "from")
		to := attrUint32(e.Attributes, "to")
		s.EvaluatedEdges[fmt.Sprintf("%d>%d", from, to)] = true
	case "relaxation":
		if accepted, _ := e.Attributes["accepted"].(bool); accepted {
			to := attrUint32(e.Attributes, "to")
			from := attrUint32(e.Attributes, "from")
			s.Parents[to] = from
			if d, ok := attrFloat(e.Attributes, "new_distance"); ok {
				s.Distances[to] = d
			}
		}
	case "incumbent_updated", "candidate_submitted":
		if p := attrPath(e.Attributes, "path"); len(p) > 0 {
			s.CandidatePaths = append(s.CandidatePaths, p)
		}
		if d, ok := attrFloat(e.Attributes, "distance"); ok {
			s.QualityHistory = append(s.QualityHistory, QualityPoint{Sequence: e.Sequence, ElapsedNS: e.ElapsedNS, Work: e.WorkAfter, Distance: d})
		}
	case "budget_extended":
		s.BudgetHistory = append(s.BudgetHistory, BudgetPoint{Sequence: e.Sequence, Work: e.WorkAfter, FromExpand: uint64(attrUint32(e.Attributes, "from_expand")), ToExpand: uint64(attrUint32(e.Attributes, "to_expand"))})
	case "fallback_started":
		s.FallbackCount++
	case "certification_started":
		s.CertificationCount++
	case "search_finished", "component_finished":
		s.Finished = true
	}
	return nil
}
func attrUint32(m map[string]any, key string) uint32 {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case float64:
		return uint32(v)
	case uint32:
		return v
	case int:
		return uint32(v)
	case uint64:
		return uint32(v)
	}
	return 0
}
func attrFloat(m map[string]any, key string) (float64, bool) {
	if m == nil {
		return 0, false
	}
	switch v := m[key].(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case uint64:
		return float64(v), true
	}
	return 0, false
}
func attrPath(m map[string]any, key string) []uint32 {
	if m == nil {
		return nil
	}
	raw, ok := m[key].([]any)
	if !ok {
		if p, ok := m[key].([]uint32); ok {
			return append([]uint32(nil), p...)
		}
		return nil
	}
	out := make([]uint32, 0, len(raw))
	for _, x := range raw {
		if f, ok := x.(float64); ok {
			out = append(out, uint32(f))
		}
	}
	return out
}
