# BRIDGE v0.9.1 Runtime and Trace Evaluation

## Scope

This revision adds component-level runtime measurement and exact portfolio-level investigation accounting without changing the discrete Work definition.

## Implemented

- ANCHOR, BOLTS, Supervisor, Arbiter, and orchestration-overhead timing.
- Stable investigated node and directed-edge identifiers in solver telemetry.
- Portfolio unique node/edge counts.
- Cross-component duplicate node/edge counts.
- Edge-ratio denominator standardized to directed adjacency slots.
- Deferred portfolio union to avoid map-building overhead when only one component runs.
- 5,000-node topology benchmark exports the new metrics.

## Important finding

In the current balanced execution path, ANCHOR completes all connected benchmark cases before Supervisor intervention. BOLTS runtime and cross-component duplicate counts are therefore zero. The previously assumed BOLTS full re-search is not active in this benchmark path.

The remaining architecture limitation is that Supervisor inspects ANCHOR only after `Solve` returns. True mid-search emergency handling requires an incremental ANCHOR execution contract (`RunSlice`, checkpoint, resume/yield). This was not simulated by restarting ANCHOR because that would increase duplicate work and produce misleading results.

## Runtime interpretation

`time_ms` is wall-clock end-to-end solver time. BRIDGE additionally reports:

- `anchor_time_ms`
- `bolts_time_ms`
- `supervisor_time_ms`
- `arbiter_time_ms`
- `orchestration_overhead_ms`

Single executions remain sensitive to sub-millisecond noise. Comparative conclusions should prioritize median and p95 over isolated runs.
