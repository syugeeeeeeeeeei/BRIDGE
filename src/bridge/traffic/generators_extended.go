package traffic

import (
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"math"
	"math/rand"
	"sort"
)

type GridTopology string

const (
	TopologyOpen         GridTopology = "open"
	TopologyWall         GridTopology = "wall"
	TopologyUShape       GridTopology = "u_shape"
	TopologyCulDeSac     GridTopology = "culdesac"
	TopologyDisconnected GridTopology = "disconnected"
)

// TopologyGrid mirrors the legacy Python quantitative grid families.
func TopologyGrid(nodes int, topology GridTopology, seed int64, noise float64) (*core.AdjacencyGraph, core.NodeID, core.NodeID, error) {
	side := int(math.Sqrt(float64(nodes)))
	if side < 10 {
		side = 10
	}
	n := side * side
	blocked := make([]bool, n)
	block := func(x, y int) {
		if x >= 0 && x < side && y >= 0 && y < side {
			blocked[y*side+x] = true
		}
	}
	switch topology {
	case TopologyOpen:
	case TopologyWall:
		x, gap := side/2, maxInt(1, side/10)
		for y := 0; y < side-gap; y++ {
			block(x, y)
		}
	case TopologyUShape:
		x0, x1, y0, y1 := side/3, 2*side/3, side/4, 3*side/4
		for y := y0; y < y1; y++ {
			block(x0, y)
			block(x1, y)
		}
		for x := x0; x <= x1; x++ {
			block(x, y1)
		}
	case TopologyCulDeSac:
		x0, x1, y0, y1 := side/3, 2*side/3, side/4, 3*side/4
		for y := y0; y < y1; y++ {
			block(x0, y)
			block(x1, y)
		}
		for x := x0; x <= x1; x++ {
			block(x, y0)
		}
	case TopologyDisconnected:
		x := side / 2
		for y := 0; y < side; y++ {
			block(x, y)
		}
	default:
		return nil, 0, 0, fmt.Errorf("unknown topology %q", topology)
	}
	g := core.NewAdjacencyGraph(n, false)
	first, last := -1, -1
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			id := y*side + x
			if blocked[id] {
				continue
			}
			_ = g.SetPosition(core.NodeID(id), core.Point{X: float64(x), Y: float64(y)})
			if first < 0 {
				first = id
			}
			last = id
		}
	}
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			u := y*side + x
			if blocked[u] {
				continue
			}
			for _, d := range [][2]int{{1, 0}, {0, 1}} {
				nx, ny := x+d[0], y+d[1]
				if nx >= side || ny >= side {
					continue
				}
				v := ny*side + nx
				if blocked[v] {
					continue
				}
				_ = g.AddEdge(core.NodeID(u), core.NodeID(v), 1+stableNoise(seed, u, v)*noise)
			}
		}
	}
	if first < 0 {
		return nil, 0, 0, fmt.Errorf("empty topology")
	}
	return g, core.NodeID(first), core.NodeID(last), nil
}

// RandomGeometric builds a deterministic k-nearest-neighbour geometric graph.
func RandomGeometric(nodes, k int, seed int64) (*core.AdjacencyGraph, core.NodeID, core.NodeID, error) {
	if nodes < 2 {
		return nil, 0, 0, fmt.Errorf("nodes must be >=2")
	}
	if k < 1 {
		k = 1
	}
	if k >= nodes {
		k = nodes - 1
	}
	rng := rand.New(rand.NewSource(seed))
	pts := make([]core.Point, nodes)
	g := core.NewAdjacencyGraph(nodes, false)
	for i := range pts {
		pts[i] = core.Point{X: rng.Float64(), Y: rng.Float64()}
		_ = g.SetPosition(core.NodeID(i), pts[i])
	}
	seen := map[[2]int]bool{}
	type cand struct {
		j int
		d float64
	}
	for i := 0; i < nodes; i++ {
		cs := make([]cand, 0, nodes-1)
		for j := 0; j < nodes; j++ {
			if i != j {
				cs = append(cs, cand{j, core.Euclidean(pts[i], pts[j])})
			}
		}
		sort.Slice(cs, func(a, b int) bool {
			if cs[a].d == cs[b].d {
				return cs[a].j < cs[b].j
			}
			return cs[a].d < cs[b].d
		})
		for _, c := range cs[:k] {
			a, b := i, c.j
			if a > b {
				a, b = b, a
			}
			key := [2]int{a, b}
			if seen[key] {
				continue
			}
			seen[key] = true
			_ = g.AddEdge(core.NodeID(a), core.NodeID(b), c.d)
		}
	}
	s, t := 0, 0
	minSum, maxSum := math.Inf(1), math.Inf(-1)
	for i, p := range pts {
		z := p.X + p.Y
		if z < minSum {
			minSum = z
			s = i
		}
		if z > maxSum {
			maxSum = z
			t = i
		}
	}
	return g, core.NodeID(s), core.NodeID(t), nil
}

func stableNoise(seed int64, u, v int) float64 {
	x := uint64(seed) ^ (uint64(uint32(u)) << 32) ^ uint64(uint32(v)) ^ 0x9e3779b97f4a7c15
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	x ^= x >> 31
	return float64(x>>11) / float64(uint64(1)<<53)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
