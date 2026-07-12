package traffic

import (
	"math"
	"math/rand"
	"runtime"
	"sort"
	"time"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

// EnvironmentMetadata records the execution environment required to interpret a benchmark artifact.
type EnvironmentMetadata struct {
	GoVersion  string `json:"go_version"`
	GOOS       string `json:"goos"`
	GOARCH     string `json:"goarch"`
	CPUs       int    `json:"cpus"`
	CapturedAt string `json:"captured_at"`
}

// GraphMetadata describes the generated graph instance without exposing solver-private state.
type GraphMetadata struct {
	Generator string           `json:"generator"`
	Topology  string           `json:"topology,omitempty"`
	Seed      int64            `json:"seed"`
	Nodes     int              `json:"actual_node_count"`
	Edges     int              `json:"edge_count"`
	Directed  bool             `json:"directed"`
	Dataset   *DatasetMetadata `json:"dataset,omitempty"`
}

// QueryMetadata identifies one source-target query within a graph instance.
type QueryMetadata struct {
	Strategy string      `json:"query_selection_method"`
	Source   core.NodeID `json:"source"`
	Target   core.NodeID `json:"target"`
}

// QualityMetadata records result-quality facts that are independent of timing aggregation.
type QualityMetadata struct {
	Found            bool     `json:"path_found"`
	SearchCompleted  bool     `json:"search_completed"`
	Reachability     bool     `json:"reachability_proven"`
	Exact            bool     `json:"optimality_proven"`
	LowerBound       *float64 `json:"lower_bound,omitempty"`
	CertifiedRatio   *float64 `json:"proven_cost_ratio,omitempty"`
	QualityCertified bool     `json:"quality_bound_proven"`
}

// SummaryStatistics is recomputable from raw scalar observations.
type SummaryStatistics struct {
	Count     int     `json:"count"`
	Mean      float64 `json:"mean"`
	StdDev    float64 `json:"stddev"`
	Min       float64 `json:"min"`
	P50       float64 `json:"p50"`
	P95       float64 `json:"p95"`
	Max       float64 `json:"max"`
	CI95Lower float64 `json:"ci95_lower"`
	CI95Upper float64 `json:"ci95_upper"`
}

func summarizeValues(values []float64) SummaryStatistics {
	if len(values) == 0 {
		return SummaryStatistics{}
	}
	x := append([]float64(nil), values...)
	sort.Float64s(x)
	sum := 0.0
	for _, v := range x {
		sum += v
	}
	mean := sum / float64(len(x))
	variance := 0.0
	if len(x) > 1 {
		for _, v := range x {
			d := v - mean
			variance += d * d
		}
		variance /= float64(len(x) - 1)
	}
	sd := math.Sqrt(variance)
	margin := 0.0
	if len(x) > 1 {
		margin = 1.96 * sd / math.Sqrt(float64(len(x)))
	}
	return SummaryStatistics{
		Count: len(x), Mean: mean, StdDev: sd, Min: x[0],
		P50: percentile(x, 0.50), P95: percentile(x, 0.95), Max: x[len(x)-1],
		CI95Lower: mean - margin, CI95Upper: mean + margin,
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	pos := p * float64(len(sorted)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return sorted[lo]
	}
	f := pos - float64(lo)
	return sorted[lo]*(1-f) + sorted[hi]*f
}

func captureEnvironment() *EnvironmentMetadata {
	return &EnvironmentMetadata{GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, CPUs: runtime.NumCPU(), CapturedAt: time.Now().UTC().Format(time.RFC3339Nano)}
}

func effectiveQueries(c ScenarioCase) []QuerySpec {
	if len(c.Queries) > 0 {
		return c.Queries
	}
	return []QuerySpec{{ID: "default", Strategy: c.Endpoints.Strategy, Source: c.Endpoints.Source, Target: c.Endpoints.Target}}
}

type runPlan struct {
	Scenario   ScenarioCase
	Algorithm  string
	Seed       int64
	Repetition int
	Warmup     bool
	Query      QuerySpec
}

func expandRunPlans(s BenchmarkScenario) []runPlan {
	plans := make([]runPlan, 0)
	for _, c := range s.Scenarios {
		for _, a := range s.Algorithms {
			for _, seed := range s.Execution.Seeds {
				for _, q := range effectiveQueries(c) {
					for w := 1; w <= s.Execution.WarmupRuns; w++ {
						plans = append(plans, runPlan{c, a, seed, w, true, q})
					}
					for rep := 1; rep <= s.Execution.Repetitions; rep++ {
						plans = append(plans, runPlan{c, a, seed, rep, false, q})
					}
				}
			}
		}
	}
	if s.Execution.RandomizeOrder {
		r := rand.New(rand.NewSource(stablePlanSeed(s.Execution.Seeds)))
		r.Shuffle(len(plans), func(i, j int) { plans[i], plans[j] = plans[j], plans[i] })
	}
	return plans
}

func stablePlanSeed(seeds []int64) int64 {
	var out int64 = 1469598103934665603
	for _, v := range seeds {
		out ^= v
		out *= 1099511628211
	}
	return out
}
