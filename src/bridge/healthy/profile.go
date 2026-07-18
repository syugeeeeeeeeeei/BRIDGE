package healthy

import (
	"fmt"
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
