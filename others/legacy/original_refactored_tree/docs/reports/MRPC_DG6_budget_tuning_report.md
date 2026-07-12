# MRPC-DG6 Budget Tuning Report

作成日: 2026-07-10
対象: MRPC-DG6 budgeted quality tuning

## 1. 目的

MRPC-DG6について、以下の方針で予算制御と性能チューニングを行った。

- 通常work目標を `node数 / 2` 付近に置く。
- まず有効経路を早く見つける。
- 有効経路が早期に見つかった場合のみ、残予算を品質改善に使う。
- exact source-target oracleは使用しない。
- 評価時のみDijkstraを参照距離として使用する。

## 2. 実装した主な変更

### 2.1 既定予算の再配置

DG6の既定値を以下へ変更した。

| 項目 | 旧値 | 新値 | 意図 |
|---|---:|---:|---|
| `target_work_ratio` | 0.50 | 0.45 | 平均workを下げる |
| `initial_path_budget_ratio` | 0.22 | 0.18 | 初期経路探索を軽量化する |
| `connector_budget_ratio` | 0.22 | 0.16 | connectorの過剰探索を抑える |
| `min_quality_budget_ratio` | 0.06 | 0.06 | 小さすぎる残予算でrepairしない |
| `repair_hops` | 1 | 1 | path-local repairの局所性を維持 |
| `max_repair_nodes_ratio` | 0.22 | 0.22 | repair領域の上限を維持 |

### 2.2 Portal strategyの強化

clustered / portal支配型に対し、long-edge portal skeletonを追加した。

- edge lengthのp95/median・max/medianを特徴量として追加した。
- long edge endpointをportal候補として抽出する。
- portal候補を1-hop拡張し、コンパクトなskeleton領域を作る。
- skeleton領域内でbudgeted local connectorを実行する。

これにより、clusteredのwithin 10%率は 0.833 から 1.000 へ改善した。

### 2.3 Weighted-cost strategyの強化

weighted-noiseでは、`N/2` では10%以内に届かないケースが多かったため、weight-risk時のみ限定的なoverrunを許容した。

- weighted-cost strategyでは、bounded bidirectional connectorの上限を概ね `0.72N` まで拡張する。
- これは全体oracleではなく、weight不整合時のstrategy固有予算としてtelemetryに残る。

これにより、weighted_noiseのwithin 10%率は 0.5 から 1.0 へ改善した。

## 3. チューニング探索

12個の代表設定を比較した。

| config | within 10%率 | 平均距離比 | 最悪距離比 | 平均work | work<=N/2率 |
|---|---:|---:|---:|---:|---:|
| current | 0.987 | 1.008 | 1.118 | 615.7 | - |
| lean_a | 0.987 | 1.008 | 1.118 | 601.4 | 0.705 |
| low_work_045 | 0.987 | 1.008 | 1.118 | 576.7 | 0.756 |

最終的に、品質を維持しながら平均workが最も低い `low_work_045` 相当を既定値として採用した。

## 4. 最終評価結果

評価条件:

- 78 cases
- 比較solver: Dijkstra / Bidirectional Dijkstra / A* / DG5 clean / DG6 tuned
- Oracle quality guard: 無効
- DG6 exact fallback: 無効

### 4.1 全体比較

| solver | valid率 | exact率 | within 10%率 | 平均距離比 | 最悪距離比 | 平均work | 平均step | oracle率 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Dijkstra | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 3559.2 | 756.0 | 0.000 |
| Bidir | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 2810.6 | 688.1 | 0.000 |
| A* | 0.936 | 0.923 | 0.987 | 1.005 | 1.133 | 2663.7 | 596.1 | 0.000 |
| DG5 clean | 1.000 | 0.782 | 0.859 | 1.087 | 2.118 | 5470.8 | 391.8 | 0.000 |
| DG6 tuned | 1.000 | 0.897 | 1.000 | 1.005 | 1.095 | 565.3 | 39.9 | 0.000 |

### 4.2 topology別DG6結果

| topology | strategy | within 10%率 | 平均距離比 | 最悪距離比 | 平均work |
|---|---|---:|---:|---:|---:|
| open | geometric_corridor | 1.000 | 1.000 | 1.000 | 450.0 |
| wall | geometric_corridor | 1.000 | 1.000 | 1.000 | 437.5 |
| u_shape | geometric_corridor | 1.000 | 1.000 | 1.000 | 429.5 |
| culdesac | geometric_corridor | 1.000 | 1.000 | 1.000 | 430.0 |
| random_obstacles | geometric_corridor | 1.000 | 1.000 | 1.000 | 394.7 |
| random_geometric | geometric_corridor | 1.000 | 1.061 | 1.095 | 292.5 |
| clustered | portal | 1.000 | 1.000 | 1.000 | 212.3 |
| scale_free_pos | hub_aware | 1.000 | 1.000 | 1.000 | 216.8 |
| scale_free_no_pos | hub_aware | 1.000 | 1.000 | 1.000 | 216.8 |
| weighted_noise | weighted_cost | 1.000 | 1.008 | 1.045 | 716.8 |
| double_wall | geometric_corridor | 1.000 | 1.000 | 1.000 | 1156.8 |
| spiral | geometric_corridor | 1.000 | 1.000 | 1.000 | 388.5 |
| disconnected | geometric_corridor | 1.000 | 1.000 | 1.000 | 2007.0 |

## 5. 解釈

### 5.1 成功した点

- within 10%率は 1.000 に到達した。
- 平均距離比は 1.005 まで改善した。
- 最悪距離比は 1.095 で、10%目標内に収まった。
- 平均workは 565.3 で、DG5 cleanの約10.3%まで削減された。
- 平均stepは 39.9 で、Dijkstra / Bidirより大幅に小さい。
- clusteredはportal skeletonにより大幅に改善した。
- weighted_noiseはrisk-based overrunにより10%以内へ改善した。

### 5.2 残課題

- `work <= N/2` は全ケースで達成していない。
- weighted_noiseは品質維持のため平均workが `N/2` を超える。
- double_wallは一部ケースでemergency探索によりworkが大きい。
- disconnectedは到達不能証明に探索量が必要で、現在はwork目標から外れやすい。
- Python実時間はまだDijkstraより遅い。特にtracemallocを含むuniform計測ではDG6の集合生成・候補領域生成のコストが大きい。

## 6. 最適予算配置

現時点の推奨配置は以下である。

| 予算カテゴリ | 推奨値 | 役割 |
|---|---:|---|
| total target | `0.45N` | 通常時の総work目標 |
| first path | `0.18N` | 早期有効経路発見 |
| connector | `0.16N` | strategy候補の接続 |
| quality repair threshold | `0.06N` | 残予算が十分ある場合のみrepair |
| weighted-cost overrun | `0.72N` | weight-risk時だけ許可 |
| portal skeleton cap | `0.30N` nodes / `max(0.28N, connector)` expansions | portal支配型対策 |

## 7. 結論

今回のチューニングで、DG6は以下の状態になった。

- 品質: 10%以内目標を達成
- work: 平均では大幅削減、ただし全ケースN/2以内ではない
- 構造: topology別strategyの効き方が明確化
- 差別化: Dijkstraより平均work・stepは小さいが、Python実時間では未達

次の改善対象は、weighted_noiseと到達不能判定のwork削減、およびPython実装の集合生成・telemetry・候補領域生成の高速化である。
