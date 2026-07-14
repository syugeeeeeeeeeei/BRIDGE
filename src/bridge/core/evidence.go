package core

import (
	"fmt"
	"math"
)

type ProofClass string

const (
	ProofEmpirical            ProofClass = "empirical"
	ProofAdmissibleLowerBound ProofClass = "admissible_lower_bound"
	ProofUnreachable          ProofClass = "unreachable"
	ProofExact                ProofClass = "exact"
)

type Evidence struct {
	ID            string     `json:"id"`
	Solver        string     `json:"solver"`
	HypothesisID  string     `json:"hypothesis_id,omitempty"`
	Scope         Region     `json:"scope"`
	GeneratedWork uint64     `json:"generated_work"`
	Proof         ProofClass `json:"proof_class"`
	Value         float64    `json:"value,omitempty"`
	InvalidatedBy []string   `json:"invalidated_by,omitempty"`
}

func (e Evidence) Validate() error {
	if e.ID == "" || e.Solver == "" {
		return fmt.Errorf("evidence requires id and solver")
	}
	switch e.Proof {
	case ProofEmpirical:
		if math.IsNaN(e.Value) {
			return fmt.Errorf("empirical evidence value is NaN")
		}
	case ProofAdmissibleLowerBound, ProofExact:
		if len(e.Scope.Nodes) == 0 {
			return fmt.Errorf("proof evidence requires non-empty scope")
		}
		if e.GeneratedWork == 0 {
			return fmt.Errorf("proof evidence requires generated work")
		}
		if math.IsNaN(e.Value) || math.IsInf(e.Value, 0) || e.Value < 0 {
			return fmt.Errorf("proof value must be finite and non-negative")
		}
	case ProofUnreachable:
		if len(e.Scope.Nodes) == 0 {
			return fmt.Errorf("unreachable proof requires non-empty scope")
		}
		if e.GeneratedWork == 0 {
			return fmt.Errorf("unreachable proof requires generated work")
		}
		if e.Value != 0 {
			return fmt.Errorf("unreachable proof must not carry a distance value")
		}
	default:
		return fmt.Errorf("invalid proof class %q", e.Proof)
	}
	seen := map[string]struct{}{}
	for _, condition := range e.InvalidatedBy {
		if condition == "" {
			return fmt.Errorf("empty invalidation condition")
		}
		if _, ok := seen[condition]; ok {
			return fmt.Errorf("duplicate invalidation condition %q", condition)
		}
		seen[condition] = struct{}{}
	}
	return nil
}
func (e Evidence) IsProof() bool { return e.Proof != ProofEmpirical }
func (e Evidence) Covers(region Region) bool {
	for _, n := range region.Nodes {
		if !e.Scope.Contains(n) {
			return false
		}
	}
	return true
}
func (e Evidence) IsInvalidated(changes []string) bool {
	changed := map[string]struct{}{}
	for _, c := range changes {
		changed[c] = struct{}{}
	}
	for _, c := range e.InvalidatedBy {
		if _, ok := changed[c]; ok {
			return true
		}
	}
	return false
}
