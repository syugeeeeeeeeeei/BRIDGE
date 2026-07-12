# HEALTHY Component Rule

HEALTHYは、TRAFFICが生成したartifactを読み取り、正当性、Work計上、比較、回帰を評価するread-onlyコンポーネントである。

## 許可依存
- COREの公開型
- TRAFFICの公開artifact型・graph再構成API
- GATEの公開ExecuteOnce API

## 禁止
- ANCHOR、BOLTS、TRUSSのprivate implementationへの依存
- solverの探索順、予算、停止条件の変更
- Exact Referenceのcandidate探索への注入
- artifactの書換え
- invalid Runの性能集計への混入
- HEALTHY独自のWork定義

評価結果は同一Run中の探索判断へ戻してはならない。

## Work検証規則
- Level 1は`WorkMetrics`の保存則とStep不変条件を検証する。
- Level 2はTRUSSのBudget Ledgerを用いてtask・component・portfolioの消費量を照合する。
- Level 3は完全なprofile traceからReconstructed Workを生成してReported Workと照合する。
- sampling、drop、truncation、SHA-256不一致があるtraceを完全検証済みとして扱ってはならない。
- `WorkerCount`はAction traceから再構成するWorkではないため、Reported WorkとのAction一致判定対象外とする。
