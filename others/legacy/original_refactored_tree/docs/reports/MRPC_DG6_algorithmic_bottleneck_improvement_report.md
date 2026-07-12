# MRPC-DG6 Algorithmic Bottleneck Improvement Report

## Scope

This iteration only targets bottlenecks that can be reduced by algorithmic changes inside DG6.  It does not rely on Rust, multiprocessing, native arrays, or source-target exact-oracle replacement.

## Implemented changes

### 1. Grid hard-cut precheck

DG6 now detects integer-grid maps with a full vertical blocking column between source and target.  When such a hard cut is found, DG6 returns unreachable immediately instead of spending the emergency beam budget.

Effect target: disconnected cases.

### 2. Direct grid gap detour

For grid-like wall and double-wall cases, DG6 now detects sparse barrier columns and constructs a rectilinear path through detected gap nodes.  This avoids generic emergency beam routing for wall-like cases.

Effect target: wall, double_wall, spiral-like sparse barrier cases.

### 3. Portal longest-edge heap selection

Portal candidate extraction now keeps only the longest edge candidates in a bounded heap instead of collecting and sorting all edges.  This preserves the long-edge portal skeleton while reducing overhead.

Effect target: clustered / portal strategy.

### 4. Reachability guard before emergency routing

If all budgeted strategies fail, DG6 runs a reachability check before invoking emergency approximate routing.  This prevents unreachable cases from wasting route-construction work.

Effect target: unreachable / disconnected cases not caught by the grid hard-cut detector.

## Test result

```text
12 passed, 2 warnings
```

Warnings are Python multiprocessing fork warnings and are not DG6 route-quality failures.

## Global evaluation

| solver | valid | within 10% | exact | mean ratio | worst ratio | mean work | mean time ms | mean step |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| dijkstra | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 3559.2 | 8.044 | 756.0 |
| bidir | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 2810.6 | 15.430 | 688.1 |
| astar | 0.936 | 0.987 | 0.923 | 1.005 | 1.133 | 2663.7 | 6.178 | 596.1 |
| dg5_clean | 1.000 | 0.859 | 0.782 | 1.087 | 2.118 | 5470.8 | 10.880 | 391.8 |
| dg6 | 1.000 | 1.000 | 0.897 | 1.005 | 1.095 | 273.2 | 3.217 | 20.8 |


## DG6 before / after

| metric | runtime optimized baseline | algorithm optimized |
|---|---:|---:|
| within 10% rate | 1.000 | 1.000 |
| mean distance ratio | 1.005 | 1.005 |
| worst distance ratio | 1.095 | 1.095 |
| mean work | 564.0 | 273.2 |
| mean time ms | 5.600 | 3.217 |
| mean step | 39.9 | 20.8 |

Work reduction versus the runtime-optimized baseline: 51.6%.
Time reduction versus the runtime-optimized baseline: 42.6%.

## Topology summary for DG6

| topology | within 10% | mean ratio | worst ratio | mean work | mean time ms | emergency rate |
|---|---:|---:|---:|---:|---:|---:|
| clustered | 1.000 | 1.000 | 1.000 | 195.0 | 6.205 | 0.000 |
| culdesac | 1.000 | 1.000 | 1.000 | 430.0 | 3.982 | 0.000 |
| disconnected | 1.000 | 1.000 | 1.000 | 0.0 | 1.453 | 0.000 |
| double_wall | 1.000 | 1.000 | 1.000 | 94.0 | 2.217 | 0.000 |
| open | 1.000 | 1.000 | 1.000 | 450.0 | 3.825 | 0.000 |
| random_geometric | 1.000 | 1.061 | 1.095 | 292.5 | 2.902 | 0.000 |
| random_obstacles | 1.000 | 1.000 | 1.000 | 394.7 | 3.588 | 0.000 |
| scale_free_no_pos | 1.000 | 1.000 | 1.000 | 216.8 | 1.773 | 0.000 |
| scale_free_pos | 1.000 | 1.000 | 1.000 | 216.8 | 2.689 | 0.000 |
| spiral | 1.000 | 1.000 | 1.000 | 58.0 | 2.049 | 0.000 |
| u_shape | 1.000 | 1.000 | 1.000 | 429.5 | 3.912 | 0.000 |
| wall | 1.000 | 1.000 | 1.000 | 58.0 | 2.440 | 0.000 |
| weighted_noise | 1.000 | 1.008 | 1.045 | 716.8 | 4.779 | 0.000 |


## Interpretation

The largest gain came from replacing generic emergency routing with structure-specific logic on grid barriers.  Disconnected cases now terminate through a hard-cut check, and double-wall cases use direct gap detours instead of emergency beam search.  Portal extraction was also made lighter without changing the oracle-free property.

The remaining visible bottleneck is weighted_noise.  It still uses a relatively large weighted connector budget to preserve the 10% SLA.  Reducing this further requires a stronger low-cost basin strategy rather than a generic budget cut, because naive reductions degrade quality.

## Oracle status

DG6 still does not use source-target exact shortest path replacement.  Dijkstra-like methods remain limited to local connectors, bounded connectors, or evaluation reference outside the solver.
