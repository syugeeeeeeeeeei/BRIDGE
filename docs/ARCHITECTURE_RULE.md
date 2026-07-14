# BRIDGE アーキテクチャ規則

**文書ID:** BRIDGE-ARCH-RULE-001  
**版:** v3.0  
**対象:** BRIDGE Go v0.15.0 以降  
**状態:** 規範文書

## 1. 目的

本書は、BRIDGEの責務境界、依存方向、オンライン実行モデル、Work Model v2、証明、観測、ベンチマークおよび文書管理の最上位規則を定める。

経路を返すだけでは正しい実装とみなさない。公開実行経路において、予算、状態遷移、証明、終了状態、観測非干渉および決定論性が一貫して成立しなければならない。

## 2. 規範文書の優先順位

1. `docs/ARCHITECTURE_RULE.md`
2. `docs/WORD_DEFINITION.md`
3. `src/bridge/<component>/COMPONENT_RULE.md`
4. `docs/architecture/`配下の現行仕様
5. `docs/algorithms/`配下の現行アルゴリズム仕様
6. `USAGE.md`
7. `docs/reports/`配下の監査・評価記録
8. `README.md`

報告書は時点記録であり、規範文書を上書きしない。コード、テストおよび規範文書の不一致は欠陥として修正する。

### 2.1 用語更新義務

公開型、公開フィールド、Scenario、trace event、研究指標、証明種別、終了状態またはコンポーネント責務を追加・変更する場合、同一変更で`docs/WORD_DEFINITION.md`を更新する。

曖昧な語を互換目的で残してはならない。特に`exact`、`fallback`、`work`、`step`、`proof`、`completed`は限定語なしで新規使用してはならない。

## 3. コンポーネント構成

```text
外部利用者 / CLI / SDK
          │
          ▼
        GATE
          │
          ▼
        TRUSS
   ┌──────┼───────────┐
   ▼      ▼           ▼
ANCHOR   BOLTS   TRUSS内部サービス
Session  Local        Scheduler
         Capability   BudgetManager
                      TaskRegistry
                      ExecutionEngine
                      SolverRouter
                      EvidenceStore
                      TerminationPolicy
          │
          ▼
         CORE

BEARING ← 各実行コンポーネントの型付きevent
   │
   ▼
ULTRASOUND（任意の観測・再構成）

TRAFFIC → GATE公開API（開発・研究評価専用）
HEALTHY → 保存済み結果・traceの検査と再構成
```

## 4. コンポーネント責務

### 4.1 CORE

共有値型と中立契約を所有する。

- Graph、RouteRequest、RouteResult
- Hypothesis、Region、Checkpoint
- Evidence、ProofClass
- HandoffRequest、HandoffResult
- TerminationStatus
- Work Model v2、WorkMetrics、Budget、Bounds

制御判断、solver選択、証明生成、外部I/Oを持たない。

### 4.2 GATE

外部公開境界を所有する。

- 入力検証と既定値適用
- 外部ID変換
- TRUSS呼出し
- 公開結果とエラーの変換
- 内部証明状態および終了状態の損失なき伝播

GATEは証明状態を推定または捏造してはならない。`budget_exhausted=false`等の間接条件から`reachability_proven`や`optimality_proven`を生成してはならない。

### 4.3 TRUSS

Route Request単位のオンライン調停とPortfolio Work Budgetの唯一の所有者である。

- epoch生成と進行
- Task Registry
- grant、予約、消費、監査
- ANCHOR Sessionの起動・中断・再開
- BOLTS Capabilityの選択とHandoff
- Evidenceの検証・統合
- Upper Bound／Lower Boundの共有
- 終了判定
- 候補の認証と最終選択

TRUSS本体は薄い調停ループとし、コリダー形状、障害物左右分岐、ヒューリスティック内部判断を持たない。

### 4.4 ANCHOR

中断・再開可能な主探索Sessionを所有する。

- HypothesisとRegion
- 局所探索状態
- Candidate、Bound、Checkpoint
- Snapshot／Resume
- 停滞診断
- 次操作提案
- Handoff Resultの適用

