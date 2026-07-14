# BRIDGE 改善適用・ベンチマークレポート

## 適用した改善

- Session終了時の最終Handoff
- ノード数比率による最小観察Work（25%、64–512 WorkにClamp）
- Frontier・Reject率を使った進捗ベースHandoff
- 通常512 / 監視128 / 確認64 Workの可変Epoch
- ANCHORからBOLTSへのg-score、predecessor、settled、Frontier引継ぎ
- BOLTS側で優先度を再計算するSeeded Weighted A*
- Handoff理由の`no_candidate_stagnation` / `anchor_exhausted`分離

## 今回未適用

- Candidate区間だけを対象とする局所BOLTS
- Handoff理由別のA*・双方向Dijkstra選択
- Random Geometric向けTargeted Improvement
- TRUSS Direct Fast Pathの構造的実装

これらは、状態引継ぎの効果を分離するため次段階とした。

## テスト

- `go test ./...`: 全パッケージ成功
- 100 / 500 / 1000ノード
- 6トポロジー
- Seed 11 / 23 / 37
- 6アルゴリズム
- Warmup込み648 Run、評価対象324 Run

## 全体結果

| 指標 | 改善前BRIDGE | 改善後BRIDGE | 判定 |
|---|---:|---:|---|
| Found率 | 100% | 100% | 維持 |
| 平均Work | 1,583.8 | 1,641.7 | 3.7%悪化 |
| 平均Solver時間 | 0.3186 ms | 0.2671 ms | 16.2%改善 |
| 平均Gap | 1.107% | 0.774% | 改善 |
| 最大Gap | 7.974% | 6.921% | 改善 |
| Handoff発生Run | 36/54 | 28/54 | 22.2%減少 |

Weighted A*との比較（改善後）:

- Work: 37.2%削減
- Solver時間: 20.5%削減
- Found率: 同等の100%
- 平均Gap: Weighted A* 0.248%に対しBRIDGE 0.774%
- 最大Gap: Weighted A* 3.224%に対しBRIDGE 6.921%

## 判断

改善後は時間目標（Weighted A*比15%以上削減）を達成し、Found率を維持し、品質も改善した。一方、改善前BRIDGEよりWorkが3.7%増えたため、今回の実装全体をそのまま採用することはできない。

Work悪化は主にU-shapeとWallで発生した。状態引継ぎ自体は有効だが、救済Budgetを`2 × node_count`まで許容したため、不要または過大なBOLTS継続探索が発生した。次段階では、局所BOLTS、残Budget上限、Handoff利得判定が必要である。

## 結論

- 可変Epoch・Handoff抑制: 採用候補
- 最終Handoff: 正しさのため採用
- 状態引継ぎ: 継続改善対象
- 救済Budget拡大: 却下または厳格化
- 現版全体: 性能目標達成版としては不採用
