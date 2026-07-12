# BRIDGE / MRPC-DG5 clean production guard report

Date: 2026-07-10

## Purpose

This update separates production BRIDGE routing from evaluation-oracle behavior.

MRPC-DG5 may use exact local solvers such as Dijkstra or bidirectional Dijkstra as internal subsolvers. That remains valid when they are used for local segments, bounded handoff, reachability-safe completion, or explicit solver delegation.

The behavior removed from default production execution is different: computing the whole source-target exact path after a completed MRPC candidate and using that exact result as an oracle to replace the candidate.

## What changed

### 1. Oracle quality guard disabled by default

`mrpc_dg5_switchback()` now defaults to:

```text
quality_guard_max_ratio = None
allow_oracle_quality_guard = False
```

The whole-query exact quality guard can only run when both are explicitly set:

```text
quality_guard_max_ratio = <ratio>
allow_oracle_quality_guard = True
```

This keeps normal BRIDGE usage clean while preserving a diagnostic/evaluation mode for ablation.

### 2. Route API made production-clean by default

`route(..., mode="mrpc_dg5")` now passes the clean defaults unless the caller explicitly opts into oracle diagnostics.

The older `route(..., mode="fast"/"balanced", constraints={"max_distance_ratio": ...})` exact replacement behavior is also gated behind:

```text
allow_oracle_quality_guard = True
```

### 3. Telemetry labels added

Telemetry now records whether oracle-style exact verification was enabled or used:

```text
oracle_quality_guard_enabled
oracle_quality_guard_used
delegated_exact_solver
mrpc_fast_path_used
```

These labels are intended to prevent future evaluation reports from mixing MRPC fast-path quality with exact-replacement quality.

## What was not removed

The following remain allowed because they are legitimate BRIDGE stack behavior rather than leakage:

- local exact search inside a bounded segment
- bidirectional handoff from detour/re-entry seeds
- component/reachability precheck
- topology-gate delegation to exact solver for clearly unsuitable topology
- safe exact completion when MRPC fails to produce a usable route

These must still be accounted for in telemetry and performance metrics.

## Validation

`pytest -q` result:

```text
10 passed, 2 warnings
```

Warnings are Python 3.13 multiprocessing fork warnings and are not DG5 correctness failures.

## Evaluation implication

Future reports should separate at least these categories:

- MRPC fast path result
- local exact / bounded handoff result
- topology-gate delegated exact result
- safe exact completion result
- explicit oracle diagnostic replacement result

Only the first three should be considered production BRIDGE stack behavior by default. The final category must be treated as diagnostic/evaluation-only.
