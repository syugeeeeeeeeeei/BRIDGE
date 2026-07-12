package bolts

import "fmt"

func Resolve(id string) (Solver, error) {
	switch id {
	case "dijkstra":
		return Dijkstra{}, nil
	case "bidirectional_dijkstra":
		return BidirectionalDijkstra{}, nil
	case "astar":
		return AStar{}, nil
	case "weighted_astar":
		return WeightedAStar{ID: "weighted_astar", Weight: 1.12}, nil
	case "reachability":
		return Reachability{}, nil
	default:
		return nil, fmt.Errorf("unknown BOLTS solver %q", id)
	}
}

func SupportedSolverIDs() []string {
	return []string{
		"astar",
		"bidirectional_dijkstra",
		"dijkstra",
		"reachability",
		"weighted_astar",
	}
}
