package healthy

import (
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
	"math"
	"sort"
)

func pairKey(r traffic.BenchmarkRun) string {
	return fmt.Sprintf("%s|%s|%s|%d|%d", r.RunMetadata.ScenarioID, r.GraphProfile.GraphInstanceID, r.QueryProfile.QueryID, r.RunMetadata.ExecutionSeed, r.RunMetadata.RepetitionIndex)
}
func ratio(a, b float64) *float64 {
	if b == 0 {
		return nil
	}
	v := a / b
	return &v
}
func Pair(runs []traffic.BenchmarkRun, p HealthProfile, valid map[string]bool) []PairedComparison {
	if p.PerformanceReferenceAlgorithm == "" {
		return nil
	}
	cand := map[string]traffic.BenchmarkRun{}
	ref := map[string]traffic.BenchmarkRun{}
	for _, r := range runs {
		if r.RunMetadata.WarmupRun {
			continue
		}
		k := pairKey(r)
		if r.RunMetadata.AlgorithmID == p.CandidateAlgorithm {
			cand[k] = r
		}
		if r.RunMetadata.AlgorithmID == p.PerformanceReferenceAlgorithm {
			ref[k] = r
		}
	}
	keys := make([]string, 0, len(cand))
	for k := range cand {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := []PairedComparison{}
	for _, k := range keys {
		c := cand[k]
		r, ok := ref[k]
		if !ok || !valid[c.RunMetadata.RunID] || !valid[r.RunMetadata.RunID] {
			continue
		}
		pc := PairedComparison{PairKey: k, CandidateRunID: c.RunMetadata.RunID, ReferenceRunID: r.RunMetadata.RunID, WorkRatio: ratio(float64(c.Measurement.Work.TotalActions), float64(r.Measurement.Work.TotalActions)), LogicalStepRatio: ratio(float64(c.Measurement.Work.LogicalSteps), float64(r.Measurement.Work.LogicalSteps)), ScheduledStepRatio: ratio(float64(c.Measurement.Work.ScheduledSteps), float64(r.Measurement.Work.ScheduledSteps)), SolverTimeRatio: ratio(c.Measurement.SolverTimeMS, r.Measurement.SolverTimeMS), EndToEndTimeRatio: ratio(c.Measurement.EndToEndTimeMS, r.Measurement.EndToEndTimeMS), AllocBytesRatio: ratio(float64(c.Measurement.SystemMetrics.AllocBytes), float64(r.Measurement.SystemMetrics.AllocBytes))}
		if c.ExecutionResult.PathCost != nil && r.ExecutionResult.PathCost != nil {
			pc.DistanceRatio = ratio(*c.ExecutionResult.PathCost, *r.ExecutionResult.PathCost)
		}
		out = append(out, pc)
	}
	return out
}
func summarize(vs []RunValidation) Summary {
	s := Summary{Runs: len(vs)}
	for _, v := range vs {
		if v.Status == StatusPass {
			s.ValidRuns++
		} else {
			s.InvalidRuns++
		}
		if v.Exact.FalsePositive {
			s.FalsePositives++
		}
		if v.Exact.FalseNegative {
			s.FalseNegatives++
		}
		if v.Work.Status == StatusInvalid {
			s.WorkMismatches++
		}
	}
	return s
}
func p95(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	sort.Float64s(xs)
	i := int(math.Ceil(.95*float64(len(xs)))) - 1
	if i < 0 {
		i = 0
	}
	return xs[i]
}
func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}
