package anchor

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

type Config struct {
	TargetWorkRatio        float64
	InitialPathBudgetRatio float64
	MinQualityBudgetRatio  float64
	MaxCorridors           int
	BaseWidthScale         float64
	RepairHops             int
	MaxRepairNodesRatio    float64
	HubCount               int
	ConnectorBudgetRatio   float64
	WeightedAStarFactor    float64
	InitialExpandRatio     float64
	ExtensionExpandRatios  []float64
	StallWindow            uint64
	ConnectorExpandRatio   float64
	MaxConnectorCalls      int
}

func DefaultConfig() Config {
	return Config{TargetWorkRatio: .45, InitialPathBudgetRatio: .18, MinQualityBudgetRatio: .06, MaxCorridors: 7, BaseWidthScale: .14, RepairHops: 1, MaxRepairNodesRatio: .22, HubCount: 8, ConnectorBudgetRatio: .16, WeightedAStarFactor: 4.0, InitialExpandRatio: .45, ExtensionExpandRatios: []float64{.75, 1.0, 1.5, 2.0}, StallWindow: 128, ConnectorExpandRatio: .20, MaxConnectorCalls: 3}
}

type Connector interface {
	Name() string
	Solve(context.Context, core.Graph, core.RouteRequest, core.WorkBudget, bearing.Observer) core.RouteResult
}

type Solver struct {
	Config    Config
	Connector Connector
}

func (Solver) Name() string { return "anchor" }

type features struct {
	hasPos                                         bool
	degreeCV, maxMeanDegreeRatio, top1DegreeShare  float64
	weightGeoRatioCV, edgeP95Median, edgeMaxMedian float64
}

func Analyze(g core.Graph) string {
	f := graphFeatures(g)
	if f.top1DegreeShare > .12 || f.maxMeanDegreeRatio > 8 || (!f.hasPos && f.degreeCV > .6) {
		return "hub_aware"
	}
	if f.hasPos && f.weightGeoRatioCV > .35 {
		return "weighted_cost"
	}
	if f.hasPos && f.degreeCV > .9 {
		return "portal"
	}
	if f.hasPos && f.edgeP95Median > 1.85 && f.edgeMaxMedian > 3 {
		return "portal"
	}
	if f.hasPos {
		return "geometric_corridor"
	}
	return "hub_aware"
}

func RecommendedWeight(g core.Graph, mode core.RouteMode) float64 {
	if mode == core.ModeExact {
		return 1.0
	}
	if mode == core.ModeFast {
		return 2.75
	}
	strategy := Analyze(g)
	if mode == core.ModeQuality {
		switch strategy {
		case "weighted_cost", "portal":
			return 1.20
		default:
			return 1.35
		}
	}
	switch strategy {
	case "weighted_cost":
		return 1.35
	case "portal":
		return 1.55
	default:
		return 1.80
	}
}

func graphFeatures(g core.Graph) features {
	n := g.NodeCount()
	if n == 0 {
		return features{}
	}
	deg := make([]float64, n)
	sum := 0.0
	maxd := 0.0
	hasPos := false
	for i := 0; i < n; i++ {
		d := float64(len(g.EdgesFrom(core.NodeID(i))))
		deg[i] = d
		sum += d
		if d > maxd {
			maxd = d
		}
		if _, ok := g.Position(core.NodeID(i)); ok {
			hasPos = true
		}
	}
	mean := sum / float64(n)
	variance := 0.0
	for _, d := range deg {
		x := d - mean
		variance += x * x
	}
	variance /= float64(n)
	cv := math.Sqrt(variance) / math.Max(mean, 1e-12)
	sortedDeg := append([]float64(nil), deg...)
	sort.Sort(sort.Reverse(sort.Float64Slice(sortedDeg)))
	topK := int(.01 * float64(n))
	if topK < 1 {
		topK = 1
	}
	top := 0.0
	for i := 0; i < topK && i < n; i++ {
		top += sortedDeg[i]
	}
	f := features{hasPos: hasPos, degreeCV: cv, maxMeanDegreeRatio: maxd / math.Max(mean, 1e-12), top1DegreeShare: top / math.Max(sum, 1)}
	if !hasPos {
		return f
	}
	ratios := []float64{}
	lens := []float64{}
	maxSamples := 768
	for u := 0; u < n && len(lens) < maxSamples; u++ {
		pu, ok := g.Position(core.NodeID(u))
		if !ok {
			continue
		}
		for _, e := range g.EdgesFrom(core.NodeID(u)) {
			pv, ok := g.Position(e.To)
			if !ok {
				continue
			}
			d := core.Euclidean(pu, pv)
			if d > 1e-12 {
				ratios = append(ratios, e.Weight/d)
				lens = append(lens, d)
			}
			if len(lens) >= maxSamples {
				break
			}
		}
	}
	if len(ratios) > 0 {
		m := 0.0
		for _, x := range ratios {
			m += x
		}
		m /= float64(len(ratios))
		v := 0.0
		for _, x := range ratios {
			q := x - m
			v += q * q
		}
		f.weightGeoRatioCV = math.Sqrt(v/float64(len(ratios))) / math.Max(m, 1e-12)
	}
	if len(lens) > 0 {
		sort.Float64s(lens)
		med := math.Max(lens[len(lens)/2], 1e-12)
		f.edgeP95Median = lens[int(.95*float64(len(lens)-1))] / med
		f.edgeMaxMedian = lens[len(lens)-1] / med
	}
	return f
}

