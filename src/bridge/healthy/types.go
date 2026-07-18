package healthy

import (
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
)

const ProfileSchemaV1 = "bridge.health.profile.v1"
const ResultSchemaV1 = "bridge.health.result.v1"

type Status string

const (
	StatusPass          Status = "pass"
	StatusWarning       Status = "warning"
	StatusFail          Status = "fail"
	StatusInvalid       Status = "invalid"
	StatusNotVerifiable Status = "not_verifiable"
)

type ValidationPolicy struct {
	DistanceAbsoluteTolerance float64 `json:"distance_absolute_tolerance"`
	DistanceRelativeTolerance float64 `json:"distance_relative_tolerance"`
	RequireExactReference     bool    `json:"require_exact_reference"`
	RequireWorkTrace          bool    `json:"require_work_trace,omitempty"`
	RequireBudgetLedger       bool    `json:"require_budget_ledger,omitempty"`
}
type RegressionPolicy struct {
	PathValidRateMin       *float64 `json:"path_valid_rate_min,omitempty"`
	FalsePositiveCountMax  *int     `json:"false_positive_count_max,omitempty"`
	FalseNegativeCountMax  *int     `json:"false_negative_count_max,omitempty"`
	WorkMismatchCountMax   *int     `json:"work_mismatch_count_max,omitempty"`
	DistanceRatioP95Max    *float64 `json:"distance_ratio_p95_max,omitempty"`
	WorkRatioMeanMax       *float64 `json:"work_ratio_mean_max,omitempty"`
	SolverTimeRatioP95Max  *float64 `json:"solver_time_ratio_p95_max,omitempty"`
	AllocBytesRatioMeanMax *float64 `json:"alloc_bytes_ratio_mean_max,omitempty"`
}
type HealthProfile struct {
	SchemaVersion                 string           `json:"schema_version"`
	CandidateAlgorithm            string           `json:"candidate_algorithm"`
	PerformanceReferenceAlgorithm string           `json:"performance_reference_algorithm,omitempty"`
	ExactReferenceAlgorithm       string           `json:"exact_reference_algorithm,omitempty"`
	Validation                    ValidationPolicy `json:"validation"`
	Policy                        RegressionPolicy `json:"policy"`
}
type PathValidation struct {
	PathValid          bool     `json:"path_valid"`
	EndpointValid      bool     `json:"endpoint_valid"`
	EdgeSequenceValid  bool     `json:"edge_sequence_valid"`
	FoundConsistent    bool     `json:"found_consistent"`
	DistanceConsistent bool     `json:"distance_consistent"`
	RecomputedDistance *float64 `json:"recomputed_distance,omitempty"`
	Errors             []string `json:"errors,omitempty"`
}
type WorkValidation struct {
	Status            Status            `json:"status"`
	StructuralValid   bool              `json:"structural_valid"`
	TraceVerifiable   bool              `json:"trace_verifiable"`
	TraceValid        bool              `json:"trace_valid"`
	LedgerVerifiable  bool              `json:"ledger_verifiable"`
	LedgerValid       bool              `json:"ledger_valid"`
	ReportedWork      core.WorkMetrics  `json:"reported_work"`
	ReconstructedWork *core.WorkMetrics `json:"reconstructed_work,omitempty"`
	Mismatches        []string          `json:"mismatches,omitempty"`
}
type ExactValidation struct {
	Verifiable        bool     `json:"verifiable"`
	ReferenceFound    bool     `json:"reference_found"`
	ReferenceDistance *float64 `json:"reference_distance,omitempty"`
	FalsePositive     bool     `json:"false_positive"`
	FalseNegative     bool     `json:"false_negative"`
	ExactClaimValid   bool     `json:"exact_claim_valid"`
	DistanceRatio     *float64 `json:"distance_ratio,omitempty"`
}
type RunValidation struct {
	RunID     string          `json:"run_id"`
	Algorithm string          `json:"algorithm"`
	Status    Status          `json:"status"`
	Path      PathValidation  `json:"path"`
	Work      WorkValidation  `json:"work"`
	Exact     ExactValidation `json:"optimality_validation"`
}
type PairedComparison struct {
	PairKey            string   `json:"pair_key"`
	CandidateRunID     string   `json:"candidate_run_id"`
	ReferenceRunID     string   `json:"reference_run_id"`
	WorkRatio          *float64 `json:"work_ratio,omitempty"`
	LogicalStepRatio   *float64 `json:"logical_step_ratio,omitempty"`
	ScheduledStepRatio *float64 `json:"scheduled_step_ratio,omitempty"`
	SolverTimeRatio    *float64 `json:"solver_time_ratio,omitempty"`
	EndToEndTimeRatio  *float64 `json:"end_to_end_time_ratio,omitempty"`
	AllocBytesRatio    *float64 `json:"alloc_bytes_ratio,omitempty"`
	DistanceRatio      *float64 `json:"distance_ratio,omitempty"`
}
type Evaluation struct {
	Status  Status   `json:"status"`
	Reasons []string `json:"reasons,omitempty"`
}
type Summary struct {
	Runs           int `json:"runs"`
	ValidRuns      int `json:"valid_runs"`
	InvalidRuns    int `json:"invalid_runs"`
	FalsePositives int `json:"false_positives"`
	FalseNegatives int `json:"false_negatives"`
	WorkMismatches int `json:"work_mismatches"`
}
type HealthCheckResult struct {
	SchemaVersion        string             `json:"schema_version"`
	SourceArtifactID     string             `json:"source_artifact_id,omitempty"`
	Profile              HealthProfile      `json:"profile"`
	RunValidations       []RunValidation    `json:"run_validations"`
	PairedComparisons    []PairedComparison `json:"paired_comparisons,omitempty"`
	RegressionEvaluation Evaluation         `json:"regression_evaluation"`
	Summary              Summary            `json:"summary"`
	SourceSchemaVersion  string             `json:"source_schema_version"`
	SourceArtifactSHA256 string             `json:"source_artifact_sha256,omitempty"`
	GeneratedAt          string             `json:"generated_at"`
	_                    traffic.BenchmarkResult
}
