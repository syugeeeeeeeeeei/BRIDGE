# TRUSS内部観測 改修報告

## 目的

アーキテクチャルール「観測はアルゴリズムを説明するために存在し、アルゴリズムの振る舞いを決定してはならない」に従い、TRUSS内部の固定オーバーヘッドをBEARING経由でULTRASOUNDから分析可能にした。

## 追加したLifecycle Span

minimum観測で取得する低頻度区間:

- `request_adaptation`
- `route`
- `deadline_setup`
- `budget_setup`
- `observer_setup`
- `policy_setup`
- `session_creation`
- `adaptive_execution`
- `final_handoff`
- `finalization`
- `result_integration`

条件成立時のみ取得する区間:

- `conditional_handoff`
- `certification`

ANCHORの既存`solve` SpanはTRUSS `route`の子Spanとして維持する。

## 性能非干渉

- `bearing.StartLifecycle`を使用し、観測無効時はSpan ID、Timestamp、Payload、Closureを生成しない。
- Epoch、Node、Edge、候補単位のTiming Spanはminimumへ追加していない。
- `search_started`、`search_finished`、`candidate_submitted`は購読可否を確認してからPayloadを生成する。
- Lifecycle Spanは正常終了・エラー終了の双方で閉じる。

## テスト

- `go test -count=1 ./...`: PASS
- TRUSS内部の必須Spanがminimumで取得されること: PASS
- 未完了Span、重複Start、孤立Completeが0件であること: PASS
- 既存のPath、Cost、Work、決定性テスト: PASS

## 分析可能になった内容

ULTRASOUNDのSpan結果から、TRUSS `route`を以下の低頻度区間へ分解できる。

- 入力適応
- Deadline準備
- Budget準備
- Observer準備
- Policy準備
- Session生成
- Adaptive実行
- 条件付きHandoff
- 最終Handoff
- Certification
- Result統合
- Finalization

これにより、TRUSS全体とSolver時間の差分だけでなく、固定費の発生区間を直接比較できる。
