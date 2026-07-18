package traffic

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/ultrasound"
	"runtime"
	"strings"
	"time"
)

func appendOptional(values []float64, value *float64) []float64 {
	if value == nil {
		return values
	}
	return append(values, *value)
}

func classifyFailure(code core.ErrorCode, _ core.SystemMetrics, fallbackUsed bool) string {
	switch code {
	case core.ErrDeadlineExceeded, core.ErrCancelled:
		return "timeout"
	case core.ErrBudgetExhausted:
		return "budget_exhausted"
	case core.ErrNoPath:
		if fallbackUsed {
			return "fallback_failure"
		}
		return "disconnected"
	case core.ErrInvalidRequest:
		return "invalid_request"
	default:
		return "no_path"
	}
}

func startHeapSampler(enabled bool, initial uint64) func() uint64 {
	if !enabled {
		return func() uint64 { return 0 }
	}
	done := make(chan struct{})
	result := make(chan uint64, 1)
	go func() {
		peak := initial
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				if m.HeapAlloc > peak {
					peak = m.HeapAlloc
				}
			case <-done:
				result <- peak
				return
			}
		}
	}()
	return func() uint64 { close(done); return <-result }
}

func averageOutDegree(g *core.AdjacencyGraph) float64 {
	if g == nil || g.NodeCount() == 0 {
		return 0
	}
	return float64(g.EdgeCount()) / float64(g.NodeCount())
}

func edgeDensity(g *core.AdjacencyGraph) float64 {
	if g == nil || g.NodeCount() < 2 {
		return 0
	}
	nodes := float64(g.NodeCount())
	denominator := nodes * (nodes - 1)
	if !g.Directed() {
		denominator /= 2
	}
	if denominator == 0 {
		return 0
	}
	return float64(g.EdgeCount()) / denominator
}

