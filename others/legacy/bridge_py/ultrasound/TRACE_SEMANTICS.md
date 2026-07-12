# ULTRASOUND Trace Semantics

**Schema version:** `1.1.0`  
**Status:** Normative  
**Scope:** BEARING events serialized or aggregated by ULTRASOUND

## 1. Purpose

This document defines what every ULTRASOUND value means. A trace field is not valid merely because it is serializable. It is valid only when its producer, unit, timing, nullability, invariants, and prohibited interpretations are defined and the emitted value satisfies those rules.

The executable counterpart is `bridge_py/ultrasound/semantics.py`. Documentation and executable validation are one contract. Changes to either require a schema-version decision and tests.

## 2. Governance rules

1. No undocumented event kind or field may be emitted into a persisted artifact.
2. Existing field meaning must never be changed without a schema-version change.
3. A producer must emit a value at the defined observation point, not reconstruct it later from unrelated totals.
4. `null` means “not available under this contract,” never zero, infinity, false, or an empty identifier.
5. Units must not be mixed. Current work units are node expansions unless explicitly documented otherwise.
6. Diagnostic time and algorithmic progress are separate. `relative_ns` cannot substitute for `logical_step` or `work_used`.
7. Heuristic estimates must not be stored in certified-bound fields.
8. ULTRASOUND may observe and validate, but must not alter solver choice, budget, tie-breaking, graph state, random-number consumption, or RouteResult.
9. Persisted traces must pass `validate_trace(..., strict=True)` before being accepted as a valid artifact.
10. TRAFFIC acceptance criteria may use a field only when this document defines its meaning and the scenario records the schema version.

## 3. Record envelope

| Field | Meaning | Unit | Required invariants | Prohibited interpretation |
|---|---|---|---|---|
| `schema_version` | Contract version used to interpret the record | semantic version | Equal to writer-supported version | Application/package version |
| `sequence` | One-based append order within one artifact | ordinal | Starts at 1; no gaps; increments by 1 | Work count or logical step |
| `kind` | Event vocabulary member selecting the payload contract | enum string | Must be documented | Arbitrary logging message |
| `relative_ns` | Monotonic elapsed time from observer creation to append | nanoseconds | Non-negative; non-decreasing | CPU time, work, or cross-run comparable timestamp |

`relative_ns` is operational telemetry. It is affected by instrumentation, scheduling, interpreter behavior, and hardware. Performance comparisons must use controlled benchmark methodology rather than raw per-event deltas alone.

## 4. Shared search identity

| Field | Meaning | Rules |
|---|---|---|
| `task_id` | Stable identifier assigned by TRUSS to one solver task | Non-empty and stable throughout the task |
| `logical_step` | One-based node-expansion ordinal local to a task/lane | `0` is reserved for task-level events outside an expansion; node expansion uses `>=1` |
| `lane` | Logical search stream, such as `anchor` or `bolt` | Does not prove physical parallel execution |
| `phase` | Controlled algorithmic phase name | Must identify where the event was produced |

A logical step is not globally unique. Consumers must key task-local events by at least `(task_id, lane, logical_step)`.

## 5. Event contracts

### 5.1 `phase_started` / `phase_finished`

They delimit a named component phase. Every finish must have a preceding matching start in the same artifact. `attributes` are descriptive metadata. An attribute key may not be used as a regression metric until its semantics are separately documented.

### 5.2 `step_started`

Marks the beginning of one logical expansion step for a lane. It is an ordering marker, not work by itself. The authoritative work increment is the corresponding `node_expanded` event.

### 5.3 `node_expanded`

| Field | Meaning |
|---|---|
| `node` | Node selected from the active frontier for expansion |
| `distance` | Best known source-to-node graph cost at expansion time |
| `frontier_size` | Queue entries immediately after removing the expanded entry and before relaxing outgoing edges |
| `work_used` | Cumulative node-expansion work charged to this task, including this expansion |

`frontier_size` counts entries, not necessarily unique nodes. A priority queue may contain stale duplicates. `work_used` does not count edge relaxations and is not the portfolio total unless the task is the only task.

### 5.4 `edge_relaxed`

One directed relaxation attempt generated from the node expanded in the same `(task_id, lane, logical_step)`.

- `old_distance`: tentative target cost before the attempt; `null` means no finite tentative cost existed.
- `new_distance`: candidate cost computed from source tentative cost plus edge weight.
- `improved`: true exactly when `new_distance < old_distance`, treating `null` as infinity.

An improved relaxation does not imply that the target will be expanded or appear on the final path.

### 5.5 `neighbor_scored`

Records ranking of a neighbor by ANCHOR's greedy geometric strategy. `heuristic_to_target`, `progress`, and `score` are strategy-local ranking values. They are deliberately separated from `edge_relaxed` because they are not tentative graph distances. `score` must never be used as route cost, lower bound, upper bound, or quality certificate.

### 5.6 `candidate_found`

Represents one complete source-to-target candidate emitted by a solver task.

- When `found=true`, `distance` must be finite/non-negative and `path_length >= 1`.
- When `found=false`, `distance=null` and `path_length=0`.
- `distance` is the sum of graph edge costs for that candidate.
- `strategy` identifies the ANCHOR hypothesis when applicable; it is not a quality guarantee.

### 5.7 `bound_updated`

- `upper_bound`: cost of the best complete valid route currently known.
- `lower_bound`: certified lower bound on the optimal route cost.
- `certified_ratio`: `upper_bound / lower_bound` when the lower bound is positive and certified.

Heuristic estimates, frontier priorities, and unproven geometric distances must not be written as `lower_bound`. If `certified_ratio=1`, the supporting bounds must prove optimality rather than merely coincide numerically by accident.

### 5.8 `budget_updated`

- `max_work`: node-expansion cap assigned to the task slice; `null` means no explicit cap.
- `work_used`: portfolio expansion work already charged when the event is emitted.
- `portfolio_remaining`: remaining portfolio expansion budget after charged work; `null` means uncapped.

This event describes an allocation/accounting state. It does not itself consume budget.

## 6. Cross-event invariants

A semantically valid trace must satisfy all applicable rules:

- `sequence` is contiguous and `relative_ns` is non-decreasing.
- Node-expansion logical steps are contiguous per `(task_id, lane)`.
- `work_used` never decreases within a task.
- Every edge relaxation and neighbor-scoring event follows a node expansion in the same task/lane/step.
- `improved` agrees with `old_distance` and `new_distance`.
- Candidate presence agrees with distance and path length.
- Certified lower bounds do not exceed known upper bounds.
- A certified ratio agrees numerically with its bounds and is at least 1.
- Every finished phase has a matching start.
- Values representing graph costs are finite and non-negative for the supported graph contract.

## 7. Change-control procedure

Any telemetry change must include all of the following in one change set:

1. Update the BEARING event schema or producer contract.
2. Update `semantics.py` registry and validator.
3. Update this document.
4. Add positive and negative semantic tests.
5. Decide schema versioning:
   - patch: clarification or stricter validation with unchanged serialized meaning;
   - minor: backward-compatible new event/optional field;
   - major: renamed/removed field, unit change, observation-point change, or changed meaning.
6. Run ULTRASOUND ON/OFF non-interference tests.
7. Run TRAFFIC trace-to-result consistency tests.

## 8. Consumer rule

Consumers must reject unsupported schema versions and must not silently guess a field's meaning. Descriptive counts from `InMemoryObserver.metrics()` are not certification; semantic validity is reported separately and persisted artifacts are validated before writing by default.
