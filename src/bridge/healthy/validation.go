package healthy

import (
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
	"math"
)

func ValidatePath(g core.Graph, run traffic.BenchmarkRun, policy ValidationPolicy) PathValidation {
	v := PathValidation{PathValid: true, EndpointValid: true, EdgeSequenceValid: true, FoundConsistent: true, DistanceConsistent: true}
	fail := func(msg string) { v.PathValid = false; v.Errors = append(v.Errors, msg) }
	if !run.ExecutionResult.PathFound {
		if len(run.ExecutionResult.Path) > 0 {
			v.FoundConsistent = false
			fail("found=false with non-empty path")
		}
		if run.ExecutionResult.PathCost != nil {
			v.FoundConsistent = false
			fail("found=false with distance")
		}
		return v
	}
	if len(run.ExecutionResult.Path) == 0 {
		v.FoundConsistent = false
		fail("found=true with empty path")
		return v
	}
	if run.ExecutionResult.Path[0] != run.QueryProfile.Source || run.ExecutionResult.Path[len(run.ExecutionResult.Path)-1] != run.QueryProfile.Target {
		v.EndpointValid = false
		fail("path endpoints do not match query")
	}
	total := 0.0
	for i, n := range run.ExecutionResult.Path {
		if !g.HasNode(n) {
			v.EdgeSequenceValid = false
			fail(fmt.Sprintf("path node %d does not exist", n))
			continue
		}
		if i == 0 {
			continue
		}
		prev := run.ExecutionResult.Path[i-1]
		found := false
		weight := 0.0
		for _, e := range g.EdgesFrom(prev) {
			if e.To == n {
				found = true
				weight = e.Weight
				break
			}
		}
		if !found {
			v.EdgeSequenceValid = false
			fail(fmt.Sprintf("edge %d->%d does not exist", prev, n))
		} else {
			total += weight
		}
	}
	if v.EdgeSequenceValid {
		v.RecomputedDistance = &total
	}
	if run.ExecutionResult.PathCost == nil {
		v.DistanceConsistent = false
		fail("found=true without distance")
	} else if !closeFloat(*run.ExecutionResult.PathCost, total, policy.DistanceAbsoluteTolerance, policy.DistanceRelativeTolerance) {
		v.DistanceConsistent = false
		fail(fmt.Sprintf("reported distance %.17g differs from recomputed %.17g", *run.ExecutionResult.PathCost, total))
	}
	return v
}
func closeFloat(a, b, absTol, relTol float64) bool {
	d := math.Abs(a - b)
	return d <= absTol || d <= relTol*math.Max(math.Abs(a), math.Abs(b))
}
func ValidateWork(reported core.WorkMetrics, reconstructed *core.WorkMetrics) WorkValidation {
	return ValidateWorkWithLedger(reported, reconstructed, nil)
}

func ValidateWorkWithLedger(reported core.WorkMetrics, reconstructed *core.WorkMetrics, ledger *core.BudgetLedger) WorkValidation {
	w := WorkValidation{Status: StatusPass, StructuralValid: reported.Valid(), ReportedWork: reported}
	w.Mismatches = append(w.Mismatches, reported.ValidationErrors()...)
	if !w.StructuralValid {
		w.Status = StatusInvalid
	}
	if ledger != nil {
		w.LedgerVerifiable = true
		entryTotal := uint64(0)
		componentTotal := uint64(0)
		for _, e := range ledger.Entries {
			entryTotal += e.Used
		}
		for _, n := range ledger.ByComponent {
			componentTotal += n
		}
		w.LedgerValid = ledger.Used == reported.TotalActions && entryTotal == ledger.Used && componentTotal == ledger.Used
		if ledger.Limit != nil && ledger.Used > *ledger.Limit {
			w.LedgerValid = false
		}
		if !w.LedgerValid {
			w.Status = StatusInvalid
			w.Mismatches = append(w.Mismatches, "budget ledger does not match reported work")
		}
	}
	if reconstructed == nil {
		w.TraceVerifiable = false
		return w
	}
	w.TraceVerifiable = true
	w.ReconstructedWork = reconstructed
	w.TraceValid = workEquivalent(reported, *reconstructed)
	if !w.TraceValid {
		w.Status = StatusInvalid
		w.Mismatches = append(w.Mismatches, "reported work differs from reconstructed work")
	}
	return w
}

func workEquivalent(a, b core.WorkMetrics) bool {
	// WorkerCount is execution configuration, not reconstructable Action accounting.
	a.WorkerCount = 0
	b.WorkerCount = 0
	return a == b
}
