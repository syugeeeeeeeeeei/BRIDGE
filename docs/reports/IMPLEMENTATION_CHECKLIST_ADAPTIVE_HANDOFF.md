# Adaptive Handoff 改修 実装・監査チェックリスト

## A. 正しさ
- [x] ANCHOR終了時にCandidateがなければ最終Handoffを評価する
- [x] Handoffは1 Runにつき最大1回とし、同一救済の重複実行を禁止する
- [x] BOLTS WorkをBRIDGE Total WorkおよびBudget Ledgerへ加算する
- [x] Local continuation失敗時に始点から暗黙再探索しない
- [x] Found率回帰をMatrixで検査する

## B. BOLTS Budget
- [x] `node_count × 2`の無条件救済Budgetを撤回する
- [x] Graph規模・転送状態量・残Budgetの最小値でGrantを制限する
- [x] 基本上限を`max(64, node_count / 4)`とする
- [x] 転送状態に対する上限`transferred_state_units × 8 + 64`を適用する
- [x] Handoff制御WorkとBOLTS探索Workを分離計上する

## C. 局所継続探索
- [x] ANCHOR FrontierをBOLTS初期Queueへ変換する
- [x] ANCHOR固有priorityを流用せず、BOLTS側でpriorityを再計算する
- [x] g-score・predecessorを引き継ぐ
- [x] ANCHORのsettled状態をBOLTSの確定Closedとして固定しない
- [x] Frontierがない場合はLocal continuationを不成立として終了する
- [x] BOLTSが始点から全体探索へ暗黙フォールバックしないことを単体テストする

## D. Solver選択
- [x] 通常停滞ではSeeded Weighted A*を使用する
- [x] Reject率50%以上の誤誘導兆候ではWeight 1.0へ下げる
- [x] Frontier膨張を型付きHandoff理由として記録する
- [x] `no_candidate_stagnation`、`heuristic_misdirection`、`frontier_explosion`、`anchor_exhausted`を区別する

## E. 状態再利用計測
- [x] 転送可能状態量を記録する
- [x] BOLTS Queueへ投入したSeed状態量を記録する
- [x] 実際に展開したSeed状態量を記録する
- [x] 最終Pathへ寄与したSeed状態量を記録する
- [x] `reused_state_units`をPath長ではなく実展開Seed数に基づける
- [x] 集計値をHandoffMetricsへ反映する

## F. Runtime/Work監査
- [x] Work Actionの定義・重みは変更しない
- [x] ANCHOR・BOLTS・TRUSS Workを分離する
- [x] ANCHOR・BOLTS・Orchestration時間を分離する
- [x] debug観測がminimumモードへ侵入しない

## G. テスト
- [x] SeedなしLocal continuationが0 Workで終了する単体テスト
- [x] Frontier SeedからPathを復元できる単体テスト
- [x] Seed展開計測の単体テスト
- [x] `go test ./...`全成功
- [x] 100/500/1000ノード、6トポロジー、3 Seed、3 Algorithmの162 Run完走
- [x] 全Benchmark RunでFound率100%
- [x] Weighted A*比平均Work 20%以上削減
- [x] ANCHOR比平均Workを悪化させない

## H. 採用保留
- [ ] Candidate区間Targeted Improvement
  - 今回の主問題はCandidateなしHandoffの全体再探索であり、独立Ablationなしに混入させない。
- [ ] Bidirectional Dijkstraによる局所区間接続
  - 両端Frontier契約を導入する別変更として検証する。
- [ ] 5000ノードRandom Geometricの通常回帰Matrix組込み
  - 長時間化を隠さないため、Timeout付きStress Suiteとして分離する。

保留項目は未実装を隠すための例外ではなく、今回の修正と効果を混同しないための独立検証対象である。
