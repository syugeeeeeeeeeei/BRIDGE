# MRPC-DG6 Runtime Bottleneck Analysis and Optimization Report

作成日: 2026-07-10
対象: MRPC-DG6 budget tuned 実装
目的: 実行時間ボトルネックを定量分析し、単一query実行時間とbatch実行時間を削減する

---

## 1. 要約

DG6 tuned の実行時間が長く見えていた最大要因は、アルゴリズム本体ではなく、通常実行中に `tracemalloc` を起動していたことだった。

`tracemalloc` は各allocationを追跡するため、DG6のように小さな `set`、`dict`、`list`、priority queueを多数作る実装では極端に重い。今回、production defaultではメモリ計測を無効化し、必要時のみ `measure_memory=True` で有効化する設計に変更した。

追加で、以下のホットスポットも削減した。

- topology / query feature extraction の全edge走査をsampling化
- corridor生成の `offset数 × 全node走査` を `全node 1回走査` に変更
- weighted A* のweight/geometry ratio再計算をsampling化
- portal検出は品質criticalなため全edge scanを維持しつつ、他の重複scanを削減
- 評価スクリプト側のuniform memory tracingをデフォルト無効化

---

## 2. 実行時間の改善

### 2.1 主要比較

| 条件 | DG6平均時間 | 備考 |
|---|---:|---|
| DG6 tuned 旧評価 | 98.07 ms | 内部 `tracemalloc` + 評価側 `tracemalloc` の影響が大きい |
| 旧DG6 production相当 | 約9.40 ms | `tracemalloc` をmonkey patchで無効化した測定 |
| 今回DG6 optimized | 5.60 ms | production default、評価側memory tracingなし |
| DG6 optimized + function profiler | 5.11 ms | wrapper profiler込み、78 case合計398.9 ms |

実用上のDG6平均時間は、約98 msではなく、production設定では約5.6 msまで下がった。

---

## 3. 品質・workを維持できているか

78ケース評価結果は以下の通り。

| solver | valid率 | exact率 | within 10%率 | 平均距離比 | 最悪距離比 | 平均work | 平均時間ms | 平均step |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Dijkstra | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 3559.2 | 7.96 | 756.0 |
| Bidirectional Dijkstra | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 2810.6 | 14.76 | 688.1 |
| A* | 0.936 | 0.923 | 0.987 | 1.005 | 1.133 | 2663.7 | 6.17 | 596.1 |
| DG5 clean | 1.000 | 0.782 | 0.859 | 1.087 | 2.118 | 5470.8 | 10.81 | 391.8 |
| DG6 optimized | 1.000 | 0.897 | 1.000 | 1.005 | 1.095 | 564.0 | 5.60 | 39.9 |

DG6 optimized は、within 10%率を100%に保ったまま、平均workをDijkstra比で約15.8%、平均stepをDijkstra比で約5.3%まで削減した。

---

## 4. 関数単位のボトルネック

production設定での関数単位profileは以下。

| function | calls | total_ms | mean_ms | 全体比 |
|---|---:|---:|---:|---:|
| `_portal_strategy` | 29 | 128.94 | 4.45 | 32.3% |
| `_geometric_corridor_strategy` | 56 | 117.18 | 2.09 | 29.4% |
| `_corridor_nodes` | 85 | 87.44 | 1.03 | 21.9% |
| `_graph_features` | 78 | 73.90 | 0.95 | 18.5% |
| `_long_edge_portal_endpoints` | 29 | 68.68 | 2.37 | 17.2% |
| `_beam_astar_path` | 95 | 59.17 | 0.62 | 14.8% |
| `_weighted_cost_strategy` | 26 | 34.93 | 1.34 | 8.8% |
| `_budgeted_bidirectional_dijkstra` | 26 | 34.49 | 1.33 | 8.6% |
| `_budgeted_dijkstra` | 278 | 27.51 | 0.10 | 6.9% |

注意: strategy関数の時間には内部関数が含まれるため、割合は排他的ではない。

---

## 5. topology別の実行時間

| topology | 平均時間ms | 平均work | 平均距離比 | 最悪距離比 | 主な理由 |
|---|---:|---:|---:|---:|---|
| scale_free_no_pos | 1.86 | 216.8 | 1.000 | 1.000 | hub候補が効きやすい |
| random_geometric | 2.76 | 292.5 | 1.061 | 1.095 | lightweight geometric探索で完了 |
| random_obstacles | 2.84 | 394.7 | 1.000 | 1.000 | beam first pathで完了 |
| u_shape | 2.99 | 429.5 | 1.000 | 1.000 | geometric corridorが効く |
| wall | 3.09 | 437.5 | 1.000 | 1.000 | geometric corridorが効く |
| open | 3.41 | 450.0 | 1.000 | 1.000 | easy caseだがfeature/corridor固定費あり |
| culdesac | 3.42 | 430.0 | 1.000 | 1.000 | geometric corridorが効く |
| weighted_noise | 3.99 | 716.8 | 1.008 | 1.045 | bounded bidirectional weighted connector |
| clustered | 7.19 | 195.0 | 1.000 | 1.000 | long-edge portal scanが支配的 |
| spiral | 8.03 | 388.5 | 1.000 | 1.000 | beam / detour形状が重い |
| double_wall | 11.31 | 1156.8 | 1.000 | 1.000 | emergency beamが必要 |
| disconnected | 20.72 | 2007.0 | 1.000 | 1.000 | 到達不能証明が重い |

