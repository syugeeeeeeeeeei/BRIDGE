# Phase 5 Completion Checklist

## Synthetic graph families

- [x] Community graph generator is deterministic by seed.
- [x] Maze graph generator is deterministic by seed.
- [x] Adversarial graph generator is deterministic by seed.
- [x] New generators use TRAFFIC only and do not alter production routing packages.
- [x] Research scenarios cover all new graph families with multiple seeds and repetitions.

## Dataset contract

- [x] `bridge.dataset.v1` document contract is defined.
- [x] Dataset loading is implemented only in TRAFFIC.
- [x] Directedness, weighted edges, optional positions, and default query are supported.
- [x] Node and edge validation rejects invalid datasets.
- [x] Source and license are mandatory.
- [x] SHA-256 is calculated from the exact dataset bytes.
- [x] Ordered preprocessing records are preserved.
- [x] Dataset provenance is copied into every raw run.
- [x] A licensed synthetic fixture and dataset Scenario are included.

## Statistical reproducibility

- [x] Statistical analysis consumes raw runs and excludes warm-up runs.
- [x] Descriptive statistics and bootstrap 95% confidence intervals are generated.
- [x] Mann–Whitney U approximate significance is generated.
- [x] Cliff's delta effect size is generated.
- [x] Group and metric are command-line selectable.
- [x] Report output is deterministic under a fixed bootstrap seed.

## Contracts and governance

- [x] Benchmark Scenario schema includes all new generators and dataset fields.
- [x] Benchmark Result schema includes dataset provenance.
- [x] Dataset specification is documented.
- [x] New BRIDGE-specific terms are defined in the glossary.
- [x] TRAFFIC component rules state the production boundary.
- [x] Phase 5 acceptance tests cover generators, loading, provenance, and Scenario execution.

## Cross-phase verification

- [x] `go test ./...`
- [x] `go test -race ./...`
- [x] `go vet ./...`
- [x] `python tests/compatibility/verify.py`
- [x] Phase 1–5 final gap audit completed.