func queryStableHash(queryID, strategy string, source, target uint32, seed int64) string {
	payload := struct {
		QueryID   string `json:"query_id"`
		Strategy  string `json:"query_selection_method"`
		Source    uint32 `json:"source"`
		Target    uint32 `json:"target"`
		QuerySeed int64  `json:"query_seed"`
	}{queryID, strategy, source, target, seed}
	b, _ := json.Marshal(payload)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func terminationReason(found bool, code core.ErrorCode) string {
	if found {
		return "path_returned"
	}
	switch code {
	case core.ErrDeadlineExceeded, core.ErrCancelled:
		return "timeout"
	case core.ErrBudgetExhausted:
		return "budget_exhausted"
	case core.ErrNoPath:
		return "unreachable"
	case core.ErrInvalidRequest:
		return "invalid_request"
	case "":
		return "completed_without_path"
	default:
		return "error"
	}
}

func rawRunStableDigest(raw BenchmarkRun) string {
	payload := struct {
		ScenarioID string           `json:"scenario_id"`
		Algorithm  string           `json:"algorithm"`
		QueryID    string           `json:"query_id"`
		Seed       int64            `json:"seed"`
		Found      bool             `json:"path_found"`
		Exact      bool             `json:"optimality_proven"`
		Distance   *float64         `json:"path_cost,omitempty"`
		Work       core.WorkMetrics `json:"work"`
		ErrorCode  core.ErrorCode   `json:"error_code,omitempty"`
	}{
		raw.RunMetadata.ScenarioID,
		raw.RunMetadata.AlgorithmID,
		raw.RunMetadata.QueryID,
		raw.RunMetadata.ExecutionSeed,
		raw.ExecutionResult.PathFound,
		raw.ExecutionResult.OptimalityProven,
		raw.ExecutionResult.PathCost,
		raw.Measurement.Work,
		raw.ExecutionResult.ErrorCode,
	}
	b, _ := json.Marshal(payload)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func buildDebugSummary(work core.WorkMetrics, ledger *core.BudgetLedger, metrics ultrasound.CollectorMetrics) *DebugSummary {
	out := &DebugSummary{
		ActionCounts: map[string]uint64{
			"select": work.SelectActions, "expand": work.ExpandActions, "evaluate": work.EvaluateActions,
			"relax": work.RelaxActions, "enqueue": work.EnqueueActions, "reject": work.RejectActions,
			"backtrack": work.BacktrackActions, "connect": work.ConnectActions, "candidate": work.CandidateActions,
			"repair": work.RepairActions, "bound": work.BoundActions, "terminate": work.TerminateActions,
			"hypothesis": work.HypothesisActions, "evidence": work.EvidenceActions, "handoff": work.HandoffActions,
			"schedule": work.ScheduleActions,
		},
		WorkByComponent: map[string]uint64{}, BudgetGrantedByPurpose: map[string]uint64{}, BudgetUsedByPurpose: map[string]uint64{},
		CandidateUpdateCount: maxUint64(metrics.DebugSummary.CandidateUpdateCount, work.CandidateActions), FallbackCount: metrics.DebugSummary.FallbackCount,
		CertificationCount: metrics.DebugSummary.CertificationCount, StateReuseAppliedCount: metrics.DebugSummary.StateReuseAppliedCount,
		MaxFrontierSize: metrics.DebugSummary.MaxFrontierSize, ComponentEventCounts: metrics.DebugSummary.ComponentEventCounts,
		ObservationOverheadNS: metrics.ObservationNS, TraceSinkWriteNS: metrics.SinkWriteNS,
		DroppedEvents: metrics.DroppedEvents, Truncated: metrics.Truncated,
	}
	if ledger != nil {
		for k, v := range ledger.ByComponent {
			out.WorkByComponent[string(k)] = v
		}
		for _, e := range ledger.Entries {
			if e.Granted != nil {
				out.BudgetGrantedByPurpose[e.Purpose] += *e.Granted
			}
			out.BudgetUsedByPurpose[e.Purpose] += e.Used
			id := strings.ToLower(e.TaskID)
			if strings.Contains(id, "certif") {
				out.CertificationCount++
			}
			if strings.Contains(id, "fallback") || strings.Contains(id, "emergency") {
				out.FallbackCount++
			}
		}
	}
	return out
}

func enrichHandoffBaselines(runs []BenchmarkRun) {
	type key struct {
		scenario, graph, query string
		seed                   int64
		rep                    int
	}
	baseline := map[key]BenchmarkRun{}
	for _, run := range runs {
		if run.RunMetadata.WarmupRun || run.RunMetadata.AlgorithmID != "weighted_astar" {
			continue
		}
		baseline[key{run.RunMetadata.ScenarioID, run.RunMetadata.GraphInstanceID, run.RunMetadata.QueryID, run.RunMetadata.ExecutionSeed, run.RunMetadata.RepetitionIndex}] = run
	}
	for i := range runs {
		h := runs[i].ExecutionResult.HandoffMetrics
		if h == nil {
			continue
		}
		b, ok := baseline[key{runs[i].RunMetadata.ScenarioID, runs[i].RunMetadata.GraphInstanceID, runs[i].RunMetadata.QueryID, runs[i].RunMetadata.ExecutionSeed, runs[i].RunMetadata.RepetitionIndex}]
		if !ok {
			continue
		}
		for j := range h.Records {
			bw := b.Measurement.Work.TotalActions
			bt := b.Measurement.SolverTimeNS
			h.Records[j].BoltsStandaloneWork = &bw
			dw := int64(h.Records[j].AnchorWorkAtHandoff+h.Records[j].BoltsWork) - int64(bw)
			h.Records[j].AdditionalWorkVsBoltsStandalone = &dw
			h.Records[j].BoltsStandaloneTimeNS = &bt
			dt := h.Records[j].BoltsTimeNS - bt
			h.Records[j].AdditionalTimeNSVsBoltsStandalone = &dt
		}
	}
}
