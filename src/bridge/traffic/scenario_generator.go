package traffic

import (
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"math"
)

func BuildScenarioGraph(spec GeneratorSpec, seed int64) (*core.AdjacencyGraph, error) {
	g, _, _, err := BuildScenarioGraphAndEndpoints(spec, seed)
	return g, err
}

func BuildScenarioGraphAndEndpoints(spec GeneratorSpec, seed int64) (*core.AdjacencyGraph, core.NodeID, core.NodeID, error) {
	switch spec.Generator {
	case "", "grid":
		if spec.Topology == "" || spec.Topology == string(TopologyOpen) {
			if spec.Nodes > 0 {
				g, err := GridNodes(spec.Nodes, seed)
				if err != nil {
					return nil, 0, 0, err
				}
				return g, 0, core.NodeID(g.NodeCount() - 1), nil
			}
			g, err := Grid(spec.Width, spec.Height, seed)
			if err != nil {
				return nil, 0, 0, err
			}
			return g, 0, core.NodeID(g.NodeCount() - 1), nil
		}
		noise := spec.Noise
		if noise == 0 {
			noise = 0.05
		}
		return TopologyGrid(spec.Nodes, GridTopology(spec.Topology), seed, noise)
	case "random_geometric":
		k := spec.K
		if k == 0 {
			k = 8
		}
		return RandomGeometric(spec.Nodes, k, seed)
	case "community":
		return CommunityGraph(spec.Nodes, spec.Communities, seed)
	case "maze":
		return MazeGraph(spec.Nodes, seed)
	case "adversarial":
		return AdversarialGraph(spec.Nodes, seed)
	case "dataset":
		d, err := LoadDataset(spec.DatasetPath)
		if err != nil {
			return nil, 0, 0, err
		}
		return d.Graph, d.Source, d.Target, nil
	default:
		return nil, 0, 0, fmt.Errorf("unsupported generator %q", spec.Generator)
	}
}

func validateGridGraphSpec(id string, spec GeneratorSpec) error {
	switch spec.Topology {
	case "", "open":
		hasNodes := spec.Nodes > 0
		hasDimensions := spec.Width > 0 || spec.Height > 0
		if hasNodes && hasDimensions {
			return fmt.Errorf("scenario %q: use graph.requested_node_count or width/height, not both", id)
		}
		if !hasNodes && (spec.Width < 1 || spec.Height < 1) {
			return fmt.Errorf("scenario %q: graph.requested_node_count must be >= 2 or both width and height provided", id)
		}
		if hasNodes && spec.Nodes < 2 {
			return fmt.Errorf("scenario %q: graph.requested_node_count must be >= 2", id)
		}
	default:
		switch spec.Topology {
		case "wall", "u_shape", "culdesac", "disconnected":
		default:
			return fmt.Errorf("scenario %q: unsupported topology %q", id, spec.Topology)
		}
		if spec.Nodes < 2 {
			return fmt.Errorf("scenario %q: graph.requested_node_count must be >= 2", id)
		}
		if spec.Width > 0 || spec.Height > 0 {
			return fmt.Errorf("scenario %q: non-open grid topology requires graph.requested_node_count and does not support width/height", id)
		}
	}
	return nil
}

func validateRandomGeometricSpec(id string, spec GeneratorSpec) error {
	if spec.Nodes < 2 {
		return fmt.Errorf("scenario %q: graph.requested_node_count must be >= 2", id)
	}
	if spec.Width > 0 || spec.Height > 0 {
		return fmt.Errorf("scenario %q: random_geometric does not support width/height", id)
	}
	if spec.Topology != "" && spec.Topology != "open" {
		return fmt.Errorf("scenario %q: random_geometric does not support topology %q", id, spec.Topology)
	}
	if spec.K < 0 {
		return fmt.Errorf("scenario %q: graph.neighbor_candidate_count must be >= 0", id)
	}
	return nil
}

func scenarioNodeCount(spec GeneratorSpec) (int, error) {
	switch spec.Generator {
	case "", "grid":
		if spec.Nodes > 0 {
			if spec.Topology == "" || spec.Topology == string(TopologyOpen) {
				return spec.Nodes, nil
			}
			side := int(math.Sqrt(float64(spec.Nodes)))
			if side < 10 {
				side = 10
			}
			return side * side, nil
		}
		if spec.Width > 0 && spec.Height > 0 {
			return spec.Width * spec.Height, nil
		}
	case "random_geometric", "community", "maze", "adversarial":
		return spec.Nodes, nil
	case "dataset":
		d, err := LoadDataset(spec.DatasetPath)
		if err != nil {
			return 0, err
		}
		return d.Graph.NodeCount(), nil
	}
	return 0, fmt.Errorf("could not determine graph node count")
}
