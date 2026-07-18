# TRUSS コンポーネント規則

**対象package:** `src/bridge/truss`  
**対象版:** v0.15.0以降  
**状態:** 規範文書

## 1. 責務

TRUSSはRoute Request単位のオンライン調停と総Work Budgetの唯一の所有者である。

- Scheduler
- BudgetManager
- TaskRegistry
- ExecutionEngine
- SolverRouter
- EvidenceStore
- TerminationPolicy
- epoch境界でのCandidate、Bound、Evidence統合
- Handoffの生成、routing、結果適用
- Arbiterによる最終候補選択

## 2. 禁止事項

- frontier操作、edge relaxation
- corridor形状やheuristic判断の直書き
- solver private stateの参照
- Observer出力による制御変更
- ANCHOR終了後に全体問題をゼロからBOLTSへ渡す旧事後fallback

## 3. 実行規則

1. runnable taskを登録する
2. Schedulerが公平に選択する
3. BudgetManagerがgrantを発行する
4. ExecutionEngineがworker数に応じて実行する
5. 消費Workを監査する
6. epoch境界で固定順にEvidenceを統合する
7. TerminationPolicyを評価する

ANCHORとBOLTSを直接結合せず、必ずTRUSSがHandoffを仲介する。

## 4. Work規則

- Solver WorkとControl Workの合計を総Budgetへ課金する
- `schedule`、`hypothesis`、`handoff`、`evidence`をWork Model v2のActionとして計上する
- Step consumedがgrantを超えた場合は即時エラーとする
- Reported Work、Budget Ledger、Trace再構成値を一致させる

## 5. 証明・候補規則

Arbiterの比較順は次のとおりである。

1. 経路妥当性
2. 証明強度
3. Proven Cost Ratio
4. 距離
5. Work

Reachability証明をOptimality証明として扱わない。

## 6. 必須テスト

- online epoch統合テスト
- budget境界・grant超過拒否テスト
- ANCHOR→BOLTS→ANCHOR統合テスト
- Evidence固定順統合テスト
- worker数反映テスト
- cancellation・race・決定論性テスト

## 条件付きBOLTS移行

- TRUSSはANCHORを単一Sessionで開始する。
- BOLTSは固定実行せず、停滞または明示的Certification要求時だけ起動する。
- BOLTS移行前後のWork、Handoff回数、State Reuseを記録する。

## 7. 観測規則

- TRUSSはULTRASOUNDへ直接依存せず、内部処理の観測はBEARINGのLifecycle Eventを介して行う。
- minimum観測では、1 Route当たり定数回または条件成立時のみ発生する低頻度境界に限定する。
- Epoch、候補、Node、Edge単位の高頻度Timing Spanをminimumへ追加してはならない。
- 標準の内部区間は `request_adaptation`、`deadline_setup`、`budget_setup`、`observer_setup`、`policy_setup`、`session_creation`、`adaptive_execution`、`final_handoff`、`finalization`、`result_integration` とする。
- `conditional_handoff`および`certification`は、該当処理が実行された場合にのみ発行する。
- 観測無効時にはSpan ID、Timestamp、Payload、Closureその他の観測用動的Allocationを生成してはならない。
- 観測の有無によってPath、Cost、Work、Solver選択、Handoff結果、停止理由を変更してはならない。
