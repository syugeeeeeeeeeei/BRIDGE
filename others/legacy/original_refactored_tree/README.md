# BRIDGE Python Reference Implementation

BRIDGEのPython参照実装です。正式な実行経路は次の一方向依存に固定されています。

````text
GATE → TRUSS → ANCHOR / BOLTS
                  │
                  └→ BEARING → ULTRASOUND（optional）
````

## Repository layout

````text
bridge_python/
├─ src/bridge_py/          # インストール対象
│  ├─ core/                # 共通データ型・Graph・Result
│  ├─ gate/                # 公開APIとrequest正規化
│  ├─ truss/               # 計画・予算・orchestration
│  ├─ anchor/              # 主探索portとadapter
│  ├─ bolts/               # 補助solver portとadapter
│  ├─ bearing/             # 非干渉observer契約
│  ├─ ultrasound/          # optional観測実装
│  ├─ traffic/             # test・benchmark実装
│  ├─ solvers/             # 研究用・旧solver群
│  └─ cable/               # 非推奨互換facade
├─ tests/
│  ├─ architecture/
│  ├─ integration/
│  └─ unit/
├─ tools/
│  ├─ benchmarks/
│  └─ profiling/
├─ docs/reports/
├─ artifacts/evaluation_results/
└─ tests/examples/
````

`src`レイアウトを採用しているため、リポジトリ直下のスクリプトや生成物がPython packageへ混入しません。`solvers`と`cable`は互換性・研究再現用であり、新規の本番コードから直接利用しません。


This package is a Python-first reference implementation for discussing BRIDGE/MPRC as an algorithm, not as a Rust runtime benchmark.

## Scope

Implemented:

- `bridge_py.route(G, source, target, mode="auto")`
- `bridge_py.path(G, source, target)`
- Dijkstra
- Bidirectional Dijkstra
- Corridor-based MPRC reference solver
- Process-level parallel corridor search for MPRC
- Process-level parallel benchmark case execution
- Random geometric graph generator
- Grid graph generator
- Benchmark CLI with raw CSV and summary CSV output

Not implemented:

- Thread-level parallelism
- GPU parallel execution
- CH/CCH/ALT/Hub Labeling
- Persistent graph loader formats
- Full SolverGate learning

MPRC records logical `parallel_steps` as the synchronization-step model. When `workers > 1`, corridor searches are executed with Python `ProcessPoolExecutor`, so `time_ms` includes process startup and IPC overhead. For small graphs this can be slower than sequential execution; for larger graphs it prevents the reference benchmark from being dominated by one long serial corridor loop.

## Install for local development

````bash
cd bridge_python
python -m pip install -e .
````

## Minimal API

````python
from bridge_py import Graph, route

G = Graph.from_edges([
    ("A", "B", 1.0),
    ("B", "C", 1.0),
    ("A", "C", 3.0),
])

result = route(G, "A", "C", mode="exact")
print(result.path, result.distance)
````

## MPRC with process workers

````python
from bridge_py.graphs.generators import diagonal_extreme_pair, random_geometric_graph
from bridge_py.solvers.mprc import mprc

G = random_geometric_graph(2000, seed=7, k_neighbors=12)
source, target = diagonal_extreme_pair(G)

result = mprc(G, source, target, workers=4)
print(result.distance, result.telemetry["parallel_backend"])
````

The public `route()` API also accepts the worker count through `constraints`.

````python
result = route(G, source, target, mode="fast", constraints={"workers": 4})
````

## Benchmark

Sequential baseline:

````bash
bridge-py 100 250 500 1000 2000 -o raw.csv --summary summary.csv -f csv --trials 5
````

MPRC corridor-level process parallelism:

````bash
bridge-py 1000 2000 5000 --trials 5 --mprc-workers 4 -o raw.csv --summary summary.csv
````

Independent benchmark case parallelism:

````bash
bridge-py 1000 2000 5000 --trials 5 --benchmark-workers 4 -o raw.csv --summary summary.csv
````

Avoid setting both `--mprc-workers` and `--benchmark-workers` too high at the same time, because that creates nested process pools.

Equivalent module invocation:

