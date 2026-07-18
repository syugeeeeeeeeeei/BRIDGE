package core

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

type NodeID uint32

type Point struct{ X, Y float64 }
type Edge struct {
	To     NodeID
	Weight float64
}

type Graph interface {
	NodeCount() int
	EdgeCount() int
	EdgesFrom(NodeID) []Edge
	HasNode(NodeID) bool
	Position(NodeID) (Point, bool)
	Directed() bool
}

// GraphAnalysisProfile contains immutable, graph-derived statistics that are
// useful to route policies and admissible geometric heuristics. It belongs to
// graph preparation, not to an individual route execution.
type GraphAnalysisProfile struct {
	HasPosition        bool
	DegreeCV           float64
	MaxMeanDegreeRatio float64
	TopOneDegreeShare  float64
	WeightGeoRatioCV   float64
	EdgeP95Median      float64
	EdgeMaxMedian      float64
	HeuristicUnitScale float64
}

// GraphAnalysisProvider is implemented by prepared graphs that cache immutable
// analysis data. Callers must fall back to direct analysis for other Graph
// implementations.
type GraphAnalysisProvider interface {
	GraphAnalysisProfile() GraphAnalysisProfile
}

type AdjacencyGraph struct {
	Adj        [][]Edge
	Pos        []Point
	HasPos     []bool
	IsDirected bool
	edges      int
	analysis   GraphAnalysisProfile
	analysisOK bool
}

func NewAdjacencyGraph(nodes int, directed bool) *AdjacencyGraph {
	return &AdjacencyGraph{Adj: make([][]Edge, nodes), Pos: make([]Point, nodes), HasPos: make([]bool, nodes), IsDirected: directed}
}
func (g *AdjacencyGraph) AddEdge(from, to NodeID, weight float64) error {
	g.analysisOK = false
	if weight < 0 || math.IsNaN(weight) || math.IsInf(weight, 0) {
		return errors.New("edge weight must be finite and non-negative")
	}
	if !g.HasNode(from) || !g.HasNode(to) {
		return fmt.Errorf("node out of range: %d -> %d", from, to)
	}
	g.Adj[from] = insertEdgeCanonical(g.Adj[from], Edge{To: to, Weight: weight})
	g.edges++
	if !g.IsDirected {
		g.Adj[to] = insertEdgeCanonical(g.Adj[to], Edge{To: from, Weight: weight})
	}
	return nil
}

// insertEdgeCanonical keeps every adjacency list ordered by destination and then
// weight. Search algorithms therefore observe the same edge order regardless of
// the order in which callers constructed the graph.
func insertEdgeCanonical(edges []Edge, edge Edge) []Edge {
	i := sort.Search(len(edges), func(i int) bool {
		if edges[i].To != edge.To {
			return edges[i].To > edge.To
		}
		return edges[i].Weight >= edge.Weight
	})
	edges = append(edges, Edge{})
	copy(edges[i+1:], edges[i:])
	edges[i] = edge
	return edges
}

// PrepareAnalysisProfile computes and caches immutable graph statistics. It is
// intended to be called while the graph is being prepared so route execution
// does not rescan the full graph.
func (g *AdjacencyGraph) PrepareAnalysisProfile() GraphAnalysisProfile {
	if g.analysisOK {
		return g.analysis
	}
	n := g.NodeCount()
	if n == 0 {
		g.analysis = GraphAnalysisProfile{}
		g.analysisOK = true
		return g.analysis
	}
	degrees := make([]float64, n)
	sumDegree := 0.0
	maxDegree := 0.0
	hasPosition := false
	minGeoRatio := math.Inf(1)
	ratios := make([]float64, 0, 768)
	lengths := make([]float64, 0, 768)
	for u := 0; u < n; u++ {
		edges := g.EdgesFrom(NodeID(u))
		d := float64(len(edges))
		degrees[u] = d
		sumDegree += d
		if d > maxDegree {
			maxDegree = d
		}
		pu, positioned := g.Position(NodeID(u))
		if positioned {
			hasPosition = true
		}
		if !positioned {
			continue
		}
		for _, e := range edges {
			pv, ok := g.Position(e.To)
			if !ok {
				continue
			}
			distance := math.Hypot(pu.X-pv.X, pu.Y-pv.Y)
			if distance <= 1e-12 {
				continue
			}
			ratio := e.Weight / distance
			if ratio < minGeoRatio {
				minGeoRatio = ratio
			}
			if len(lengths) < 768 {
				ratios = append(ratios, ratio)
				lengths = append(lengths, distance)
			}
		}
	}
	meanDegree := sumDegree / float64(n)
	variance := 0.0
	for _, d := range degrees {
		delta := d - meanDegree
		variance += delta * delta
	}
	variance /= float64(n)
	sortedDegrees := append([]float64(nil), degrees...)
	sort.Sort(sort.Reverse(sort.Float64Slice(sortedDegrees)))
	topK := int(0.01 * float64(n))
	if topK < 1 {
		topK = 1
	}
	topDegree := 0.0
	for i := 0; i < topK && i < len(sortedDegrees); i++ {
		topDegree += sortedDegrees[i]
	}
	profile := GraphAnalysisProfile{
		HasPosition:        hasPosition,
		DegreeCV:           math.Sqrt(variance) / math.Max(meanDegree, 1e-12),
		MaxMeanDegreeRatio: maxDegree / math.Max(meanDegree, 1e-12),
		TopOneDegreeShare:  topDegree / math.Max(sumDegree, 1),
	}
	if !math.IsInf(minGeoRatio, 1) {
		profile.HeuristicUnitScale = minGeoRatio
	}
	if len(ratios) > 0 {
		mean := 0.0
		for _, ratio := range ratios {
			mean += ratio
		}
		mean /= float64(len(ratios))
		v := 0.0
		for _, ratio := range ratios {
			delta := ratio - mean
			v += delta * delta
		}
		profile.WeightGeoRatioCV = math.Sqrt(v/float64(len(ratios))) / math.Max(mean, 1e-12)
	}
	if len(lengths) > 0 {
		sort.Float64s(lengths)
		median := math.Max(lengths[len(lengths)/2], 1e-12)
		profile.EdgeP95Median = lengths[int(0.95*float64(len(lengths)-1))] / median
		profile.EdgeMaxMedian = lengths[len(lengths)-1] / median
	}
	g.analysis = profile
	g.analysisOK = true
	return profile
}

func (g *AdjacencyGraph) GraphAnalysisProfile() GraphAnalysisProfile {
	return g.PrepareAnalysisProfile()
}

func (g *AdjacencyGraph) SetPosition(n NodeID, p Point) error {
	g.analysisOK = false
	if !g.HasNode(n) {
		return fmt.Errorf("node out of range: %d", n)
	}
	g.Pos[n] = p
	g.HasPos[n] = true
	return nil
}
func (g *AdjacencyGraph) NodeCount() int { return len(g.Adj) }
func (g *AdjacencyGraph) EdgeCount() int { return g.edges }
func (g *AdjacencyGraph) EdgesFrom(n NodeID) []Edge {
	if !g.HasNode(n) {
		return nil
	}
	return g.Adj[n]
}
func (g *AdjacencyGraph) HasNode(n NodeID) bool { return uint64(n) < uint64(len(g.Adj)) }
func (g *AdjacencyGraph) Position(n NodeID) (Point, bool) {
	if !g.HasNode(n) || !g.HasPos[n] {
		return Point{}, false
	}
	return g.Pos[n], true
}
func (g *AdjacencyGraph) Directed() bool { return g.IsDirected }