ANCHORは総予算、worker管理、BOLTS選択、システム終了方針を所有しない。

### 4.5 BOLTS

Capabilityベースの補助・局所solver群を所有する。

- `CONNECT_CHECKPOINTS`
- `ESCAPE_REGION`
- `REPAIR_SEGMENT`
- `PROVE_UNREACHABLE`
- `TIGHTEN_BOUND`
- `CERTIFY_CANDIDATE`
- Dijkstra、双方向Dijkstra、A*、Weighted A*、Reachability

Reachability Solverは到達可能性のみを証明する。重み付き最短路の最適性を証明したものとして扱ってはならない。

### 4.6 BEARING

型付きevent契約と安全なObserver境界を所有する。探索制御、保存、集計、予算変更を行わない。

### 4.7 ULTRASOUND

eventの収集、保存、replay、Anytime曲線、state reuse、duplicate work、Work再構成を担当する。観測結果を探索順序へ返してはならない。

### 4.8 TRAFFIC

Scenario、dataset、benchmark、ablation、統計、結果不変条件検査を担当する。公開APIのみを使用し、矛盾するRunをfail-closedで拒否する。

### 4.9 HEALTHY

保存済みRun、trace、ledgerからWork、Timing、証明、再利用、決定論性を検査する。探索に介入しない。

## 5. 依存規則

### 許可

```text
CORE       → Go標準ライブラリ
BEARING    → COREの中立型
ANCHOR     → CORE, BEARING
BOLTS      → CORE, BEARING
TRUSS      → CORE, ANCHOR公開契約, BOLTS公開契約, BEARING
GATE       → CORE, TRUSS, BEARING Observer契約
ULTRASOUND → CORE read-only schema, BEARING
TRAFFIC    → GATE公開API, CORE公開schema, ULTRASOUND公開API
HEALTHY    → 保存済みartifact schema, CORE read-only schema
```

### 禁止

- GATEからANCHORまたはBOLTSへの直接依存
- ANCHORからTRUSS、GATE、ULTRASOUND、TRAFFICへの依存
- BOLTSからTRUSS、ANCHOR concrete、GATE、ULTRASOUNDへの依存
- BEARINGからULTRASOUNDへの依存
- ULTRASOUNDまたはHEALTHYから探索制御APIへの依存
- TRAFFICからsolver private stateへの依存
- production packageから`others/legacy`への依存
- package間循環依存

## 6. オンライン実行モデル

### 6.1 Session

ANCHORの内部実行は`NewSession`、`Step`、`Snapshot`、`Resume`、`Progress`、`Result`、`Finished`、`Cancel`で表現する。一括`Solve`を提供する場合もSession adapterでなければならない。

### 6.2 Epoch

TRUSSはepoch境界で以下を行う。

1. runnable taskを登録する
2. Schedulerが公平に選択する
3. BudgetManagerがgrantを発行する
4. ExecutionEngineが実行する
5. 消費Workを監査する
6. Candidate、Bound、Checkpoint、Evidenceを固定順で統合する
7. TerminationPolicyを評価する

同一入力、予算、worker数、seedでは、goroutine完了順に依存せず同一論理結果を返す。

### 6.3 Handoff

Handoffは全体問題のゼロからの再実行ではない。目的、Region、Checkpoint、期待出力、Evidence、Budget、Scopeを持つ局所依頼である。

標準経路は次のとおりである。

```text
ANCHOR pause → TRUSS route → BOLTS local execution
→ Evidence検証 → ANCHOR ApplyHandoff → ANCHOR resume
```

ANCHORとBOLTSが互いを直接起動してはならない。

### 6.4 旧実行経路の禁止

以下を主実行経路として残してはならない。

- ANCHOR一括終了後にBOLTS全体探索をゼロから起動する事後fallback
- 終了済みRouteResultだけを検査するSupervisor中心の制御
- 新サービスを生成するだけでRouteから使用しない飾り実装
- 固定値telemetryによる未計測値の偽装

## 7. Work Model v2

Work Model versionは`2.0`とする。Solver Workだけでなく、意味を持つ制御操作も総Budgetへ課金する。