````bash
python -m bridge_py.bench.benchmark 100 250 500 -o raw.csv --summary summary.csv --trials 5 --mprc-workers 4
````

## Raw CSV fields

The raw CSV follows the minimum MPRC evaluation schema:

````text
run_id,experiment_id,trial,seed,graph_type,nodes,edges,source,target,query_class,solver_name,found,distance,exact_distance,distance_ratio,exact_match,work_relaxations,work_expanded_nodes,total_work,parallel_steps,time_ms,peak_memory_kib,k_corridors,candidate_count,best_corridor_id,rescue_triggered,repair_triggered,error_code
````

## Fix round: bug fixes and structural improvements

This package includes the post-benchmark repair pass.

### Fixed implementation bugs

- `tracemalloc` is no longer started/stopped independently inside nested solver calls. Nested Dijkstra calls now preserve the outer measurement context.
- The random geometric graph generator now uses `scipy.spatial.cKDTree` when available instead of all-pairs distance sorting.
- MPRC process execution now has an auto gate. For small graphs or small corridor workloads, `workers > 1` falls back to sequential corridor search and records `parallel_backend="sequential_auto_gate"`.
- The multiprocessing consistency test can explicitly bypass the auto gate to verify process execution correctness.

### Structural limitations still present

- Per-query process-level parallelism remains too coarse for small and medium graphs.
- MPRC still spends more total work than bidirectional Dijkstra in the tested RGG condition.
- Candidate count still drops as node count grows, so obstacle, clustered, and adversarial tests remain necessary.
- The current repair policy is still a conservative exact fallback, not a fully bounded local repair system.

## MRPC-CG prototype

This package includes `bridge_py.solvers.mrpc_cg.mrpc_cg`, a compressed-graph MRPC prototype.

```python
from bridge_py.solvers.mrpc_cg import mrpc_cg
result = mrpc_cg(G, source, target, workers=4, max_distance_ratio=1.02)
```

`MRPC-CG` builds connected-component supernodes, creates witness-edge superedges, searches a compressed graph, expands the skeleton path back to the original graph, and uses local repair / exact fallback when unsafe.

The prototype is intended for structural evaluation, not production use. Current evaluation shows low query work in fast mode but insufficient distance quality without fallback.

## DG5 diagnostic telemetry

Cold-start diagnostic run programmatically:

````python
from bridge_py.bench.dg5_diagnostics import diagnose_graph

report = diagnose_graph(graph, source, target, trace_level=2, trace_sample_every=1)
````

`comparison[*].total_time_ms` and `comparison[*].total_work` include preprocessing. Detailed DG5 events are stored in `dg5_telemetry.trace_events`.

CLI:

````bash
bridge-dg5-diagnose graph.json SOURCE TARGET -o dg5_diagnostic.json --trace-level 2 --sample-every 1
````

## DG6 high-load evaluation

`dg6_eval.py` now defaults to the `high` preset. It evaluates grid graphs up to about 50,000 nodes and non-grid graphs up to 50,000 nodes with five seeds per condition.

```powershell
python .\dg6_eval.py
```

Available presets:

```powershell
python .\dg6_eval.py --preset quick
python .\dg6_eval.py --preset standard
python .\dg6_eval.py --preset high
python .\dg6_eval.py --preset stress
```

A practical high-load run that omits the slower legacy solver:

```powershell
python .\dg6_eval.py --preset high --solvers dijkstra,bidir,astar,dg6
```

Custom graph sizes and seed count:

```powershell
python .\dg6_eval.py `
  --grid-sides 20,40,70,100,141,224 `
  --graph-sizes 400,900,2000,5000,10000,20000,50000 `
  --seeds 5 `
  --output evaluation_results/dg6_custom
```

Memory measurement is disabled by default because `tracemalloc` significantly increases runtime. Enable it explicitly with `--measure-memory`.

## Architecture v0.0.1 migration

Public production routing now follows `GATE -> TRUSS -> ANCHOR/BOLTS` and uses
BEARING's `NullObserver` by default. ULTRASOUND and TRAFFIC remain optional
validation packages and are not imported by the production path.

See `IMPLEMENTATION_ARCHITECTURE_SPEC_v0.0.1.md` for implemented contracts and
remaining hard-budget/session limitations.
