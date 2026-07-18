package healthy

import (
	"fmt"
	"path/filepath"
)

func evaluate(r HealthCheckResult, p HealthProfile) Evaluation {
	e := Evaluation{Status: StatusPass}
	bad := func(msg string) { e.Status = StatusFail; e.Reasons = append(e.Reasons, msg) }
	if r.Summary.InvalidRuns > 0 {
		e.Status = StatusInvalid
		e.Reasons = append(e.Reasons, fmt.Sprintf("%d invalid runs", r.Summary.InvalidRuns))
	}
	if x := p.Policy.PathValidRateMin; x != nil && r.Summary.Runs > 0 && float64(r.Summary.ValidRuns)/float64(r.Summary.Runs) < *x {
		bad("path valid rate below policy")
	}
	if x := p.Policy.FalsePositiveCountMax; x != nil && r.Summary.FalsePositives > *x {
		bad("false positive count exceeds policy")
	}
	if x := p.Policy.FalseNegativeCountMax; x != nil && r.Summary.FalseNegatives > *x {
		bad("false negative count exceeds policy")
	}
	if x := p.Policy.WorkMismatchCountMax; x != nil && r.Summary.WorkMismatches > *x {
		bad("work mismatch count exceeds policy")
	}
	wr, st, al, dr := []float64{}, []float64{}, []float64{}, []float64{}
	for _, c := range r.PairedComparisons {
		if c.WorkRatio != nil {
			wr = append(wr, *c.WorkRatio)
		}
		if c.SolverTimeRatio != nil {
			st = append(st, *c.SolverTimeRatio)
		}
		if c.AllocBytesRatio != nil {
			al = append(al, *c.AllocBytesRatio)
		}
		if c.DistanceRatio != nil {
			dr = append(dr, *c.DistanceRatio)
		}
	}
	if x := p.Policy.WorkRatioMeanMax; x != nil && len(wr) > 0 && mean(wr) > *x {
		bad("mean work ratio exceeds policy")
	}
	if x := p.Policy.SolverTimeRatioP95Max; x != nil && len(st) > 0 && p95(st) > *x {
		bad("p95 solver time ratio exceeds policy")
	}
	if x := p.Policy.AllocBytesRatioMeanMax; x != nil && len(al) > 0 && mean(al) > *x {
		bad("mean allocation ratio exceeds policy")
	}
	if x := p.Policy.DistanceRatioP95Max; x != nil && len(dr) > 0 && p95(dr) > *x {
		bad("p95 distance ratio exceeds policy")
	}
	return e
}

func resolveArtifactReference(outputDirectory, reference string) string {
	if reference == "" || filepath.IsAbs(reference) {
		return reference
	}
	return filepath.Join(outputDirectory, filepath.FromSlash(reference))
}