func (s Solver) Solve(ctx context.Context, g core.Graph, r core.RouteRequest, b core.WorkBudget, o bearing.Observer) core.RouteResult {
	started := time.Now()
	sess, err := NewSession(g, r, o)
	if err != nil {
		return core.RouteResult{SolverName: "anchor", TerminationStatus: core.TerminationInvalid, ErrorCode: core.ErrInvalidRequest}
	}
	grant := uint64(^uint64(0) >> 1)
	if b.MaxWork != nil {
		grant = *b.MaxWork
		if sess.metrics.TotalActions >= grant {
			grant = 0
		} else {
			grant -= sess.metrics.TotalActions
		}
	}
	for !sess.Finished() && grant > 0 {
		chunk := uint64(512)
		if grant < chunk {
			chunk = grant
		}
		step := sess.Step(ctx, chunk)
		if step.Consumed == 0 {
			break
		}
		grant -= step.Consumed
		if step.Candidate != nil && r.Mode != core.ModeExact {
			break
		}
	}
	if !sess.Finished() {
		sess.status = core.TerminationUnknownBudget
	}
	res := sess.Result()
	res.TimeMS = float64(time.Since(started).Nanoseconds()) / 1_000_000
	res.Telemetry = map[string]any{"anchor_strategy": "adaptive_fast_path", "graph_profile": Analyze(g), "work_model_version": core.WorkModelVersion, "session_adapter": true, "hypothesis_count": 1, "heuristic_weight": sess.HeuristicScale(), "candidate_update_count": sess.CandidateUpdates(), "max_frontier_size": sess.MaxFrontier()}
	return finalizeTiming(res, "anchor")
}

type qitem struct {
	n   core.NodeID
	pri float64
	seq uint64
}
type queue []qitem

func (q queue) Len() int { return len(q) }
func (q queue) Less(i, j int) bool {
	if q[i].pri != q[j].pri {
		return q[i].pri < q[j].pri
	}
	if q[i].n != q[j].n {
		return q[i].n < q[j].n
	}
	return q[i].seq < q[j].seq
}
func (q queue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q *queue) Push(x any)   { *q = append(*q, x.(qitem)) }
func (q *queue) Pop() any     { a := *q; x := a[len(a)-1]; *q = a[:len(a)-1]; return x }

func finalizeTiming(result core.RouteResult, component string) core.RouteResult {
	if result.TimeBreakdown.TotalMS == 0 {
		result.TimeBreakdown.TotalMS = result.TimeMS
		result.TimeBreakdown.SolverMS = result.TimeMS
		result.TimeBreakdown.TotalNS = int64(result.TimeMS * 1_000_000)
		result.TimeBreakdown.SolverNS = result.TimeBreakdown.TotalNS
		if component == "anchor" {
			result.TimeBreakdown.AnchorMS = result.TimeMS
			result.TimeBreakdown.AnchorNS = result.TimeBreakdown.TotalNS
		}
		if component == "bolts" {
			result.TimeBreakdown.BoltsMS = result.TimeMS
			result.TimeBreakdown.BoltsNS = result.TimeBreakdown.TotalNS
		}
	}
	return result
}
