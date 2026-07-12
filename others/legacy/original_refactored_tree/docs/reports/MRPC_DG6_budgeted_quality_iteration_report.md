# MRPC-DG6 Budgeted Quality Iteration Report
作成日: 2026-07-10
## 1. 目的
DG6を「経路発見を先に行い、早く見つかった場合だけ残り予算で品質改善する」構造へ変更した。通常予算の目標は `node数 / 2` とし、workはedge relaxationではなくnode expansion / node touch中心に再定義した。
## 2. 実装変更
| 変更 | 内容 | 狙い |
|---|---|---|
| `target_work_ratio` | DG6Configに `target_work_ratio=0.50` を追加 | 通常探索のwork目標をN/2へ固定 |
| `fast-first path` | beam A* / greedy geometric pathで最初の有効経路を優先 | 最初の経路発見を高速化 |
| `residual quality budget` | first_path_work後の残予算だけpath-local repairへ投入 | 早く見つかったケースでのみ品質改善 |
| `budgeted connector` | connector Dijkstraをmax_expansions付きに変更 | local exactが全体Dijkstra化するのを抑制 |
| `weighted-cost` | bounded bidirectional connectorを追加 | weight noiseでの品質改善 |
| `emergency approximate path` | 通常予算で未発見の場合のみbeam/bidirの追加探索を明示 | 可用性維持。ただしbudget overrunとしてtelemetry化 |
| `component precheck` | production defaultでは無効、明示指定時のみ実行 | connected easy caseのcold-start workを削減 |

## 3. テスト

```text
12 passed, 2 warnings
```

warningsはPython 3.13のmultiprocessing fork関連であり、DG6の経路正当性失敗ではない。
## 4. 全体評価
| solver | valid率 | exact率 | within 10%率 | 平均距離比 | 最悪距離比 | 平均work | 平均時間ms | 平均step |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| dijkstra | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 3559.2 | 8.20 | 756.0 |
| bidir | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 2810.6 | 15.46 | 688.1 |
| astar | 0.936 | 0.923 | 0.987 | 1.005 | 1.133 | 2663.7 | 6.33 | 596.1 |
| dg5_clean | 1.000 | 0.782 | 0.859 | 1.087 | 2.118 | 5470.8 | 64.42 | 391.8 |
| dg6 | 1.000 | 0.782 | 0.910 | 1.024 | 1.318 | 704.7 | 70.48 | 14.0 |

## 5. DG6のwork目標達成状況
- reachable case数: 72
- reachableにおけるwithin 10%率: 0.903
- reachableにおけるfound率: 0.986
- reachableにおける平均距離比: 1.026
- reachableにおける最悪距離比: 1.318
- reachableにおける平均work: 590.4
- reachableにおけるwork/node中央値: 0.500
- reachableにおけるwork<=N/2達成率: 0.611
- reachableにおけるemergency approximate path使用率: 0.222

## 6. topology別DG6評価
| topology | found率 | exact率 | within 10%率 | 平均距離比 | 最悪距離比 | 平均work | 平均時間ms |
|---|---:|---:|---:|---:|---:|---:|---:|
| clustered | 0.833 | 0.167 | 0.333 | 1.140 | 1.318 | 974.2 | 149.89 |
| culdesac | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 477.5 | 44.67 |
| disconnected | 0.000 | 1.000 | 1.000 | 1.000 | 1.000 | 2076.0 | 144.21 |
| double_wall | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 1217.0 | 129.80 |
| open | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 500.0 | 45.38 |
| random_geometric | 1.000 | 0.000 | 1.000 | 1.061 | 1.095 | 325.0 | 57.60 |
| random_obstacles | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 439.3 | 36.63 |
| scale_free_no_pos | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 243.0 | 9.54 |
| scale_free_pos | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 243.0 | 29.25 |
| spiral | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 1203.5 | 105.36 |
| u_shape | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 476.5 | 44.39 |
| wall | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 486.0 | 49.53 |
| weighted_noise | 1.000 | 0.000 | 0.500 | 1.127 | 1.292 | 500.0 | 70.02 |

## 7. 判断
N/2予算の導入により、DG6の平均workは旧DG6の4635から約705へ大幅に下がった。一方で、旧DG6のwithin 10%率1.0は維持できず、全体では約0.91となった。これは、local exact connectorを強く使っていた旧DG6から、予算制約付きの自立探索へ寄せたためである。

特に改善された点は、open / wall / u_shape / culdesac / random_obstacles / random_geometric / scale-free / double_wall / spiralである。一方、clusteredとweighted_noiseはまだ品質・可用性のボトルネックであり、portal strategyとweighted-cost strategyの再設計が必要である。

## 8. 次の改善対象
| 優先度 | 対象 | 改善内容 |
|---:|---|---|
| 1 | clustered | portal候補抽出をdegreeだけでなくcluster境界・bridge-like edge・双方向近傍frontierで行う |
| 2 | weighted_noise | bounded bidirectionalだけでなくmulti-start low-cost basin searchを追加する |
| 3 | emergency overrun | 通常予算内で失敗するケースを減らし、emergency使用率を20%未満へ下げる |
| 4 | Python時間 | feature抽出とcorridor生成をsampling化し、telemetryをopt-in化する |
