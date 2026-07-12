package traffic

import (
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"math"
	"math/rand"
)

func Grid(width, height int, seed int64) (*core.AdjacencyGraph, error) {
	if width < 1 || height < 1 {
		return nil, fmt.Errorf("invalid grid size")
	}
	g := core.NewAdjacencyGraph(width*height, false)
	rng := rand.New(rand.NewSource(seed))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			u := core.NodeID(y*width + x)
			_ = g.SetPosition(u, core.Point{X: float64(x), Y: float64(y)})
			if x+1 < width {
				_ = g.AddEdge(u, u+1, 1+rng.Float64()*.05)
			}
			if y+1 < height {
				_ = g.AddEdge(u, u+core.NodeID(width), 1+rng.Float64()*.05)
			}
		}
	}
	return g, nil
}

// GridNodes generates a connected grid-shaped graph with exactly nodes vertices.
// The final row may be shorter than the preceding rows.
func GridNodes(nodes int, seed int64) (*core.AdjacencyGraph, error) {
	if nodes < 1 {
		return nil, fmt.Errorf("invalid node count")
	}
	width := int(math.Ceil(math.Sqrt(float64(nodes))))
	g := core.NewAdjacencyGraph(nodes, false)
	rng := rand.New(rand.NewSource(seed))
	for i := 0; i < nodes; i++ {
		x, y := i%width, i/width
		u := core.NodeID(i)
		_ = g.SetPosition(u, core.Point{X: float64(x), Y: float64(y)})
		if x > 0 {
			_ = g.AddEdge(u, u-1, 1+rng.Float64()*.05)
		}
		if i-width >= 0 {
			_ = g.AddEdge(u, core.NodeID(i-width), 1+rng.Float64()*.05)
		}
	}
	return g, nil
}