---

## 6. 変更内容

### 6.1 memory tracingをproduction defaultから外した

`DG6Config` に以下を追加した。

````text
measure_memory: bool = False
````

`measure_memory=True` のときだけ `tracemalloc` を使う。通常のroute APIも `measure_memory=False` がデフォルトになっている。

### 6.2 `_graph_features` を軽量化した

変更前は、edge weight / geometry ratio と edge length 分布を全edgeに対して別々に走査していた。

変更後は、degree統計は全nodeで取り、edge統計はdeterministic samplingで取る。

効果:

- 旧production相当profile: `_graph_features` 約300.75 ms / 78 case
- 新profile: `_graph_features` 約73.90 ms / 78 case
- 約75%削減

### 6.3 `_corridor_nodes` を1-pass化した

変更前は、offsetごとに全nodeを走査していた。

変更後は、全nodeを1回だけ走査し、各offsetに対する所属判定を同じloop内で行う。

効果:

- 旧production相当profile: `_corridor_nodes` 約204.38 ms / 78 case
- 新profile: `_corridor_nodes` 約87.44 ms / 78 case
- 約57%削減

### 6.4 weighted A* のedge ratio計算をsampling化した

weighted A*内部で毎回全edgeからweight/geometry比を再計算していたため、samplingに変更した。

### 6.5 portal検出は精度優先で全edge scanを維持した

clusteredでwithin 10%率を維持するには、long-edge portalを落とさないことが重要だった。sampling化すると900 node clusteredの一部caseで距離比1.118となり、10%目標を超えた。

そのため、`_long_edge_portal_endpoints` は全edge scanに戻した。これは速度より品質を優先した箇所である。

---

## 7. マルチプロセス評価

単一query内部でのmultiprocessingは、現時点では優先度が低い。

理由:

- 1 queryのDG6 production平均時間が約5.6 msまで下がった
- Graph、allowed node set、path候補をprocess間でpickleすると通信コストが大きい
- intra-query candidate並列は、fork済みworker poolと共有graphが前提でないと効果が出にくい

一方、複数queryやbenchmark batchではmultiprocessingが有効だった。

78 query batchでの測定:

| 実行方式 | wall-clock | speedup |
|---|---:|---:|
| sequential | 742.4 ms | 1.00x |
| ProcessPool 2 workers | 456.3 ms | 1.63x |
| ProcessPool 4 workers | 322.4 ms | 2.30x |
| ProcessPool 8 workers | 467.7 ms | 1.59x |

今回の環境では4 workersが最良だった。8 workersではprocess overheadとCPU contentionで悪化した。

---

## 8. 残るボトルネック

### 8.1 disconnected

到達不能ケースは、経路がないことを証明するために探索が広がる。平均workは2007で、時間も20.7 msと最も重い。

改善案:

- component indexをgraph build時に持つ
- static graphではconnected component IDをcacheする
- dynamic graphではlazy invalidationを導入する

### 8.2 double_wall

平均workが1156.8で重い。原因はemergency beamが発生しやすいこと。

改善案:

- detour episode strategyを独立実装する
- wall crossing候補をgeometryから先に作る
- emergency beamではなく、obstacle boundary followingを導入する

### 8.3 clustered / portal

品質維持のためlong-edge portal scanを全edgeで行っている。

改善案:

- graph build時にedge length percentileをcacheする
- long edge endpointsをtop-k cache化する
- repeated queryではportal indexを再利用する

### 8.4 corridor generation

1-pass化後も `_corridor_nodes` は全体の約21.9%を占める。

改善案:

- spatial grid index / k-d treeでcorridor付近nodeだけを列挙する
- repeated queryではnode座標配列をNumPy化する
- Rust backendで距離計算をtight loop化する

---

## 9. 次の実装優先度

1. component index cacheを導入する。
2. portal index cacheを導入する。
3. detour episode strategyをdouble_wall / spiral向けに分離する。
4. corridor node selectionをspatial index化する。
5. batch / multi-query APIにProcessPool実行を正式搭載する。
6. Rust backendでgraph scan、corridor判定、priority queueを置き換える。

---

## 10. 結論

DG6の実行時間が長く見えていた主因は、通常実行中のmemory tracingだった。production defaultから外したことで、DG6の平均時間は約98 msから約5.6 msまで下がった。

さらに、feature extractionとcorridor generationを軽量化し、within 10%率100%、平均距離比1.005、最悪距離比1.095、平均work564を維持した。

現時点のDG6は、Python reference実装でもDijkstra平均時間7.96 msを下回った。ただし、A*平均6.17 msとの差は小さく、hidden seedや大規模graphでの追加評価が必要である。
