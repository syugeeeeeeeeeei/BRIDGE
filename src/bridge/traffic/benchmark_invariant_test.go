package traffic

import (
	"testing"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

func TestValidateBenchmarkRunClaimsRejectsFalseOptimality(t *testing.T) {
	run := BenchmarkRun{}
	run.ExecutionResult.PathFound = true
	run.ExecutionResult.SearchCompleted = true
	run.ExecutionResult.ReachabilityProven = true
	run.ExecutionResult.OptimalityProven = true
	run.Measurement.TimingValid = true
	run.Measurement.EndToEndTimeNS = 10
	run.Measurement.SolverTimeNS = 10
	if err := validateBenchmarkRunClaims(run); err != nil {
		t.Fatalf("valid run rejected: %v", err)
	}

	run.ExecutionResult.PathFound = false
	if err := validateBenchmarkRunClaims(run); err == nil {
		t.Fatal("false optimality claim was accepted")
	}
}

func TestValidateBenchmarkRunClaimsRejectsBudgetProof(t *testing.T) {
	run := BenchmarkRun{}
	run.ExecutionResult.ErrorCode = core.ErrBudgetExhausted
	run.ExecutionResult.SearchCompleted = true
	run.ExecutionResult.ReachabilityProven = true
	if err := validateBenchmarkRunClaims(run); err == nil {
		t.Fatal("budget exhaustion proof claim was accepted")
	}
}

func TestValidateBenchmarkRunClaimsRejectsInvalidTiming(t *testing.T) {
	run := BenchmarkRun{}
	run.Measurement.TimingValid = true
	run.Measurement.EndToEndTimeNS = 100
	run.Measurement.SolverTimeNS = 0
	if err := validateBenchmarkRunClaims(run); err == nil {
		t.Fatal("zero solver duration marked valid")
	}
}
