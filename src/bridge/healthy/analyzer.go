package healthy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
)

func DefaultProfile(candidate, reference string) HealthProfile {
	one := 1.0
	zero := 0
	return HealthProfile{SchemaVersion: ProfileSchemaV1, CandidateAlgorithm: candidate, PerformanceReferenceAlgorithm: reference, ExactReferenceAlgorithm: "dijkstra", Validation: ValidationPolicy{DistanceAbsoluteTolerance: 1e-9, DistanceRelativeTolerance: 1e-9, RequireExactReference: true}, Policy: RegressionPolicy{PathValidRateMin: &one, FalsePositiveCountMax: &zero, FalseNegativeCountMax: &zero, WorkMismatchCountMax: &zero}}
}
func (p *HealthProfile) ApplyDefaults() {
	if p.SchemaVersion == "" {
		p.SchemaVersion = ProfileSchemaV1
	}
	if p.ExactReferenceAlgorithm == "" {
		p.ExactReferenceAlgorithm = "dijkstra"
	}
	if p.Validation.DistanceAbsoluteTolerance == 0 {
		p.Validation.DistanceAbsoluteTolerance = 1e-9
	}
	if p.Validation.DistanceRelativeTolerance == 0 {
		p.Validation.DistanceRelativeTolerance = 1e-9
	}
}
func (p HealthProfile) Validate() error {
	if p.SchemaVersion != ProfileSchemaV1 {
		return fmt.Errorf("schema_version must be %q", ProfileSchemaV1)
	}
	if p.CandidateAlgorithm == "" {
		return fmt.Errorf("candidate_algorithm is required")
	}
	if p.ExactReferenceAlgorithm != "dijkstra" && p.ExactReferenceAlgorithm != "bidirectional_dijkstra" {
		return fmt.Errorf("exact_reference_algorithm must be dijkstra or bidirectional_dijkstra")
	}
	if p.Validation.DistanceAbsoluteTolerance < 0 || p.Validation.DistanceRelativeTolerance < 0 {
		return fmt.Errorf("distance tolerances must be non-negative")
	}
	return nil
}

func Analyze(ctx context.Context, artifact traffic.BenchmarkResult, profile HealthProfile) (HealthCheckResult, error) {
	profile.ApplyDefaults()
	if err := profile.Validate(); err != nil {
		return HealthCheckResult{}, err
	}
	payload, _ := json.Marshal(artifact)
	sum := sha256.Sum256(payload)
	out := HealthCheckResult{SchemaVersion: ResultSchemaV1, SourceArtifactID: artifact.ArtifactID, SourceSchemaVersion: artifact.SchemaVersion, SourceArtifactSHA256: hex.EncodeToString(sum[:]), GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), Profile: profile}
	validByRun := map[string]bool{}
	for _, run := range artifact.RawRuns {
		if run.Warmup {
			continue
		}
		rv := RunValidation{RunID: run.RunID, Algorithm: run.Algorithm, Status: StatusPass}
		g, err := traffic.BuildScenarioGraph(run.GraphSpec, run.Graph.Seed)
		if err != nil {
			rv.Status = StatusNotVerifiable
			rv.Path.Errors = []string{"graph reconstruction: " + err.Error()}
			out.RunValidations = append(out.RunValidations, rv)
			continue
		}
		rv.Path = ValidatePath(g, run, profile.Validation)
		var reconstructed *core.WorkMetrics
		if run.TraceManifestPath != "" || run.TracePath != "" {
			events, manifest, traceErr := loadTrace(run.TraceManifestPath, run.TracePath)
			if traceErr == nil {
				rec := ReconstructWork(events, manifest.SampleRate, manifest.Truncated, manifest.Dropped)
				if rec.Verifiable {
					reconstructed = &rec.Work
				}
			}
		}
		rv.Work = ValidateWorkWithLedger(run.Work, reconstructed, run.BudgetLedger)
		if profile.Validation.RequireWorkTrace && !rv.Work.TraceVerifiable {
			rv.Work.Status = StatusInvalid
			rv.Work.Mismatches = append(rv.Work.Mismatches, "required work trace is not verifiable")
		}
		if profile.Validation.RequireBudgetLedger && run.Algorithm == "bridge" && !rv.Work.LedgerVerifiable {
			rv.Work.Status = StatusInvalid
			rv.Work.Mismatches = append(rv.Work.Mismatches, "required budget ledger is missing")
		}
		rv.Exact = validateExact(ctx, g, run, profile)
		if !rv.Path.PathValid || rv.Work.Status == StatusInvalid || (profile.Validation.RequireExactReference && !rv.Exact.Verifiable) {
			rv.Status = StatusInvalid
		}
		if rv.Exact.FalsePositive || rv.Exact.FalseNegative || !rv.Exact.ExactClaimValid {
			rv.Status = StatusInvalid
		}
		validByRun[run.RunID] = rv.Status == StatusPass
		out.RunValidations = append(out.RunValidations, rv)
	}
	out.PairedComparisons = Pair(artifact.RawRuns, profile, validByRun)
	out.Summary = summarize(out.RunValidations)
	out.RegressionEvaluation = evaluate(out, profile)
	return out, nil
}

