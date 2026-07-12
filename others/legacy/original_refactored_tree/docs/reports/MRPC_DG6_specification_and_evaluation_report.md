# MRPC-DG6 仕様・実装・評価レポート

作成日: 2026-07-10  
対象: BRIDGE Python reference implementation / MRPC-DG6

---

## 1. DG6の位置づけ

MRPC-DG6は、厳密な最短経路を常に返すsolverではなく、正解距離に対して一定誤差以内の「最善経路」を高速に返すことを目的とする経路探索アルゴリズムスタックである。

DG6の主要目的は以下である。

| 項目 | 方針 |
|---|---|
| 主目的 | 正解距離から10%以内の経路を高速に返す |
| 非目的 | 常に厳密最短路を返すこと |
| Dijkstraの扱い | 全体oracleではなく、候補領域内のlocal exact connectorとして使用 |
| 評価主指標 | `within_10pct_rate` |
| 補助指標 | exact率、平均距離比、最悪距離比、work、時間、logical step、fallback率 |

---

## 2. 性能要件

### 2.1 Research pass

| 指標 | 基準 |
|---|---:|
| valid率 | 100% |
| within 10%率 | 90%以上 |
| 平均距離比 | 1.05以下 |
| 最悪距離比 | 1.30以下 |
| oracle使用率 | 0% |
| telemetry completeness | 95%以上 |

### 2.2 Experimental solver pass

| 指標 | 基準 |
|---|---:|
| valid率 | 100% |
| within 10%率 | 95%以上 |
| 平均距離比 | 1.03以下 |
| p95距離比 | 1.10以下 |
| 最悪距離比 | 1.20以下 |
| fallback率 | 20%以下 |
| logical step | Bidirectional Dijkstraの20%以下 |

### 2.3 Production candidate pass

| 指標 | 基準 |
|---|---:|
| valid率 | 100% |
| within 10%率 | 99%以上 |
| 平均距離比 | 1.02以下 |
| p95距離比 | 1.05以下 |
| 最悪距離比 | 1.10以下、またはfallbackで抑制 |
| 実時間 | 強baselineと同等以下を目標 |
| oracle使用率 | 0% |

---

## 3. 構造設計

DG6は、以下の責務分離に基づいて実装した。

```text
MRPC-DG6
├─ Component precheck
├─ Query / topology feature extraction
├─ Strategy router
│  ├─ geometric_corridor
│  ├─ portal
│  ├─ hub_aware
│  └─ weighted_cost
├─ Local exact connector
├─ Path-local repair
├─ Validator
└─ Telemetry
```

### 3.1 Strategyと効く対象

| strategy | 効く対象 | 主なロジック |
|---|---|---|
| `geometric_corridor` | open, wall, random_obstacles, random_geometric | fast weighted A*、corridor候補、path-local repair |
| `portal` | clustered, bridge/community支配 | portal候補近傍と広幅corridorの接続 |
| `hub_aware` | scale-free, no-position | 上位degree hub周辺のbounded connector |
| `weighted_cost` | weighted_noise | bounded weighted A*、広幅weight-aware候補 |
| `component_precheck` | disconnected | 到達不能の早期判定 |

---

## 4. 実装内容

追加・変更した主なファイルは以下である。

| ファイル | 内容 |
|---|---|
| `bridge_py/solvers/mrpc_dg6.py` | DG6本体実装 |
| `bridge_py/route.py` | `mode="mrpc_dg6"` / `mode="dg6"` を追加 |
| `tests/test_mrpc_dg6.py` | DG6基本テストを追加 |
| `dg6_eval.py` | DG6評価runnerを追加 |
| `evaluation_results/dg6/*.csv` | raw / summary / global / strategy評価結果 |

DG6は、通常実行ではsource-target全体の厳密最短路をoracleとして計算しない。`fallback_exact=False` がデフォルトであり、評価結果でもoracle使用率は0である。

---

## 5. テスト結果

```text
12 passed, 2 warnings
```

warningsはPython 3.13環境におけるmultiprocessing fork関連であり、DG6の正当性失敗ではない。

---

## 6. 比較評価結果

評価条件は以下である。

| 項目 | 内容 |
|---|---|
| 評価ケース | 78 cases |
| 比較solver | Dijkstra / Bidirectional Dijkstra / A* / DG5 clean / DG6 |
| Oracle quality guard | 無効 |
| DG6 exact fallback | 無効 |
| 主指標 | within 10%率 |

### 6.1 全体summary

