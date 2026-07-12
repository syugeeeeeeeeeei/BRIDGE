package traffic

import (
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"math"
	"math/rand"
)

// CommunityGraph creates dense deterministic communities connected by sparse bridges.
func CommunityGraph(nodes, communities int, seed int64) (*core.AdjacencyGraph, core.NodeID, core.NodeID, error) {
	if nodes < 4 {
		return nil, 0, 0, fmt.Errorf("nodes must be >=4")
	}
	if communities == 0 {
		communities = int(math.Max(2, math.Round(math.Sqrt(float64(nodes))/2)))
	}
	if communities < 2 || communities > nodes {
		return nil, 0, 0, fmt.Errorf("communities must be between 2 and nodes")
	}
	g := core.NewAdjacencyGraph(nodes, false)
	rng := rand.New(rand.NewSource(seed))
	starts := make([]int, communities+1)
	for c := 0; c <= communities; c++ {
		starts[c] = c * nodes / communities
	}
	for c := 0; c < communities; c++ {
		start, end := starts[c], starts[c+1]
		for i := start; i < end; i++ {
			angle := 2 * math.Pi * float64(i-start) / math.Max(1, float64(end-start))
			_ = g.SetPosition(core.NodeID(i), core.Point{X: float64(c)*3 + math.Cos(angle), Y: math.Sin(angle)})
			if i+1 < end {
				_ = g.AddEdge(core.NodeID(i), core.NodeID(i+1), 1+rng.Float64()*.05)
			}
		}
		if end-start > 2 {
			_ = g.AddEdge(core.NodeID(start), core.NodeID(end-1), 1+rng.Float64()*.05)
		}
		for i := start; i < end; i++ {
			for j := i + 2; j < end && j <= i+3; j++ {
				_ = g.AddEdge(core.NodeID(i), core.NodeID(j), 1.1+rng.Float64()*.1)
			}
		}
	}
	for c := 0; c+1 < communities; c++ {
		_ = g.AddEdge(core.NodeID(starts[c+1]-1), core.NodeID(starts[c+1]), 2+rng.Float64()*.1)
	}
	return g, 0, core.NodeID(nodes - 1), nil
}

// MazeGraph creates a deterministic perfect maze using randomized depth-first carving.
func MazeGraph(nodes int, seed int64) (*core.AdjacencyGraph, core.NodeID, core.NodeID, error) {
	if nodes < 4 {
		return nil, 0, 0, fmt.Errorf("nodes must be >=4")
	}
	width := int(math.Ceil(math.Sqrt(float64(nodes))))
	g := core.NewAdjacencyGraph(nodes, false)
	for i := 0; i < nodes; i++ {
		_ = g.SetPosition(core.NodeID(i), core.Point{X: float64(i % width), Y: float64(i / width)})
	}
	rng := rand.New(rand.NewSource(seed))
	seen := make([]bool, nodes)
	stack := []int{0}
	seen[0] = true
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for len(stack) > 0 {
		u := stack[len(stack)-1]
		x, y := u%width, u/width
		candidates := make([]int, 0, 4)
		for _, d := range dirs {
			nx, ny := x+d[0], y+d[1]
			v := ny*width + nx
			if nx >= 0 && nx < width && ny >= 0 && v >= 0 && v < nodes && !seen[v] {
				candidates = append(candidates, v)
			}
		}
		if len(candidates) == 0 {
			stack = stack[:len(stack)-1]
			continue
		}
		v := candidates[rng.Intn(len(candidates))]
		_ = g.AddEdge(core.NodeID(u), core.NodeID(v), 1)
		seen[v] = true
		stack = append(stack, v)
	}
	return g, 0, core.NodeID(nodes - 1), nil
}

// AdversarialGraph creates a main chain with cheap-looking dead ends and costly shortcuts.
func AdversarialGraph(nodes int, seed int64) (*core.AdjacencyGraph, core.NodeID, core.NodeID, error) {
	if nodes < 4 {
		return nil, 0, 0, fmt.Errorf("nodes must be >=4")
	}
	g := core.NewAdjacencyGraph(nodes, false)
	rng := rand.New(rand.NewSource(seed))
	backbone := (nodes + 1) / 2
	for i := 0; i < nodes; i++ {
		_ = g.SetPosition(core.NodeID(i), core.Point{X: float64(i % backbone), Y: float64(i / backbone)})
	}
	for i := 0; i+1 < backbone; i++ {
		_ = g.AddEdge(core.NodeID(i), core.NodeID(i+1), 1+rng.Float64()*.02)
	}
	for i := backbone; i < nodes; i++ {
		attach := (i - backbone) % (backbone - 1)
		_ = g.AddEdge(core.NodeID(attach), core.NodeID(i), .2+rng.Float64()*.02)
	}
	for i := 0; i+3 < backbone; i += 3 {
		_ = g.AddEdge(core.NodeID(i), core.NodeID(i+3), 4+rng.Float64())
	}
	return g, 0, core.NodeID(backbone - 1), nil
}