func ValidatePath(g core.Graph, run traffic.RawRunResult, policy ValidationPolicy) PathValidation {
	v := PathValidation{PathValid: true, EndpointValid: true, EdgeSequenceValid: true, FoundConsistent: true, DistanceConsistent: true}
	fail := func(msg string) { v.PathValid = false; v.Errors = append(v.Errors, msg) }
	if !run.Found {
		if len(run.Path) > 0 {
			v.FoundConsistent = false
			fail("found=false with non-empty path")
		}
		if run.Distance != nil {
			v.FoundConsistent = false
			fail("found=false with distance")
		}
		return v
	}
	if len(run.Path) == 0 {
		v.FoundConsistent = false
		fail("found=true with empty path")
		return v
	}
	if run.Path[0] != run.Query.Source || run.Path[len(run.Path)-1] != run.Query.Target {
		v.EndpointValid = false
		fail("path endpoints do not match query")
	}
	total := 0.0
	for i, n := range run.Path {
		if !g.HasNode(n) {
			v.EdgeSequenceValid = false
			fail(fmt.Sprintf("path node %d does not exist", n))
			continue
		}
		if i == 0 {
			continue
		}
		prev := run.Path[i-1]
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
	if run.Distance == nil {
		v.DistanceConsistent = false
		fail("found=true without distance")
	} else if !closeFloat(*run.Distance, total, policy.DistanceAbsoluteTolerance, policy.DistanceRelativeTolerance) {
		v.DistanceConsistent = false
		fail(fmt.Sprintf("reported distance %.17g differs from recomputed %.17g", *run.Distance, total))
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

func validateExact(ctx context.Context, g core.Graph, run traffic.RawRunResult, p HealthProfile) ExactValidation {
	in := graphInput(g)
	res, err := gate.NewRouter().ExecuteOnce(ctx, gate.ExecuteRequest{SchemaVersion: gate.ExecuteRequestSchemaV1, Target: gate.ExecuteTargetInput{ID: p.ExactReferenceAlgorithm}, Graph: in, Route: gate.RouteInput{Source: uint32(run.Query.Source), Target: uint32(run.Query.Target), Mode: core.ModeExact, Workers: 1}, Observation: gate.ObservationInput{Mode: gate.ObservationOff}}, gate.RouteOptions{})
	if err != nil {
		return ExactValidation{Verifiable: false, ExactClaimValid: !run.Exact}
	}
	x := ExactValidation{Verifiable: true, ReferenceFound: res.Found, FalsePositive: run.Found && !res.Found, FalseNegative: !run.Found && res.Found, ExactClaimValid: true}
	if res.Distance != nil {
		d := *res.Distance
		x.ReferenceDistance = &d
	}
	if run.Found && res.Found && run.Distance != nil && res.Distance != nil {
		if *res.Distance == 0 {
			r := 1.0
			if *run.Distance != 0 {
				r = math.Inf(1)
			}
			x.DistanceRatio = &r
		} else {
			r := *run.Distance / *res.Distance
			x.DistanceRatio = &r
		}
		if run.Exact && !closeFloat(*run.Distance, *res.Distance, p.Validation.DistanceAbsoluteTolerance, p.Validation.DistanceRelativeTolerance) {
			x.ExactClaimValid = false
		}
	} else if run.Exact && run.Found != res.Found {
		x.ExactClaimValid = false
	}
	return x
}
func graphInput(g core.Graph) gate.GraphInput {
	in := gate.GraphInput{Type: "inline", Directed: g.Directed(), Nodes: make([]gate.GraphNode, g.NodeCount())}
	for i := 0; i < g.NodeCount(); i++ {
		in.Nodes[i] = gate.GraphNode{ID: uint32(i)}
		for _, e := range g.EdgesFrom(core.NodeID(i)) {
			if !g.Directed() && uint32(i) > uint32(e.To) {
				continue
			}
			in.Edges = append(in.Edges, gate.GraphEdge{From: uint32(i), To: uint32(e.To), Weight: e.Weight})
		}
	}
	return in
}
func pairKey(r traffic.RawRunResult) string {
	return fmt.Sprintf("%s|%s|%s|%d|%d", r.ScenarioID, r.GraphInstanceID, r.QueryID, r.Seed, r.Repetition)
}
func ratio(a, b float64) *float64 {
	if b == 0 {
		return nil
	}
	v := a / b
	return &v
}
func Pair(runs []traffic.RawRunResult, p HealthProfile, valid map[string]bool) []PairedComparison {
	if p.PerformanceReferenceAlgorithm == "" {
		return nil
	}
	cand := map[string]traffic.RawRunResult{}
	ref := map[string]traffic.RawRunResult{}
	for _, r := range runs {
		if r.Warmup {
			continue
		}
		k := pairKey(r)
		if r.Algorithm == p.CandidateAlgorithm {
			cand[k] = r
		}
		if r.Algorithm == p.PerformanceReferenceAlgorithm {
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
		if !ok || !valid[c.RunID] || !valid[r.RunID] {
			continue
		}
		pc := PairedComparison{PairKey: k, CandidateRunID: c.RunID, ReferenceRunID: r.RunID, WorkRatio: ratio(float64(c.Work.TotalActions), float64(r.Work.TotalActions)), LogicalStepRatio: ratio(float64(c.Work.LogicalSteps), float64(r.Work.LogicalSteps)), ScheduledStepRatio: ratio(float64(c.Work.ScheduledSteps), float64(r.Work.ScheduledSteps)), SolverTimeRatio: ratio(c.SolverTimeMS, r.SolverTimeMS), EndToEndTimeRatio: ratio(c.EndToEndTimeMS, r.EndToEndTimeMS), AllocBytesRatio: ratio(float64(c.SystemMetrics.AllocBytes), float64(r.SystemMetrics.AllocBytes))}
		if c.Distance != nil && r.Distance != nil {
			pc.DistanceRatio = ratio(*c.Distance, *r.Distance)
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
