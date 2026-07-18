package gate

import (
	"errors"
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"math"
)

func buildGraph(in GraphInput) (*core.AdjacencyGraph, error) {
	if in.Type != "inline" {
		return nil, fmt.Errorf("graph.type must be inline or file")
	}
	if len(in.Nodes) == 0 {
		return nil, errors.New("graph.nodes must not be empty")
	}
	ids := map[uint32]struct{}{}
	max := uint32(0)
	for _, n := range in.Nodes {
		if _, ok := ids[n.ID]; ok {
			return nil, fmt.Errorf("duplicate node id: %d", n.ID)
		}
		ids[n.ID] = struct{}{}
		if n.ID > max {
			max = n.ID
		}
	}
	if int(max)+1 != len(in.Nodes) {
		return nil, errors.New("node ids must be contiguous from 0")
	}
	g := core.NewAdjacencyGraph(len(in.Nodes), in.Directed)
	for _, n := range in.Nodes {
		if (n.X == nil) != (n.Y == nil) {
			return nil, fmt.Errorf("node %d must provide both x and y", n.ID)
		}
		if n.X != nil {
			if !finite(*n.X) || !finite(*n.Y) {
				return nil, fmt.Errorf("node %d coordinates must be finite", n.ID)
			}
			_ = g.SetPosition(core.NodeID(n.ID), core.Point{X: *n.X, Y: *n.Y})
		}
	}
	for _, e := range in.Edges {
		if _, ok := ids[e.From]; !ok {
			return nil, fmt.Errorf("edge references missing node %d", e.From)
		}
		if _, ok := ids[e.To]; !ok {
			return nil, fmt.Errorf("edge references missing node %d", e.To)
		}
		if !finite(e.Weight) || e.Weight <= 0 {
			return nil, fmt.Errorf("edge weight must be finite and positive")
		}
		if err := g.AddEdge(core.NodeID(e.From), core.NodeID(e.To), e.Weight); err != nil {
			return nil, err
		}
	}
	// Graph-derived policy and heuristic statistics are immutable setup data.
	// Compute them during graph preparation so TRUSS and ANCHOR do not rescan
	// the full graph for every route.
	g.PrepareAnalysisProfile()
	return g, nil
}
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
