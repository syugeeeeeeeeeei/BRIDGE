# Adaptive Handoff 改修 実装・監査報告

## 実装判断

詳細検証で否定された`node_count × 2`の救済Budgetを撤回し、BOLTSをANCHOR Frontierからの局所継続探索として実行するよう変更した。Handoff時の状態利用については、転送しただけの状態を「再利用」と数えず、Queue投入、実展開、最終Path寄与を個別に記録する。

## 主要変更

1. BOLTS Grantを`min(残Budget, max(64,node_count/4), transferred×8+64)`へ制限。
2. Seeded BOLTSへ`RequireSeed`を追加。Frontierなしでは始点から再探索しない。
3. ANCHORのsettled配列をBOLTS Closedへコピーしない。より良いg-scoreによる再Openを妨げない。
4. Reject率50%以上ではWeight 1.0、通常停滞では1.12を使用。
5. Handoff理由を`heuristic_misdirection`、`frontier_explosion`等へ分離。
6. `queued_seed_state_units`、`expanded_seed_state_units`、`path_contributing_seed_state_units`を追加。
7. `reused_state_units`をPath長ではなく、実際に展開したSeed状態数へ修正。

## 監査結果

- `go test ./...`: 全成功
- Benchmark: 162 Run完走
- Found率: 全Algorithm 100%

### 全Scenario平均

| Algorithm | 平均Work | 平均Solver時間 |
|---|---:|---:|
| BRIDGE | 1336.39 | 0.2190 ms |
| ANCHOR | 1439.87 | 0.2111 ms |
| Weighted A* | 2615.33 | 1.4332 ms |

BRIDGEはANCHOR比で平均Workを約7.2%、Weighted A*比で約48.9%削減した。Solver時間はWeighted A*より大幅に短い一方、ANCHOR比では約3.7%長く、Orchestration削減は引き続き必要である。

### 問題トポロジー

1000ノードでは、以前数千Work規模だったWall/U-shapeの追加Workが次まで縮小した。

- Grid Wall: ANCHOR 2045.3、BRIDGE 2150.0、差+104.7
- Grid U-shape: ANCHOR 2216.0、BRIDGE 2237.3、差+21.3

完全な非劣化には未到達だが、無条件Budget拡大による致命的な悪化は解消した。

## 厳格な結論

今回の修正は、BOLTS全体再探索と過大Budgetという主要欠陥を除去し、平均WorkのANCHOR非劣化とFound率100%を達成したため採用可能である。ただし、Wall 1000の局所的Work悪化とANCHOR比の時間オーバーヘッドは残る。Candidate区間改善、双方向局所接続、5000ノードStressは独立Ablationなしに完了扱いしない。
