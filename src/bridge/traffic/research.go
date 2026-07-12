package traffic

import (
	"math"
	"sort"
)

// ResearchRow is a language-neutral benchmark observation used for migration studies.
type ResearchRow struct {
	Implementation string  `json:"implementation"`
	Topology       string  `json:"topology"`
	Nodes          int     `json:"requested_node_count"`
	Seed           int64   `json:"seed"`
	Mode           string  `json:"route_mode"`
	Found          bool    `json:"path_found"`
	Distance       float64 `json:"path_cost"`
	ExactDistance  float64 `json:"exact_distance"`
	DistanceRatio  float64 `json:"cost_ratio_to_exact_reference"`
	ExactMatch     bool    `json:"matches_exact_reference"`
	TotalWork      uint64  `json:"total_work"`
	ScheduledSteps uint64  `json:"scheduled_steps"`
	TimeMS         float64 `json:"end_to_end_time_ms"`
}

type ReadinessThresholds struct {
	ValidPathRateMin       float64 `json:"valid_path_rate_min"`
	ConnectedFoundRateMin  float64 `json:"connected_found_rate_min"`
	MeanDistanceRatioMax   float64 `json:"mean_distance_ratio_max"`
	P95DistanceRatioMax    float64 `json:"p95_distance_ratio_max"`
	WorstDistanceRatioMax  float64 `json:"worst_distance_ratio_max"`
	ExactModeExactRateMin  float64 `json:"exact_mode_exact_rate_min"`
	BudgetViolationRateMax float64 `json:"budget_violation_rate_max"`
	RepeatabilityRateMin   float64 `json:"repeatability_rate_min"`
	TrendCorrelationMin    float64 `json:"trend_correlation_min"`
	TopologyCoverageMin    float64 `json:"topology_coverage_min"`
}

func DefaultReadinessThresholds() ReadinessThresholds {
	return ReadinessThresholds{
		ValidPathRateMin:       1.0,
		ConnectedFoundRateMin:  0.99,
		MeanDistanceRatioMax:   1.05,
		P95DistanceRatioMax:    1.15,
		WorstDistanceRatioMax:  1.35,
		ExactModeExactRateMin:  1.0,
		BudgetViolationRateMax: 0.0,
		RepeatabilityRateMin:   1.0,
		TrendCorrelationMin:    0.70,
		TopologyCoverageMin:    0.90,
	}
}

// Spearman computes rank correlation. Ties receive average ranks.
func Spearman(a, b []float64) float64 {
	if len(a) != len(b) || len(a) < 2 {
		return math.NaN()
	}
	ra, rb := ranks(a), ranks(b)
	var ma, mb float64
	for i := range ra {
		ma += ra[i]
		mb += rb[i]
	}
	ma /= float64(len(ra))
	mb /= float64(len(rb))
	var num, da, db float64
	for i := range ra {
		xa, xb := ra[i]-ma, rb[i]-mb
		num += xa * xb
		da += xa * xa
		db += xb * xb
	}
	if da == 0 || db == 0 {
		return 1
	}
	return num / math.Sqrt(da*db)
}

func ranks(v []float64) []float64 {
	idx := make([]int, len(v))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(i, j int) bool { return v[idx[i]] < v[idx[j]] })
	out := make([]float64, len(v))
	for i := 0; i < len(idx); {
		j := i + 1
		for j < len(idx) && v[idx[j]] == v[idx[i]] {
			j++
		}
		rank := (float64(i+1) + float64(j)) / 2
		for k := i; k < j; k++ {
			out[idx[k]] = rank
		}
		i = j
	}
	return out
}
