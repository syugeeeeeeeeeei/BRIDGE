package healthy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
	"time"
)

func Analyze(ctx context.Context, artifact traffic.BenchmarkResult, profile HealthProfile) (HealthCheckResult, error) {
	profile.ApplyDefaults()
	if err := profile.Validate(); err != nil {
		return HealthCheckResult{}, err
	}
	if err := validateArtifact(artifact); err != nil {
		return HealthCheckResult{}, err
	}
	payload, _ := json.Marshal(artifact)
	sum := sha256.Sum256(payload)
	out := HealthCheckResult{SchemaVersion: ResultSchemaV1, SourceArtifactID: artifact.ExecutionID, SourceSchemaVersion: artifact.SchemaVersion, SourceArtifactSHA256: hex.EncodeToString(sum[:]), GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), Profile: profile}
	validByRun := map[string]bool{}
	for _, run := range artifact.Runs {
		if run.RunMetadata.WarmupRun {
			continue
		}
		rv := RunValidation{RunID: run.RunMetadata.RunID, Algorithm: run.RunMetadata.AlgorithmID, Status: StatusPass}
		g, err := traffic.BuildScenarioGraph(run.References.GraphSpecification, run.GraphProfile.GraphSeed)
		if err != nil {
			rv.Status = StatusNotVerifiable
			rv.Path.Errors = []string{"graph reconstruction: " + err.Error()}
			out.RunValidations = append(out.RunValidations, rv)
			continue
		}
		rv.Path = ValidatePath(g, run, profile.Validation)
		var reconstructed *core.WorkMetrics
		if run.References.TraceManifestPath != "" || run.References.TracePath != "" {
			manifestPath := resolveArtifactReference(artifact.OutputDirectory, run.References.TraceManifestPath)
			tracePath := resolveArtifactReference(artifact.OutputDirectory, run.References.TracePath)
			events, manifest, traceErr := loadTrace(manifestPath, tracePath)
			if traceErr == nil {
				rec := ReconstructWork(events, manifest.SampleRate, manifest.Truncated, manifest.Dropped)
				if rec.Verifiable {
					reconstructed = &rec.Work
				}
			}
		}
		rv.Work = ValidateWorkWithLedger(run.Measurement.Work, reconstructed, run.ExecutionResult.BudgetLedger)
		if profile.Validation.RequireWorkTrace && !rv.Work.TraceVerifiable {
			rv.Work.Status = StatusInvalid
			rv.Work.Mismatches = append(rv.Work.Mismatches, "required work trace is not verifiable")
		}
		if profile.Validation.RequireBudgetLedger && run.RunMetadata.AlgorithmID == "bridge" && !rv.Work.LedgerVerifiable {
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
		validByRun[run.RunMetadata.RunID] = rv.Status == StatusPass
		out.RunValidations = append(out.RunValidations, rv)
	}
	out.PairedComparisons = Pair(artifact.Runs, profile, validByRun)
	out.Summary = summarize(out.RunValidations)
	out.RegressionEvaluation = evaluate(out, profile)
	return out, nil
}

func validateArtifact(artifact traffic.BenchmarkResult) error {
	if artifact.SchemaVersion != traffic.BenchmarkResultSchemaV1 {
		return fmt.Errorf("schema_version must be %q", traffic.BenchmarkResultSchemaV1)
	}
	if artifact.TerminologyVersion != traffic.TerminologyVersionV1 {
		return fmt.Errorf("terminology_version must be %q", traffic.TerminologyVersionV1)
	}
	if artifact.RunMetadata.StartedAt == "" {
		return fmt.Errorf("run_metadata.started_at is required")
	}
	if len(artifact.Runs) == 0 {
		return fmt.Errorf("runs must not be empty")
	}
	return nil
}