追加の標準Actionは次のとおりである。

| Action | 意味 |
|---|---|
| `hypothesis` | Hypothesis生成、分岐、凍結、再開、枝刈り |
| `schedule` | task登録、選択、状態遷移、epoch制御 |
| `handoff` | Handoff生成、局所依頼、結果適用 |
| `evidence` | Evidence検証、登録、統合、失効処理 |

次を満たさなければならない。

```text
全Solver Work + 全Control Work <= Portfolio Work Budget
Step consumed <= Step grant
Reported Work = Ledger Work = Traceから再構成した論理Work
```

Telemetry生成、Traceシリアライズ、I/O待ち、GC、壁時計時間はWorkへ含めない。

## 8. Evidenceと証明

### 8.1 ProofClass

- `empirical`: 推定。証明値として使用禁止
- `admissible_lower_bound`: 許容的Lower Bound
- `unreachable`: Scope内の到達不能証明
- `exact`: Candidateの最適性証明

Evidenceは生成Solver、Hypothesis ID、Scope、Generated Work、ProofClass、失効条件を持つ。

### 8.2 証明昇格の禁止

以下を禁止する。

- 経験的推定値をLower Boundへ昇格する
- Reachability成功をOptimalityへ昇格する
- Search CompletedをReachability ProvenまたはOptimality Provenへ昇格する
- `error_code == ""`を証明として扱う
- GATEまたはTRAFFICが内部証明を再推定する

### 8.3 Candidate認証

Arbiterは次の順に比較する。

1. 経路妥当性
2. 証明強度
3. Proven Cost Ratio
4. 距離
5. Work

返却Pathの全辺が存在し、距離再計算値と一致しなければ候補として採用しない。

## 9. 終了状態

`TerminationStatus`は排他的に扱う。

- `FOUND`
- `UNREACHABLE`
- `UNKNOWN_BUDGET`
- `CANCELLED`
- `DEADLINE_EXCEEDED`
- `INVALID_REQUEST`

補助booleanはこの状態と矛盾してはならない。予算不足と到達不能証明を同一視してはならない。

## 10. 観測とTiming

### 10.1 Observer非干渉

Observerの有無でpath、distance、termination、proof、Work、Step、seed消費順、solver選択、budget配分を変更してはならない。

### 10.2 Timing契約

- end-to-end時間はTRAFFICが公開API境界で直接測定する
- solver時間は各solverが全終了経路で記録する
- `timing_valid=true`かつ`solver_ns=0`を禁止する
- 表示用msはnsから導出する
- 無効Timingを性能比較へ使用しない

### 10.3 Warm-up

Warm-upは本計測から除外する。Warm-upではObservation、Collector、Trace保存を無効化し、破棄される高頻度eventを生成しない。

## 11. Benchmark不変条件

TRAFFICは以下を検出した場合、集計を継続せずRunを失敗させる。

- Pathなしで`optimality_proven=true`
- Search未完了で`optimality_proven=true`
- Pathありで`reachability_proven=false`
- Budget不足なのに到達不能または最適性を証明済み
- `NO_PATH`とPathありの併存
- `timing_valid=true`かつsolver時間0
- Path距離と再計算値の不一致

性能値と意味論の正しさを分離する。正しいが遅い実装は性能課題であり、虚偽の証明または無効Timingは研究データを破壊する致命的欠陥である。

## 12. テストとリリースゲート

最低限、以下を実行する。

```text
go test -count=1 ./...
go test -race -count=1 ./...
```

加えて次を証明する。

- Work Budget超過0件
- Snapshot/Resume不一致0件
- Observer差分0件
- 決定論epoch差分0件
- 不正な証明昇格0件
- invalid benchmark Runのfail-closed拒否
- 文書、コード、テストの整合

## 13. 破壊的変更と旧仕様削除

v0.15.0は旧内部仕様との互換を要件としない。互換分岐、deprecated adapter、未使用fallback、dead code、固定値telemetryを残すことより、現行契約を単純かつ一意に保つことを優先する。

公開契約を変更した場合は、互換コードではなく移行ガイドで扱う。
