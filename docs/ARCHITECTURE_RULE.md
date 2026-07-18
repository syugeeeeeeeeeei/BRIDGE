# BRIDGE アーキテクチャ規則

**文書ID:** BRIDGE-ARCH-RULE-001  
**版:** v3.1  
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

TRAFFIC / GATE / TRUSS / ANCHOR / BOLTS
   │ 型付き意味event
   ▼
BEARING（契約・購読判定・時刻付与・配送）
   │
   ▼
ULTRASOUND（任意の収集・集計・再構成）

TRAFFIC → GATE公開API（開発・研究評価専用）
HEALTHY → 保存済み結果・traceの検査と再構成
```

### 3.1 コンポーネント階層

- メインコンポーネントは、BRIDGE全体で独立した責務、依存規則、不変条件および`COMPONENT_RULE.md`を持つ。
- サブコンポーネントは、メインコンポーネントまたは製品境界の内部責務を変更理由ごとに分割する実装単位である。
- サブコンポーネントは親の責務境界を越えず、親の依存規則と禁止事項を継承する。
- ファイル肥大化、I/O技術、CLI framework、HTTP transport等だけを理由に橋名メインコンポーネントを新設してはならない。
- `products/cli`および`products/server`は製品境界であり、BRIDGEのメインコンポーネントではない。`contracts`と`internal`も中立契約・技術基盤であり、メインコンポーネントではない。
- 用語の正式定義は`docs/WORD_DEFINITION.md`の`main-component`および`subcomponent`に従う。

### 3.2 責務分離リファクタリングの不変条件

責務分離のみを目的とする変更では、ANCHOR、TRUSS、BOLTSのアルゴリズム、探索順序、Work計上、Budget Ledger、証明状態および終了判定を変更してはならない。

同一Scenario、Seed、Logical Worker、Work Budgetおよび実行設定について、変更前後で以下の決定的結果が一致しなければならない。

- Stable Digest
- PathとCost
- Search Completed、Reachability Proven、Optimality Proven
- Termination Reason
- Work MetricsとBudget Ledger
- Graph、QueryおよびAlgorithm Configuration

実時間、割当量、実行日時、Execution ID等の非決定的計測値は、同値性判定から明示的に除外できる。詳細は`docs/architecture/RESPONSIBILITY_REFACTORING_RULES.md`に従う。

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

型付きevent契約、安全なObserver境界、購読可否判定、観測時刻の付与および配送を所有する。探索制御、保存、集計、統計化、予算変更を行わない。

BEARINGはTRAFFIC、GATE、TRUSS、ANCHORおよびBOLTSとULTRASOUNDの間の唯一の観測中間層である。発行元とULTRASOUNDの型依存を遮断し、観測無効時のNo-op経路を提供する。

### 4.7 ULTRASOUND

BEARING eventの収集、開始・終了eventの対応付け、区間時間算出、保存、replay、Anytime曲線、state reuse、duplicate work、Work再構成および観測artifact生成を担当する。観測結果を探索順序、solver選択、予算配分または終了判定へ返してはならない。

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
TRAFFIC    → GATE公開API, CORE公開schema, BEARING, ULTRASOUND公開API
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

## 10. 観測アーキテクチャとTiming

### 10.1 基本原則

> 観測はアルゴリズムを説明するために存在し、アルゴリズムの振る舞いを決定してはならない。

観測は探索・調停・公開実行で発生した事実を外部から説明、検査および再構成するための補助機構である。観測結果または観測器の状態を、探索順序、候補選択、solver選択、Budget配分、証明昇格、終了判定その他のアルゴリズム判断へ入力してはならない。

### 10.2 正式な観測経路

TRAFFIC、GATE、TRUSS、ANCHORおよびBOLTSが観測情報を発行する場合、必ずBEARINGで定義された型付き意味eventを使用し、ULTRASOUNDへ直接依存してはならない。

```text
TRAFFIC / GATE / TRUSS / ANCHOR / BOLTS
                     │
                     ▼
                  BEARING
                     │
                     ▼
                ULTRASOUND