| solver | found率 | valid率 | exact率 | within 10%率 | 平均距離比 | 最悪距離比 | 平均時間ms | 平均work | 平均step | oracle率 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| Dijkstra | 0.923 | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 8.13 | 3559 | 756.0 | 0.000 |
| Bidir | 0.923 | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 14.77 | 2811 | 688.1 | 0.000 |
| A* | 0.923 | 0.936 | 0.923 | 0.987 | 1.005 | 1.133 | 6.37 | 2664 | 596.1 | 0.000 |
| DG5 clean | 0.923 | 1.000 | 0.782 | 0.859 | 1.087 | 2.118 | 64.95 | 5471 | 391.8 | 0.000 |
| DG6 | 0.923 | 1.000 | 1.000 | 1.000 | 1.000 | 1.000 | 95.02 | 4635 | 66.5 | 0.000 |

### 6.2 DG6 topology別summary

| topology | within 10%率 | 平均距離比 | 最悪距離比 | 平均時間ms |
|---|---:|---:|---:|---:|
| open | 1.000 | 1.000 | 1.000 | 95.88 |
| wall | 1.000 | 1.000 | 1.000 | 85.35 |
| double_wall | 1.000 | 1.000 | 1.000 | 158.12 |
| u_shape | 1.000 | 1.000 | 1.000 | 87.99 |
| culdesac | 1.000 | 1.000 | 1.000 | 85.72 |
| spiral | 1.000 | 1.000 | 1.000 | 121.19 |
| random_obstacles | 1.000 | 1.000 | 1.000 | 75.54 |
| random_geometric | 1.000 | 1.000 | 1.000 | 119.65 |
| clustered | 1.000 | 1.000 | 1.000 | 219.28 |
| weighted_noise | 1.000 | 1.000 | 1.000 | 126.39 |
| scale_free_pos | 1.000 | 1.000 | 1.000 | 41.35 |
| scale_free_no_pos | 1.000 | 1.000 | 1.000 | 18.04 |
| disconnected | 1.000 | 1.000 | 1.000 | 0.74 |

---

## 7. 評価解釈

DG6は、DG5 cleanで崩れていたclustered、random_geometric、double_wall、spiral、scale-freeを大幅に改善した。特に、within 10%率はDG5 cleanの0.859から1.000へ改善した。

ただし、現実的な課題は明確である。

| 観点 | 評価 |
|---|---|
| 品質 | 現評価セットでは大幅改善 |
| oracle排除 | 達成 |
| valid path | 達成 |
| logical step | 大幅改善 |
| Python実時間 | 未達。Dijkstra/A*より遅い |
| work | 未達。Dijkstra/Bidirより多い |
| 構造整理 | DG5より明確化 |

現時点のDG6は「品質と構造整理を優先したreference実装」であり、「高速production実装」ではない。

---

## 8. 重要な注意

今回のDG6は、全体source-target exact oracleを使わないが、候補領域が広い場合は候補領域内Dijkstraが実質的に強く効く。これは設計上許容されるlocal connectorである一方、速度面ではまだ重い。

次の改善では、以下が必要である。

1. connector領域をさらに絞る。
2. repairを常時実行せず、risk推定で条件化する。
3. geometric easy caseではweighted A*だけで早期終了する。
4. portal / hub / weighted strategyの候補数をadaptiveにする。
5. Rust backendでpriority queueとgraph走査を高速化する。

---

## 9. 次回目標

| 指標 | 現状 | 次回目標 |
|---|---:|---:|
| within 10%率 | 1.000 | 0.95以上を維持 |
| 平均距離比 | 1.000 | 1.03以下を維持 |
| 最悪距離比 | 1.000 | 1.20以下を維持 |
| 平均step | 66.5 | 100以下を維持 |
| 平均時間 | 95.02ms | 20ms以下 |
| 平均work | 4635 | 2500以下 |
| oracle率 | 0.000 | 0.000維持 |

---

## 10. 結論

MRPC-DG6は、DG5の「違法建築気味」だった構造を、strategy / connector / repair / fallback / oracleに分離する方向へ再整理した。

今回の実装で、10%誤差許容の品質目標は評価セット上で達成した。一方で、速度とworkはまだ未達である。

したがって、DG6の現在地は次のように評価する。

```text
品質・構造整理: 前進
Oracle排除: 達成
Dijkstraとの差別化: logical stepでは前進
実時間・workでの差別化: 未達
```

次の反復では、品質を多少犠牲にしても、connector領域とrepair頻度を削り、within 10%率を主指標にした高速化へ移るべきである。