```

- 発行元は、実行上の事実と最小限の集約値だけを通知する。
- BEARINGはevent契約、購読可否判定、観測時刻の付与、Sequence管理および配送を担当する。
- ULTRASOUNDはeventの収集、対応付け、時間差算出、集計、保存、traceおよび観測artifact生成を担当する。
- BEARINGは集計器ではなく、ULTRASOUNDは探索制御器ではない。

### 10.3 HEALTHYの独立

HEALTHYはBEARINGおよびULTRASOUNDへ依存しない独立評価器とする。HEALTHYの正式な出力は、Path妥当性、Cost検証、証明整合性、品質評価および回帰判定に使用する評価結果であり、観測eventではない。

HEALTHYの実行時間または呼出し回数を観測する必要がある場合、HEALTHY自身ではなく、TRAFFICまたはBenchmark Runner等の呼出し側がHEALTHY呼出しの前後で低頻度の境界eventを発行する。

### 10.4 発行元の責務と禁止事項

TRAFFIC、GATE、TRUSS、ANCHORおよびBOLTSは、観測のために以下を行ってはならない。

- ULTRASOUND固有型への変換または直接呼出し
- 時間差、percentile、分散その他の統計集計
- JSON、trace file、manifest、ZIPその他のartifact生成
- 観測履歴または集計状態の保持
- 観測結果による探索状態または実行方針の変更
- 無効なeventのpayload、Map、Slice、closureまたはSpan識別子の先行生成

ANCHORおよびBOLTSはアルゴリズム意味論に必要なWork、Bound、Candidate、Proof、Termination等を所有できる。これらの時系列化、分布化、相関分析および保存はULTRASOUNDの責務である。

### 10.5 eventの意味と粒度

eventは計測実装上の番号ではなく、ドメインまたは実行境界として意味のある事象を表さなければならない。`timer_1`、`measurement_point_7`等の意味を持たないevent名を新設してはならない。

観測粒度は次の3段階に分離する。

| 種別 | 用途 | 例 | 有効化条件 |
|---|---|---|---|
| Lifecycle | 主要な実行境界と区間時間 | request、graph build、solve、handoff、result conversion | minimum以上 |
| Diagnostic | 低頻度の制御・診断 | epoch、fallback、candidate certification | diagnostic以上 |
| Trace | 高頻度の探索事象 | node expansion、edge evaluation、relaxation、queue operation | traceのみ |

高頻度eventをminimumまたは通常の性能回帰経路で生成してはならない。Lifecycle eventも1 Route当たり定数件を原則とし、WorkまたはGraph規模に比例して増加させてはならない。

### 10.6 観測無効時の性能非干渉

観測無効時は、観測に由来する処理を可能な限り実質ゼロコストにしなければならない。少なくとも以下を満たす。

- event payloadを生成しない
- `map[string]any`等の動的構造を生成しない
- 観測時刻を取得しない
- SequenceまたはSpan識別子を生成しない
- closure、interface boxingその他の不要Allocationを発生させない
- 高頻度ループ内で購読可否を毎回再計算せず、SessionまたはRoute境界で有効性を確定・キャッシュする

発行APIは、引数評価によって無効eventのpayloadが先に生成される形にしてはならない。購読判定はpayload生成より前に行う。

### 10.7 Observer非干渉と決定的同値性

ObserverまたはULTRASOUNDの有無、観測レベル、trace保存有無によって、以下を変更してはならない。

- Path、CostおよびPath Found
- Termination Statusと停止理由
- ProofClass、Reachability Proven、Optimality Proven
- Work、StepおよびBudget Ledger
- seed消費順と決定論性
- solver選択、Handoff判断およびBudget配分

同一入力・設定について、観測ON/OFFの結果同値性を自動テストで保証する。観測無効時のAllocation回帰も継続的に検査する。

### 10.8 Timing契約

主要区間の経過時間は、開始・終了Lifecycle eventへBEARINGが付与した単調時計由来の時刻をULTRASOUNDが対応付け、その差分から算出することを原則とする。発行元はDurationを集計または保持しない。

- BEARINGは購読が有効な場合に限り、配送処理より前に観測時刻を付与する。
- ULTRASOUNDはRun ID、Operationおよび必要なSpan識別情報により開始・終了eventを対応付ける。
- 欠落、重複、逆転した境界eventからDurationを捏造してはならず、無効Timingとして扱う。
- JSON等へ保存する壁時計時刻と、Duration算出に使用する単調時計由来値を混同してはならない。
- end-to-end時間はTRAFFICが公開API境界で直接測定できる。Benchmark Runner自身のJSON、I/O、圧縮等はRunnerの責務として別区間で測定する。
- TRAFFICによるGraph GenerationはBenchmark Setupとして独立観測し、`benchmark_run`、end-to-end時間、通常の性能回帰判定へ含めない。Graph Generation時間は再現性確認およびSetup性能の参考値としてのみ保持する。
- 既存のsolver自己計測を維持する場合も、全終了経路で記録し、`timing_valid=true`かつ`solver_ns=0`を禁止する。
- 表示用msはnsから導出し、無効Timingを性能比較へ使用しない。

### 10.9 Warm-up

Warm-upは本計測から除外する。Warm-upではObservation、Collector、Trace保存を無効化し、破棄される高頻度eventを生成しない。

## 11. BRIDGE実行境界

BRIDGEの処理と計測は、実際に外部システムまたはアプリケーションへ組み込んだ場合の責務境界に基づき、`Preparing`、`Running`、`Finalizing`のPhaseへ分類する。Phaseは単なる進捗状態ではなく、処理責務、計測境界および性能指標の意味を定める分類である。

### 11.1 Preparing

BRIDGEへ渡す入力を外部で準備するPhase。Scenario解釈、Graph生成、Query生成および実行条件確定を含む。地図アプリケーション、外部scriptまたはTRAFFICによるGraph生成はBRIDGE外部の責務であり、BRIDGE実行時間へ含めない。

### 11.2 Running

BRIDGEが公開interfaceで入力を受理してから、外部利用可能な結果を返却するまでのPhase。入力検証、外部Graphの内部表現化、Graph分析と特徴抽出、実行方針決定、経路探索、結果構築、および要求されたTraceの取得・構築・返却を含む。

内部Graph構築、Graph分析および特徴抽出はRouteまたはSolver時間とは分離して記録できるが、BRIDGEの責務であるためRunningから除外してはならない。

### 11.3 Finalizing

BRIDGE返却後の結果を外部で集計、分析、評価、保存または可視化するPhase。複数Run統計、HEALTHY評価、性能回帰判定、Artifact生成、および外部SimulatorによるTrace可視化を含む。

### 11.4 境界規則

- Phaseは実装コンポーネントではなく、実運用上の責務と公開interface境界によって決定する。
- Graph生成時間をBRIDGE実行時間として扱ってはならない。
- 外部Graphの内部表現化時間をBRIDGE実行時間から除外してはならない。
- Trace取得・構築・返却はRunning、Traceを用いた描画・可視化はFinalizingに属する。
- 経路探索性能、BRIDGE全体性能、Benchmark全体性能を独立して記録し、混同してはならない。
- Benchmark実装上の関数境界ではなく、BRIDGEの公開interface境界を計測の正本とする。

## 12. Benchmark不変条件

TRAFFICは以下を検出した場合、集計を継続せずRunを失敗させる。

- Pathなしで`optimality_proven=true`
- Search未完了で`optimality_proven=true`
- Pathありで`reachability_proven=false`
- Budget不足なのに到達不能または最適性を証明済み
- `NO_PATH`とPathありの併存
- `timing_valid=true`かつsolver時間0
- Path距離と再計算値の不一致

性能値と意味論の正しさを分離する。正しいが遅い実装は性能課題であり、虚偽の証明または無効Timingは研究データを破壊する致命的欠陥である。

## 13. テストとリリースゲート

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

## 14. 破壊的変更と旧仕様削除

v0.15.0は旧内部仕様との互換を要件としない。互換分岐、deprecated adapter、未使用fallback、dead code、固定値telemetryを残すことより、現行契約を単純かつ一意に保つことを優先する。

公開契約を変更した場合は、互換コードではなく移行ガイドで扱う。
