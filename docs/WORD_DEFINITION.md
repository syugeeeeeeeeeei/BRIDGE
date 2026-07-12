# BRIDGE 用語定義集

**文書ID:** BRIDGE-WORD-DEFINITION-002  
**版:** v2.0-draft  
**状態:** 正本候補  
**対象:** Go実装、GATE、TRUSS、ANCHOR、BOLTS、TRAFFIC、ULTRASOUND、HEALTHY、CLI、SDK、Scenario、Trace、Result Artifact、設計文書

---

## 1. 目的

本書は、BRIDGEで使用する用語の唯一の正本である。

コード、JSON Schema、CLI、SDK、Scenario、Trace、Result Artifact、テスト、設計文書は、本書と異なる意味で同じ用語を使用してはならない。

本書は、用語を知識領域ごとに見出しで整理するが、定義ファイル自体は分割しない。同一概念を複数箇所で再定義せず、一つの正式定義を他の領域から参照する。

---

## 2. 用語管理規則

### 2.1 正式用語

正式用語には、可能な限り一般的なアルゴリズム、グラフ理論、統計、性能評価の用語を使用する。

BRIDGE固有の名称は、コンポーネント名またはBRIDGE固有の責務を表す場合に限る。一般概念を橋梁由来の比喩へ置き換えてはならない。

### 2.2 用語項目の構成

主要用語は、必要に応じて次を記載する。

- **Term ID:** 文書とコードから参照する安定ID
- **分類:** 用語が属する知識領域
- **定義:** BRIDGEにおける正式な意味
- **意味しないもの:** 混同してはならない概念
- **推奨フィールド:** 公開JSONで推奨するフィールド名
- **旧フィールド:** 非推奨または互換目的で残る名称
- **所有・算出:** 値を決定するコンポーネント

### 2.3 フィールド名との関係

用語とJSONフィールドは同一ではない。

- 用語は概念の定義である。
- フィールドは、特定の契約でその概念を表現する名前である。
- 同じ用語を複数の契約で使用する場合も、意味を変更してはならない。
- 同じフィールド名を異なる意味で使用してはならない。

コードおよびSchemaでは、可能な限り次の形式で用語を参照する。

````text
Term: optimality-proven
See: docs/WORD_DEFINITION.md#optimality-proven
````

### 2.4 曖昧な無修飾語

公開契約では、対象を特定できない次の無修飾語を原則として使用しない。

- `mode`
- `strategy`
- `time`
- `average_time`
- `target`
- `result`
- `status`
- `count`
- `ratio`
- `exact`

既存互換性のために残す場合は、非推奨であることと移行先を明記する。

### 2.5 定義変更

用語の定義変更は、単なる文書修正として扱わない。以下への影響を確認する。

- Go型およびJSONタグ
- JSON Schema
- CLIおよびSDK
- Scenario
- Raw RunおよびSummary
- TraceおよびReplay
- Acceptance
- HEALTHY検証
- 保存済みArtifactとの互換性

意味を破壊的に変更する場合は、Schema Versionまたは契約Versionを更新する。

### 2.6 未計測、対象外、ゼロ

数値の`0`を、未計測または対象外の代用として使用してはならない。

| 状態 | 意味 |
|---|---|
| Measured Zero | 計測を行い、結果が0だった |
| Not Measured | 計測を実施していない |
| Not Applicable | 当該Runには概念が適用されない |
| Unavailable | 本来必要だが取得できなかった |

公開Resultではnullable値、availability field、または明示的な状態値により区別する。

---

## 3. 基本的な経路探索用語

### 3.1 Graph

- **Term ID:** `graph`
- **分類:** グラフ

NodeとEdgeから構成され、経路探索の対象となる構造。

### 3.2 Node

- **Term ID:** `node`
- **分類:** グラフ
- **推奨フィールド:** `source_node`, `target_node`, `node_count`

Graph上の頂点。単独の`source`または`target`より、公開契約では`source_node`、`target_node`を推奨する。

### 3.3 Edge

- **Term ID:** `edge`
- **分類:** グラフ

二つのNode間の接続。方向、Weight、その他の制約を持ち得る。

### 3.4 Edge Weight

- **Term ID:** `edge-weight`
- **分類:** グラフ・品質

Edgeを通過するCost。物理的な距離とは限らない。

### 3.5 Cost

- **Term ID:** `cost`
- **分類:** 品質
- **推奨フィールド:** `path_cost`, `optimal_cost`, `lower_bound_cost`

目的関数によって算出される経路の評価値。BRIDGEの標準経路探索では、Edge Weightの総和を指す。

`distance`は互換上使用される場合があるが、物理距離を意味しない場合は`cost`を推奨する。

### 3.6 Path

- **Term ID:** `path`
- **分類:** 経路
- **推奨フィールド:** `path`

Source NodeからTarget Nodeまで連続するEdge列またはNode列。すべての接続がGraphに存在し、方向および制約を満たす場合のみ有効である。

### 3.7 Path Found

- **Term ID:** `path-found`
- **分類:** 経路・結果
- **推奨フィールド:** `path_found`
- **旧フィールド:** `found`

有効なSource–Target Pathが返却結果に存在する状態。

#### 意味しないもの

- 最適なPathであること
- 探索を完了したこと
- 到達不能を証明したこと
- 品質境界を証明したこと

### 3.8 Candidate Path

- **Term ID:** `candidate-path`
- **分類:** 経路・探索

最終採用前の完全経路候補。部分経路は`Partial Path`と呼び、完全なSource–Target Pathと混同しない。

### 3.9 Incumbent Path

- **Term ID:** `incumbent-path`
- **分類:** Anytime・探索

現時点で既知の有効なCandidate Pathのうち、目的Costが最小のPath。

旧文書の`Best Path`は、この概念を指す場合に限り使用できる。公開契約では`incumbent_path`または`best_path_found`を推奨する。

### 3.10 First Path

- **Term ID:** `first-path`
- **分類:** Anytime・探索

実行中に最初に確定した有効なSource–Target Path。

---

## 4. 探索状態、完了、到達可能性、最適性

### 4.1 Search Completed

- **Term ID:** `search-completed`
- **分類:** 実行状態
- **推奨フィールド:** `search_completed`
- **所有・算出:** 各SolverまたはTRUSS

Solverが、そのアルゴリズムで定義された通常終了条件まで探索を実行した状態。

#### 意味しないもの

- Pathが見つかったこと
- Pathが最適であること
- 到達可能性を証明したこと
- 全Graphを探索したこと

Budget Exhaustion、Deadline Exceeded、Cancellationなどにより中断された場合は、通常`false`となる。

### 4.2 Reachable

- **Term ID:** `reachable`
- **分類:** 到達可能性
- **推奨フィールド:** `reachable`

Source NodeからTarget Nodeへ到達可能であるという判定値。

到達可能性が未確定の場合、単純なbooleanではなく、`unknown`を表現できる状態型を使用する。

### 4.3 Reachability Proven

- **Term ID:** `reachability-proven`
- **分類:** 到達可能性・証明
- **推奨フィールド:** `reachability_proven`

Source NodeからTarget Nodeが到達可能または到達不能であることを、完全な探索または同等の証明によって確定した状態。

#### 意味しないもの

- 返却Pathが最短であること
- Weighted Graph上の最小Costを求めたこと
- 品質比を証明したこと

### 4.4 Optimality Proven

<a id="optimality-proven"></a>

- **Term ID:** `optimality-proven`
- **分類:** 品質・証明
- **推奨フィールド:** `optimality_proven`
- **旧用語・旧フィールド:** `Exact`, `exact`

返却された解が、指定された目的関数と制約に対する最適解であることを、アルゴリズムの性質または独立した証明によって確定した状態。

#### 設定条件

以下のいずれかが成立する場合に限り`true`とする。

- Solverの正当性と終了条件により最適性が保証される
- 独立したExact SolverまたはBound証明により最適性が確定する

#### 意味しないもの

- Search Completedであること
- Reachability Provenであること
- Exact Referenceと偶然同じCostになったこと
- 名前に`exact`を含むSolverを使用したこと
- 近似比が小さいこと

### 4.5 Exact

- **Term ID:** `exact-deprecated`
- **分類:** 非推奨
- **状態:** Deprecated

旧契約で最適性、探索完了、到達可能性の証明など複数の意味に使用されていた曖昧な語。

新しい公開契約では使用せず、次へ分離する。

- `search_completed`
- `reachability_proven`
- `optimality_proven`
- `matches_exact_reference`

文書中で`Exact Solver`または`Exact Reference`の限定語として使用することはできる。

### 4.6 Matches Exact Reference

- **Term ID:** `matches-exact-reference`
- **分類:** ベンチマーク評価
- **推奨フィールド:** `matches_exact_reference`
- **所有・算出:** TRAFFICまたはHEALTHY

評価対象の結果が、同一Graph Instance、Query、目的関数、制約条件に対するExact Referenceと一致した状態。

これは評価後の比較結果であり、Solver自身のOptimality Provenとは異なる。

### 4.7 Quality Bound Proven

- **Term ID:** `quality-bound-proven`
- **分類:** 品質・証明
- **推奨フィールド:** `quality_bound_proven`
- **旧用語・旧フィールド:** `Quality Certified`, `quality_certified`

返却PathのCostが、最適Costに対して指定された比率以内であることをUpper BoundとLower Bound、またはExact Solverにより証明した状態。

### 4.8 Proven Cost Ratio

- **Term ID:** `proven-cost-ratio`
- **分類:** 品質・証明
- **推奨フィールド:** `proven_cost_ratio`
- **旧用語・旧フィールド:** `Certified Ratio`, `certified_ratio`

`Upper Bound / Lower Bound`で求める、証明済みのCost比。

- 値は原則として1以上である。
- 1.0はOptimality Provenを意味する。
- Lower Boundが未確定または0の場合は算出しない。

### 4.9 Upper Bound

- **Term ID:** `upper-bound`
- **分類:** 品質・証明
- **推奨フィールド:** `upper_bound_cost`

既知の有効Path Cost。最適Cost以上である。

### 4.10 Lower Bound

- **Term ID:** `lower-bound`
- **分類:** 品質・証明
- **推奨フィールド:** `lower_bound_cost`

最適Costを超えないことが保証された値。

### 4.11 Maximum Allowed Cost Ratio

- **Term ID:** `maximum-allowed-cost-ratio`
- **分類:** 品質制約
- **推奨フィールド:** `maximum_allowed_cost_ratio`
- **旧用語・旧フィールド:** `Max Suboptimality`, `max_suboptimality`

利用者が許容する、最適Costに対する最大比率。

例として1.05は、最適Costの1.05倍以下を許容する。

`suboptimality`は分野により差分または比率の双方を指し得るため、公開契約では`cost_ratio`を明示する。

---

## 5. Work、Step、並列性

### 5.1 Work

- **Term ID:** `work`
- **分類:** 探索量
- **推奨フィールド:** `work_actions`, `work_breakdown`
- **旧フィールド:** `total_actions`, `average_work`

経路探索中に実行された、意味を持つ探索Actionの総数。CPU命令数、関数呼出し数、Node訪問数、Edge走査数、経過時間ではない。

標準Actionは次のとおり。

| Action | 定義 |
|---|---|
| `SELECT` | Frontier等から次の探索対象を選択する |
| `EXPAND` | 選択した探索状態を展開する |
| `EVALUATE` | Node、Edge、候補、制約を評価する |
| `RELAX` | DistanceまたはCostの更新を試行する |
| `ENQUEUE` | 探索候補をFrontierへ追加する |
| `REJECT` | Stale候補、非改善候補、制約違反候補または探索枝を棄却する |
| `BACKTRACK` | 親状態または分岐点へ戻る |
| `CONNECT` | 複数の探索方向または部分経路を接続する |
| `CANDIDATE` | 完全経路候補を生成する |
| `REPAIR` | 既存経路の区間を置換または改善する |
| `BOUND` | Upper BoundまたはLower Boundを有効に更新する |
| `TERMINATE` | 探索LaneまたはTaskの終了判断を確定する |

次はWorkへ含めない。

- Telemetry生成
- JSONまたはCSV出力
- Traceのシリアライズ
- GC
- Mutex待機
- ファイル読込み
- SDK変換
- 外部ID変換

### 5.2 Reported Work

- **Term ID:** `reported-work`
- **分類:** Work Accounting

SolverまたはTRUSSがResultに記録したWork。

### 5.3 Reconstructed Work

- **Term ID:** `reconstructed-work`
- **分類:** Work Accounting

Trace、Action内訳、Budget Ledger等からHEALTHYが再構成したWork。

### 5.4 Work Accounting

- **Term ID:** `work-accounting`
- **分類:** Work Accounting

WorkをAction、Component、SolverTask、Portfolio、Runの各単位で計上し、保存する仕組み。

### 5.5 Work Conservation

- **Term ID:** `work-conservation`
- **分類:** Work Accounting

親単位のWorkが、子単位のWorkと明示的なOrchestration Workの合計に一致する性質。

### 5.6 Work Mismatch

- **Term ID:** `work-mismatch`
- **分類:** Work Accounting・検証

Reported WorkとReconstructed Work、または親子Ledgerの合計が許容差を超えて不一致となった状態。

### 5.7 Step

- **Term ID:** `step`
- **分類:** 並列性

依存関係を保ちながら同時実行可能な複数Workを一つにまとめた論理実行段階。

単独の`steps`を公開せず、次の限定語を使用する。

### 5.8 Logical Step

- **Term ID:** `logical-step`
- **分類:** 並列性
- **推奨フィールド:** `logical_steps`

無制限の並列資源を仮定した場合の最小Step数。

### 5.9 Scheduled Step

- **Term ID:** `scheduled-step`
- **分類:** 並列性
- **推奨フィールド:** `scheduled_steps`

指定された論理Worker数へWorkを配置した場合のStep数。

常に次を満たす。

````text
Logical Steps <= Scheduled Steps <= Work Actions
````

### 5.10 Parallel Steps

- **Term ID:** `parallel-steps-deprecated`
- **分類:** 非推奨・互換
- **旧フィールド:** `parallel_steps`

Logical StepsまたはScheduled Stepsのどちらを意味するか不明確な旧互換Counter。新しい公開契約では使用しない。

### 5.11 Logical Worker

- **Term ID:** `logical-worker`
- **分類:** 並列性
- **推奨フィールド:** `logical_worker_count`
- **旧フィールド:** `workers`, `worker_count`

同時実行可能性の計算または探索構造上の割当てに使用する論理実行単位。

#### 意味しないもの

- OS Thread数
- Goroutine数
- Process数
- CPU Core数
- 実際に並列実行された数

実並列実装を評価する場合は、別に`execution_thread_count`または`process_count`を記録する。

### 5.12 Lane

- **Term ID:** `lane`
- **分類:** 探索構造

独立または半独立に進行できる探索系列。例としてForward Lane、Backward Lane、Hypothesis Laneがある。

LaneはWorkerと同義ではない。

### 5.13 Available Parallelism

- **Term ID:** `available-parallelism`
- **分類:** 並列性
- **推奨フィールド:** `available_parallelism`

`Work Actions / Logical Steps`で表す、探索依存関係上利用可能な平均並列度。

実測CPU利用率ではない。

---

## 6. 予算、期限、中断

### 6.1 Work Budget

- **Term ID:** `work-budget`
- **分類:** 予算
- **推奨フィールド:** `work_budget`

TRUSSが所有する、Portfolio全体で消費可能なWork上限。

### 6.2 Step Budget

- **Term ID:** `step-budget`
- **分類:** 予算

許容されるLogical StepsまたはScheduled Stepsの上限。対象とするStep種別を必ず明記する。

### 6.3 Memory Budget

- **Term ID:** `memory-budget`
- **分類:** 予算
- **推奨フィールド:** `memory_budget_kib`

探索に許容するMemory上限。Heap Allocated、Resident Set Size等、対象指標を契約で明示する。

### 6.4 Budget Slice

- **Term ID:** `budget-slice`
- **分類:** 予算・TRUSS

TRUSSが一つのSolver Taskへ割り当てるWork、Deadline、Logical Worker、Memory等の部分予算。

### 6.5 Budget Ledger

- **Term ID:** `budget-ledger`
- **分類:** 予算・監査
- **推奨フィールド:** `budget_ledger`

Portfolio内の予算付与、使用、返却、再配分を記録する監査可能な台帳。

### 6.6 Budget Exhausted

- **Term ID:** `budget-exhausted`
- **分類:** 終了理由
- **推奨フィールド:** `budget_exhausted`

割り当てられた予算を使い切ったため、通常終了条件より前に探索を停止した状態。

### 6.7 Deadline

- **Term ID:** `deadline`
- **分類:** 時間制約
- **推奨フィールド:** `deadline_ms`

経過時間に基づく実行終了時点。Work Budgetとは独立する。

### 6.8 Deadline Exceeded

- **Term ID:** `deadline-exceeded`
- **分類:** 終了理由
- **推奨フィールド:** `deadline_exceeded`

Deadlineを超過したため探索を停止した状態。

### 6.9 Timeout

- **Term ID:** `timeout`
- **分類:** 実行制約

外部または実行基盤が設定する最大許容時間。Deadlineと同義として扱う場合は、契約上どちらか一つへ統一する。

### 6.10 Cancellation

- **Term ID:** `cancellation`
- **分類:** 終了理由

外部要求またはTRUSS判断により実行を中断すること。Deadline ExceededおよびBudget Exhaustedとは区別する。

---

## 7. 探索アルゴリズムと制御

### 7.1 Anytime

- **Term ID:** `anytime`
- **分類:** アルゴリズム性質

少ない予算でも有効解を返し、追加予算に応じて解品質または品質証明を改善できる性質。

### 7.2 Frontier

- **Term ID:** `frontier`
- **分類:** 探索構造

今後処理される探索状態の集合。Priority Queue、Beam、Bucket等で表現される。

### 7.3 Hypothesis

- **Term ID:** `hypothesis`
- **分類:** ANCHOR

ANCHORが試行する探索方針。Hypothesisの生成はFallbackではない。

### 7.4 Corridor

- **Term ID:** `corridor`
- **分類:** ANCHOR

Source–Target間の幾何学的または構造的仮説に基づいて探索対象を制限した部分領域。

### 7.5 Portal

- **Term ID:** `portal`
- **分類:** ANCHOR

離れた領域間を接続する可能性が高いNodeまたはEdge Endpoint。

### 7.6 Hub

- **Term ID:** `hub`
- **分類:** ANCHOR

Degree、Centralityまたは接続性が周囲より高く、経路接続点として利用価値が高いNode。

### 7.7 Detour

- **Term ID:** `detour`
- **分類:** 探索処理

局所障害、高Cost区間、Barrier等を回避する部分経路またはその探索処理。

### 7.8 Repair

- **Term ID:** `repair`
- **分類:** 探索処理

既存Pathの一部を再探索し、より低Cost、より確実、または制約適合したSubpathへ置換する処理。

### 7.9 Fallback

- **Term ID:** `fallback`
- **分類:** TRUSS・BOLTS

ANCHORによる候補生成または通常経路が必要条件を満たさない場合に、TRUSSが別のBOLTS Solverへ明示的に切り替える処理。

ANCHOR内部の別Hypothesisへの切替えはFallbackに含めない。

### 7.10 Recovery

- **Term ID:** `recovery`
- **分類:** TRUSS・BOLTS

失敗、中断、到達不能の疑い、品質不足等の後に、有効な結果または確定的な判定を得るため追加Solverを実行する処理。

FallbackはRecoveryを開始する一方式である。

### 7.11 Certification

- **Term ID:** `certification`
- **分類:** 品質・TRUSS・BOLTS

Exact SolverまたはLower Bound Providerにより、Optimality ProvenまたはQuality Bound Provenを得る処理。

### 7.12 Reachability Solver

- **Term ID:** `reachability-solver`
- **分類:** BOLTS

Source–Targetの到達可能性を判定するSolver。Weighted Shortest Path Solverではない。

### 7.13 Stagnation

- **Term ID:** `stagnation`
- **分類:** 探索状態

一定のWorkまたはStepの間、Upper Bound、Lower Bound、Candidate生成、Frontier進展に有意な改善がない状態。

### 7.14 State Reuse

- **Term ID:** `state-reuse`
- **分類:** 探索最適化

一つのSolverまたはTaskで生成した有効な探索状態を、後続のSolverまたはTaskで再利用すること。

### 7.15 Budget Reallocation

- **Term ID:** `budget-reallocation`
- **分類:** TRUSS・予算

未使用または返却されたBudget Sliceを、別のSolver Taskへ再配分する処理。

---

## 8. 実行単位

### 8.1 Route Request

- **Term ID:** `route-request`
- **分類:** GATE・実行

一回の経路探索要求。Graph、Source Node、Target Node、品質制約、予算、Route Mode等を含む。

### 8.2 Route Result

- **Term ID:** `route-result`
- **分類:** GATE・結果

一回のRoute Requestに対する結果。Path、Cost、状態、証明、Work、Time Breakdown等を含む。

### 8.3 Portfolio

- **Term ID:** `portfolio`
- **分類:** TRUSS

一つのRoute Requestを処理するためにTRUSSが運用するANCHORおよびBOLTSのSolver Task全体。

### 8.4 Solver Task

- **Term ID:** `solver-task`
- **分類:** TRUSS
- **推奨フィールド:** `task_id`, `purpose`

TRUSSが作成する一つの探索実行単位。Purpose、Budget Slice、Deadline、Logical Worker、品質目標を持つ。

### 8.5 Session

- **Term ID:** `session`
- **分類:** 実行

中断、再開、Cancellation、Snapshotが可能なSolver実行状態。

### 8.6 Probe

- **Term ID:** `probe`
- **分類:** TRUSS

少量予算でGraphまたはQueryとの相性、進展率、到達性等を評価する短いSolver Task。

### 8.7 Route Mode

- **Term ID:** `route-mode`
- **分類:** 公開設定
- **推奨フィールド:** `route_mode`
- **旧構造:** `route.mode`

BRIDGEの経路探索方針を指定する設定。

Route Modeの値は、実装上の動作差が定義されている場合にのみ公開する。名称だけ異なり実装動作が同一の場合は、別Modeとして扱わない。

### 8.8 Execution Target

- **Term ID:** `execution-target`
- **分類:** TRAFFIC

TRAFFICが一つのRunで評価対象として指定する公開実行対象。BRIDGE全体、ANCHOR単体、BOLTS Solver等を含む。

Execution TargetはBaselineであることを自動的には意味しない。

### 8.9 Algorithm ID

- **Term ID:** `algorithm-id`
- **分類:** 公開契約
- **推奨フィールド:** `algorithm_id`
- **旧フィールド:** `solver_name`, `algorithm`

公開実行対象となるアルゴリズムまたはSolver実装を識別する安定ID。

表示名と分離し、名称変更によってIDを変更しない。

### 8.10 Execution Path

- **Term ID:** `execution-path`
- **分類:** TRAFFIC・監査

RequestがGATE、TRUSS、ANCHOR、BOLTS等のどの経路を通って実行されたかを示す識別情報。

---

## 9. Scenarioとグラフ生成

### 9.1 Benchmark Suite

- **Term ID:** `benchmark-suite`
- **分類:** TRAFFIC

共通の目的、実行条件、比較対象、保存方針を持つScenario群。

### 9.2 Scenario

- **Term ID:** `scenario`
- **分類:** TRAFFIC

Graph条件、Query条件、Route条件、Budget条件、Ablation条件をまとめた一つの実験条件。

Suite全体の反復回数、並列Job数、出力先等はExecution ManifestまたはOutput Configurationに属し、Scenarioの意味には含めない。

### 9.3 Execution Manifest

- **Term ID:** `execution-manifest`
- **分類:** TRAFFIC

Benchmark Suiteの実行条件を定義する構造。Seed、Repetition、Warm-up、Job、Timeout、実行順序等を含む。

### 9.4 Query

- **Term ID:** `query`
- **分類:** TRAFFIC・グラフ

一つのGraph Instance上で実行するSource Node、Target Nodeおよび選択方法を表す経路探索条件。

### 9.5 Query ID

- **Term ID:** `query-id`
- **分類:** TRAFFIC
- **推奨フィールド:** `query_id`

Queryを識別する安定ID。Query IDは識別子であり、Queryの選択方法や性質を保証しない。

`diameter`、`cross-community`等の性質をIDだけで表現してはならない。

### 9.6 Query Selection Method

- **Term ID:** `query-selection-method`
- **分類:** TRAFFIC・グラフ
- **推奨フィールド:** `query_selection_method`
- **旧フィールド:** `strategy`

Source NodeとTarget Nodeを決定する方法。

推奨値の例:

| 値 | 意味 |
|---|---|
| `generator_default_endpoints` | Generatorが返す既定端点を使用する |
| `explicit_endpoints` | Scenarioで明示した端点を使用する |
| `graph_diameter_endpoints` | Graph Diameterを形成する端点を算出して使用する |
| `cross_community_endpoints` | 異なるCommunityから端点を選択する |
| `random_reachable_endpoints` | 到達可能性を確認したランダム端点を使用する |

実装されていない方法を値として公開してはならない。

### 9.7 Generator Default Endpoints

- **Term ID:** `generator-default-endpoints`
- **分類:** グラフ・Query
- **推奨値:** `generator_default_endpoints`
- **旧値:** `opposite-corners`

Graph GeneratorがGraphとともに返す既定のSource NodeとTarget Node。

全Graph Familyで幾何学的な対角点を意味するわけではないため、`opposite-corners`という名称は使用しない。

### 9.8 Graph Generator

- **Term ID:** `graph-generator`
- **分類:** グラフ
- **推奨フィールド:** `graph_generator`
- **旧フィールド:** `generator`

SeedおよびGenerator ConfigurationからGraph Instanceを生成する実装。

### 9.9 Graph Family

- **Term ID:** `graph-family`
- **分類:** グラフ・研究

構造的性質を共有するGraph Instance群。Grid、Random Geometric、Community、Maze、Adversarial等を含む。

### 9.10 Topology

- **Term ID:** `topology`
- **分類:** グラフ

NodeとEdgeの接続構造。Grid Generatorにおける`open`、`wall`、`u_shape`等はTopology Variantである。

### 9.11 Requested Node Count

- **Term ID:** `requested-node-count`
- **分類:** Graph Configuration
- **推奨フィールド:** `requested_node_count`
- **旧フィールド:** `nodes`

Graph Generatorへ要求するNode数。実際の生成数と一致しない可能性がある。

### 9.12 Actual Node Count

- **Term ID:** `actual-node-count`
- **分類:** Graph Metadata
- **推奨フィールド:** `actual_node_count`

生成されたGraphに含まれる実際のNode数。

### 9.13 Traversable Node Count

- **Term ID:** `traversable-node-count`
- **分類:** Graph Metadata
- **推奨フィールド:** `traversable_node_count`

探索可能なNode数。BlockされたNodeをGraph上に保持する実装では、Actual Node Countと異なり得る。

### 9.14 Edge Weight Noise

- **Term ID:** `edge-weight-noise`
- **分類:** Graph Configuration
- **推奨フィールド:** `edge_weight_noise`
- **旧フィールド:** `noise`

Edge Weightに加える確率的変動の設定。

分布、加算方式、範囲または標準偏差をGenerator契約で明示する。単なる`noise`は意味が広すぎるため使用しない。

### 9.15 Neighbor Candidate Count

- **Term ID:** `neighbor-candidate-count`
- **分類:** Graph Configuration
- **推奨フィールド:** `neighbor_candidate_count`
- **旧フィールド:** `k`

Random Geometric等のGeneratorで、各Nodeが接続候補として評価する近傍数。

最終Degreeを保証しない場合、`nearest_neighbor_count`ではなくこの名称を使用する。

### 9.16 Community Count

- **Term ID:** `community-count`
- **分類:** Graph Configuration
- **推奨フィールド:** `community_count`
- **旧フィールド:** `communities`

Community Graphで生成するCommunity数。

### 9.17 Graph Instance

- **Term ID:** `graph-instance`
- **分類:** TRAFFIC・グラフ

Graph Generator、Configuration、SeedまたはDatasetから確定した一つの具体的Graph。

### 9.18 Graph Instance ID

- **Term ID:** `graph-instance-id`
- **分類:** TRAFFIC・再現性
- **推奨フィールド:** `graph_instance_id`

Graph Instanceを識別する安定ID。同じIDは、同じGraph構造とWeightを表さなければならない。

### 9.19 Graph Digest

- **Term ID:** `graph-digest`
- **分類:** 再現性
- **推奨フィールド:** `graph_digest`

Node、Edge、方向、Weight等を正規化して算出するGraph専用Digest。

Run全体のStable Digestと混同しない。

### 9.20 Dataset

- **Term ID:** `dataset`
- **分類:** グラフ・研究

TRAFFICが研究・検証時に読み込む外部Graph Artifact。GATEのProduction Route入力とは別契約である。

### 9.21 Dataset Provenance

- **Term ID:** `dataset-provenance`
- **分類:** 再現性・研究

Datasetの出典、License、Version、取得日時、Checksum、前処理履歴を示す情報。

---

## 10. Benchmark実行と統計

### 10.1 Seed

- **Term ID:** `seed`
- **分類:** 再現性

Graph生成、Query選択、実行順序等の決定論的乱数系列を初期化する値。

Seedがどの処理へ適用されるかをExecution Manifestで明示する。

### 10.2 Repetition

- **Term ID:** `repetition`
- **分類:** 統計

同一Graph Instance、Query、Execution Target、Configurationを繰り返し実行する一回の本計測。

Graph個体差を増やすSeedとは異なる。

### 10.3 Warm-up

- **Term ID:** `warm-up`
- **分類:** 性能計測

本計測前に実行し、通常のSummary Statisticsから除外する事前実行。

### 10.4 Job

- **Term ID:** `job`
- **分類:** Benchmark実行
- **推奨フィールド:** `job_count`
- **旧フィールド:** `jobs`

TRAFFICが同時に実行できる独立Runの数。Solver内部のLogical Workerとは異なる。

### 10.5 Run

- **Term ID:** `run`
- **分類:** TRAFFIC

一つのScenario、Graph Instance、Query、Execution Target、Seed、Repetitionの組合せに対する一回の実行。

### 10.6 Run ID

- **Term ID:** `run-id`
- **分類:** TRAFFIC・再現性
- **推奨フィールド:** `run_id`

一つのRunを一意に識別する安定ID。

### 10.7 Statistical Unit

- **Term ID:** `statistical-unit`
- **分類:** 統計

統計分析で独立な観測単位として扱う単位。例として`scenario-algorithm-seed-query-repetition`がある。

### 10.8 Raw Run

- **Term ID:** `raw-run`
- **分類:** TRAFFIC・Artifact

一回のRunについて保存する、未集約の結果記録。

### 10.9 Summary Statistics

- **Term ID:** `summary-statistics`
- **分類:** 統計

複数Raw Runから算出する統計量。Mean、Median、Standard Deviation、Percentile、Confidence Interval等を含む。

### 10.10 Mean

- **Term ID:** `mean`
- **分類:** 統計
- **推奨フィールド接頭辞:** `mean_`
- **旧接頭辞:** `average_`

算術平均。`average`はMean、Median等を含む曖昧な語として解釈され得るため、公開統計では`mean`を推奨する。

### 10.11 Median

- **Term ID:** `median`
- **分類:** 統計

値を順序付けたときの中央の値。

### 10.12 Percentile

- **Term ID:** `percentile`
- **分類:** 統計

分布の指定割合以下に値が含まれる境界。例として`p95_solver_time_ms`がある。

### 10.13 Effect Size

- **Term ID:** `effect-size`
- **分類:** 統計

二つの条件間の差の大きさを表す統計量。p値とは異なる。

### 10.14 Paired Comparison

- **Term ID:** `paired-comparison`
- **分類:** 統計

同一Graph Instance、Query、Seed等の対応するRun同士を比較する方法。

### 10.15 Acceptance

- **Term ID:** `acceptance`
- **分類:** TRAFFIC・評価

ScenarioまたはSuiteに宣言された合否条件を、Raw RunまたはSummary Statisticsへ適用した評価結果。

Solver自身のPath Found、Search Completed、Optimality Proven、Quality Bound Provenとは区別する。

### 10.16 Ablation

- **Term ID:** `ablation`
- **分類:** 研究評価

BRIDGEの機構単位の寄与を評価するため、同一Scenario系列で一つ以上の機構を明示的に無効化する実験設定。

Production既定動作を変更せず、Scenarioから型付きOptionとして注入する。

### 10.17 Baseline

- **Term ID:** `baseline`
- **分類:** 研究評価

評価対象を比較するために選択された参照結果、参照アルゴリズム、参照Versionまたは参照実装。

Baselineは評価上の役割名であり、特定Solverの固有属性ではない。

無修飾の`baseline`ではなく、次の限定語を使用する。

### 10.18 Exact Reference

- **Term ID:** `exact-reference`
- **分類:** 研究評価
- **推奨フィールド:** `exact_reference_cost`, `exact_reference_reachable`

評価対象と同一のGraph Instance、Query、目的関数、制約条件に対する最適Costまたは到達不能という正解。

旧用語`Exact Baseline`より、参照値であることが明確な`Exact Reference`を推奨する。

### 10.19 Performance Baseline

- **Term ID:** `performance-baseline`
- **分類:** 研究評価

時間、Work、Step、Memory、初期解品質、改善曲線等を比較する参照アルゴリズムまたは構成。最適解を返す必要はない。

### 10.20 Regression Baseline

- **Term ID:** `regression-baseline`
- **分類:** 回帰評価

変更による退行を判定するための過去Version、Commit、Configurationまたは保存済みBenchmark Result。

### 10.21 Reference Implementation

- **Term ID:** `reference-implementation`
- **分類:** 移植・検証

別言語実装、移植元実装、または独立実装との意味的整合性を確認するための参照実装。性能上の優劣を示す語ではない。

### 10.22 Cost Ratio to Exact Reference

- **Term ID:** `cost-ratio-to-exact-reference`
- **分類:** 研究評価
- **推奨フィールド:** `cost_ratio_to_exact_reference`
- **旧用語・旧フィールド:** `Distance Ratio`, `distance_ratio`

`Result Cost / Exact Reference Cost`で算出する評価値。

Solver内部の証明値ではなく、TRAFFICまたはHEALTHYが評価後に算出する。

---

## 11. 時間とSystem Metrics

### 11.1 Time Breakdown

- **Term ID:** `time-breakdown`
- **分類:** 性能計測

一つのRunの時間を、計測境界の異なる複数の区間へ分解した記録。

### 11.2 Solver Time

- **Term ID:** `solver-time`
- **分類:** 性能計測
- **推奨フィールド:** `solver_time_ms`
- **旧フィールド:** `solver_ms`

Solverの探索処理に要した経過時間。Graph生成、Artifact出力、外部SDK変換等を含めない。

### 11.3 End-to-End Time

- **Term ID:** `end-to-end-time`
- **分類:** 性能計測
- **推奨フィールド:** `end_to_end_time_ms`

公開入口でRequest受付を開始してからResult返却までの総経過時間。

### 11.4 Orchestration Time

- **Term ID:** `orchestration-time`
- **分類:** 性能計測
- **推奨フィールド:** `orchestration_time_ms`
- **旧フィールド:** `orchestration_ms`, `truss_ms`

TRUSSによるTask作成、割当て、切替え、結果統合等に要した時間。

### 11.5 Instrumentation Overhead

- **Term ID:** `instrumentation-overhead`
- **分類:** 観測・性能計測
- **推奨フィールド:** `instrumentation_overhead_ms`
- **旧用語・旧フィールド:** `Observation Overhead`, `observation_overhead_ms`

Event生成、集計、Trace、Profile等の観測機能により追加された経過時間。

### 11.6 First Path Elapsed Time

- **Term ID:** `first-path-elapsed-time`
- **分類:** Anytime
- **推奨フィールド:** `first_path_elapsed_ms`
- **旧フィールド:** `time_to_first_path_ms`

Run開始からFirst Pathが確定したEvent時点までの経過時間。

厳密なEvent時刻を取得していない推定値へ、この名称を使用してはならない。

### 11.7 Best Path Elapsed Time

- **Term ID:** `best-path-elapsed-time`
- **分類:** Anytime
- **推奨フィールド:** `best_path_elapsed_ms`
- **旧フィールド:** `time_to_best_found_ms`

Run開始から最終的に返却されたIncumbent Pathが初めて確定したEvent時点までの経過時間。

### 11.8 Improvement Count

- **Term ID:** `incumbent-improvement-count`
- **分類:** Anytime
- **推奨フィールド:** `incumbent_improvement_count`
- **旧フィールド:** `improvement_count`

Incumbent Path Costが厳密に改善した回数。

Pathを返したTask数やCandidate数を代用してはならない。

### 11.9 System Metrics

- **Term ID:** `system-metrics`
- **分類:** 性能計測

Memory、Allocation、GC等、探索Workとは別に取得するRuntime指標。

### 11.10 Heap Allocation Boundary Maximum

- **Term ID:** `heap-allocation-boundary-maximum`
- **分類:** Memory計測
- **推奨フィールド:** `heap_alloc_boundary_max_bytes`

計測開始前と終了後のHeap Allocation値のうち大きい方。実行中Peakとは異なる。

### 11.11 Heap Allocation Sampled Peak

- **Term ID:** `heap-allocation-sampled-peak`
- **分類:** Memory計測
- **推奨フィールド:** `heap_alloc_sampled_peak_bytes`

一定間隔でSamplingしたHeap Allocationの最大値。真の瞬間Peakを保証しない。

---

## 12. 観測、Trace、再現性

### 12.1 Observation Configuration

- **Term ID:** `observation-configuration`
- **分類:** ULTRASOUND・設定
- **推奨フィールド:** `observation_config`
- **旧フィールド:** `observation`

何をどの粒度で観測するかを指定する入力設定。

### 12.2 Observation Data

- **Term ID:** `observation-data`
- **分類:** ULTRASOUND・結果
- **推奨フィールド:** `observation_data`
- **旧フィールド:** `observation`

Run中に取得されたEvent、集計値、Trace、Profile等の観測結果。

Observation Configurationと同じフィールド名を使用しない。

### 12.3 Observation Level

- **Term ID:** `observation-level`
- **分類:** ULTRASOUND・設定
- **推奨フィールド:** `observation_level`
- **旧構造:** `observation.mode`

観測の粒度を指定する設定。

標準値:

| 値 | 定義 |
|---|---|
| `off` | 探索Eventを収集しない |
| `summary` | 集約Counterと主要指標を収集する |
| `trace` | 時系列Eventを保存可能な形で収集する |
| `profile` | 時間、Memory、Allocation等の詳細計測を行う |

### 12.4 Sample Rate

- **Term ID:** `sample-rate`
- **分類:** ULTRASOUND・設定
- **推奨フィールド:** `sample_rate`

観測対象Eventのうち採取する割合。値域は0より大きく1以下とする。

### 12.5 Event

- **Term ID:** `event`
- **分類:** ULTRASOUND

探索、制御、Candidate、品質、Budget、Profile等の状態変化を表す時系列記録。

### 12.6 Event Class

- **Term ID:** `event-class`
- **分類:** ULTRASOUND

Eventの意味領域を分類する値。

- Control Event
- Candidate Event
- Detail Event
- Profile Event

### 12.7 Trace

- **Term ID:** `trace`
- **分類:** ULTRASOUND

時系列Eventを、順序、意味、関連ID、再現情報とともに保存したArtifact。

### 12.8 Replay

- **Term ID:** `replay`
- **分類:** ULTRASOUND・再現性

保存済みTraceまたはManifestから、実行過程、品質履歴、Budget履歴等を再構成する処理。

Replayは必ずしもSolverの再実行を意味しない。

### 12.9 Quality History

- **Term ID:** `quality-history`
- **分類:** ULTRASOUND・Anytime

時系列のIncumbent Cost、Upper Bound、Lower Bound、Proven Cost Ratio等の履歴。

### 12.10 Budget History

- **Term ID:** `budget-history`
- **分類:** ULTRASOUND・予算

時系列のBudget付与、使用、返却、再配分の履歴。

### 12.11 Deterministic Mode

- **Term ID:** `deterministic-mode`
- **分類:** 再現性

同一入力、Seed、Configuration、環境条件のもとで、探索結果と意味的なEvent順序を再現する実行方式。

### 12.12 Deterministic Sampling

- **Term ID:** `deterministic-sampling`
- **分類:** 再現性・ULTRASOUND

SeedおよびEvent属性に基づき、同一条件で同じEvent集合を採取するSampling方式。

### 12.13 Stable Digest

- **Term ID:** `stable-digest`
- **分類:** 再現性
- **推奨フィールド:** `stable_digest`

正規化された意味データから算出し、非決定的な順序、絶対Path、時刻等に影響されないDigest。

Graph専用のGraph Digestとは区別する。

### 12.14 Environment Metadata

- **Term ID:** `environment-metadata`
- **分類:** 再現性

OS、Architecture、Runtime Version、CPU、Memory、Commit、Build Configuration等、結果解釈と再現に必要な実行環境情報。

### 12.15 Effective Configuration Digest

- **Term ID:** `effective-configuration-digest`
- **分類:** 再現性

Default適用、互換変換、正規化後に実際に使用されたConfigurationから算出するDigest。

### 12.16 Reproduction Manifest

- **Term ID:** `reproduction-manifest`
- **分類:** 再現性

Runを再現するために必要なScenario、Graph、Dataset Provenance、Seed、Execution Target、Version、Environment等をまとめたArtifact。

---

## 13. Artifactと公開契約

### 13.1 Artifact

- **Term ID:** `artifact`
- **分類:** 出力

Benchmark、Trace、Dataset、Summary、Manifest等として保存される成果物。

### 13.2 Artifact ID

- **Term ID:** `artifact-id`
- **分類:** 出力
- **推奨フィールド:** `artifact_id`

Artifactを安定して識別するID。File Pathと同義ではない。

### 13.3 Output Configuration

- **Term ID:** `output-configuration`
- **分類:** TRAFFIC・設定

Output Directory、Artifact ID、Raw Result保存、Trace保存、Environment Metadata取得等を指定する設定。

### 13.4 Schema Version

- **Term ID:** `schema-version`
- **分類:** 公開契約
- **推奨フィールド:** `schema_version`

JSON等のデータ契約の構造と意味のVersion。単なるアプリケーションVersionではない。

### 13.5 Failure Reason

- **Term ID:** `failure-reason`
- **分類:** 結果・検証
- **推奨フィールド:** `failure_reason`

Runまたは検証が成功条件を満たさなかった理由を、機械判定可能な分類として表す値。

自由文のError Messageとは分離する。

### 13.6 Unsupported Feature

- **Term ID:** `unsupported-feature`
- **分類:** 公開契約

要求された機能、Configuration、Graph特性、Solver Option等を実装がサポートしていない状態。

失敗や不正入力と区別する。

### 13.7 Telemetry

- **Term ID:** `telemetry`
- **分類:** 補助情報

型付きResult契約へまだ昇格していない補助的観測値。

主要な研究指標、制御判断、互換性判定をTelemetry Mapだけへ依存させてはならない。安定して参照する値は型付きFieldへ昇格する。

---

## 14. HEALTHYと検証

### 14.1 HEALTHY

- **Term ID:** `healthy-component`
- **分類:** コンポーネント

BRIDGEの結果、Path、Distance、Work、Ledger、Trace、Reference比較、Regression Policy等を検証するコンポーネント。

### 14.2 Health Check

- **Term ID:** `health-check`
- **分類:** HEALTHY

一つ以上の検証規則を実行し、Pass、Fail、Warning等を返す処理。

### 14.3 Health Profile

- **Term ID:** `health-profile`
- **分類:** HEALTHY

適用するHealth Check、閾値、厳格度をまとめた設定。

### 14.4 Result Validation

- **Term ID:** `result-validation`
- **分類:** HEALTHY

Route ResultまたはRaw Runの構造、状態、Path、Cost、証明、Work等が契約と整合するか検証する処理。

### 14.5 Path Validation

- **Term ID:** `path-validation`
- **分類:** HEALTHY

返却PathのNodeとEdgeがGraph上に存在し、連続性、方向、Source、Target、制約を満たすか検証する処理。

### 14.6 Cost Consistency

- **Term ID:** `cost-consistency`
- **分類:** HEALTHY
- **旧用語:** `Distance Consistency`

返却されたPathから再計算したCostと、Resultに記録されたCostが許容誤差内で一致する性質。

### 14.7 Work Accounting Validation

- **Term ID:** `work-accounting-validation`
- **分類:** HEALTHY

Reported Work、Action内訳、Budget Ledger、Trace等の整合性を検証する処理。

### 14.8 Ledger Validation

- **Term ID:** `ledger-validation`
- **分類:** HEALTHY

Budget Ledgerの付与、使用、返却、残量、親子関係が保存則と契約を満たすか検証する処理。

### 14.9 Trace Verifiability

- **Term ID:** `trace-verifiability`
- **分類:** HEALTHY・ULTRASOUND

Traceから主要なRun結果、Work、品質履歴、Budget履歴を検証または再構成できる性質。

### 14.10 Invalid Run

- **Term ID:** `invalid-run`
- **分類:** HEALTHY・統計

契約違反、Path不正、計測欠損、Reference不整合等により、統計評価へ含めてはならないRun。

AlgorithmがPathを発見できなかったRunと自動的に同義ではない。

### 14.11 Regression Policy

- **Term ID:** `regression-policy`
- **分類:** HEALTHY・回帰評価

Regression Baselineに対する許容差、閾値、統計条件、判定方法を定義する規則。

### 14.12 Performance Improvement / Regression

- **Term ID:** `performance-change`
- **分類:** 回帰評価

同一条件のPerformance BaselineまたはRegression Baselineに対する、時間、Work、Memory、品質等の改善または退行。

改善・退行の方向は指標ごとに定義する。

---

## 15. BRIDGEコンポーネント

### 15.1 BRIDGE

予算管理型Anytime経路探索スタック全体、およびGATEから公開される統合経路探索対象。

### 15.2 TRUSS

Route Requestの制御、Portfolio作成、Budget管理、Solver Taskの割当て、Fallback、Recovery、Certification、結果統合を担う。

### 15.3 ANCHOR

BRIDGEの主要な候補経路探索アルゴリズム。複数Hypothesis、Corridor、Portal、Hub、Detour、Repair等を利用し得る。

### 15.4 BOLTS

TRUSSまたはANCHORから使用される補助Solver群。Dijkstra、Bidirectional Dijkstra、A*、Weighted A*、Reachability等を含む。

### 15.5 GATE

CLI、SDK、外部プログラム等へ公開する入力・出力境界。内部コンポーネント構造を直接公開しない。

### 15.6 TRAFFIC

Scenario実行、Benchmark Suite、比較評価、Raw Run、Summary、Acceptance、Artifact生成を担う研究・検証コンポーネント。

Production Route制御には使用しない。

### 15.7 ULTRASOUND

Event、Observation Data、Trace、Profile、Replay用情報等の観測を担う研究・検証コンポーネント。

Production Routeの探索判断を観測結果へ依存させない。

### 15.8 HEALTHY

結果契約、Path、Cost、Work Accounting、Budget Ledger、Trace、Reference、Regression等の検証を担う。

### 15.9 Core

Graph、Route Request、Route Result、Work Metrics、Budget、共通型等、複数コンポーネントが共有する安定したドメイン契約を提供する。

### 15.10 BEARING

旧または互換上の観測境界名称。現行Architectureで独立コンポーネントとして採用しない場合は、Legacy名称としてのみ扱う。

---

## 16. 非推奨用語と移行先

| 非推奨用語・フィールド | 問題 | 推奨移行先 |
|---|---|---|
| `exact` | 最適性、探索完了、到達可能性証明が混在 | `optimality_proven`, `search_completed`, `reachability_proven`, `matches_exact_reference` |
| `found` | 何を発見したか不明 | `path_found` |
| `mode` | 対象不明 | `route_mode`, `observation_level`, `execution_mode` |
| `strategy` | 対象不明 | `query_selection_method` |
| `opposite-corners` | Generatorにより意味が異なる | `generator_default_endpoints` |
| `average_*` | Meanか不明 | `mean_*` |
| `average_time_ms` | 計測境界不明 | `mean_solver_time_ms`または`mean_end_to_end_time_ms` |
| `time_ms` | 計測境界不明 | 対象を含む時間Field |
| `distance` | 物理距離と誤解される | `cost`。互換契約では定義を明示 |
| `distance_ratio` | 実測比か証明比か不明 | `cost_ratio_to_exact_reference` |
| `quality_certified` | 何を証明したか不明 | `quality_bound_proven` |
| `certified_ratio` | Cost比であることが不明 | `proven_cost_ratio` |
| `max_suboptimality` | 差または比率か不明 | `maximum_allowed_cost_ratio` |
| `workers` | 実Thread数と誤解される | `logical_worker_count` |
| `parallel_steps` | Logical/Scheduledの区別がない | 廃止 |
| `improvement_count` | 改善の対象不明 | `incumbent_improvement_count` |
| `time_to_first_path_ms` | Event時刻でない推定値にも使われ得る | `first_path_elapsed_ms`。正確なEvent計測時のみ |
| `time_to_best_found_ms` | Bestの定義が曖昧 | `best_path_elapsed_ms` |
| `noise` | 何のNoiseか不明 | `edge_weight_noise` |
| `k` | 意味不明 | `neighbor_candidate_count` |
| `communities` | 数値の役割が不明 | `community_count` |
| `nodes` | 要求数か実数か不明 | `requested_node_count`, `actual_node_count` |
| `target` | 終点か実行対象か不明 | `target_node`, `execution_target` |
| `solver_name` | 表示名かIDか不明 | `algorithm_id`または`solver_id` |
| `observation` | 設定と結果が混在 | `observation_config`, `observation_data` |
| `baseline` | 参照目的が不明 | `exact_reference`, `performance_baseline`, `regression_baseline` |

---

## 17. 禁止される曖昧な用法

以下を禁止する。

1. Search Completedを`Exact`と呼ぶこと。
2. Reachability Provenを`Exact`と呼ぶこと。
3. Exact ReferenceとCostが一致しただけで、Solver ResultをOptimality Provenとすること。
4. Reachability SolverのPathを最短Pathとして評価すること。
5. Query IDだけでGraph Diameter、Cross-community等の選択条件を保証したことにすること。
6. Logical Worker Countを実Thread数またはCPU Core数として説明すること。
7. Trace、JSON出力、GC等を探索Workへ含めること。
8. `0`をNot MeasuredまたはNot Applicableの代用にすること。
9. Solver TimeとEnd-to-End Timeを無修飾の`time`として集計すること。
10. Proven Cost RatioとCost Ratio to Exact Referenceを同じ`ratio`として扱うこと。
11. Performance BaselineをProduction Routeの探索判断へ入力すること。
12. Telemetry Mapのみを主要な公開研究指標の正本にすること。

---

## 18. 用語追加・変更チェックリスト

新しい公開用語またはフィールドを追加・変更する際は、次を確認する。

- [ ] 一般的な既存用語で表現できないか
- [ ] 名前から対象と単位が分かるか
- [ ] 設定、状態、結果、証明を混同していないか
- [ ] 既存用語と意味が重複していないか
- [ ] 「意味しないもの」を明記すべきか
- [ ] 対応する推奨フィールド名が明確か
- [ ] 0、未計測、対象外を区別できるか
- [ ] JSON Schema、Go型、CLI、SDKへ同じ意味で反映されるか
- [ ] TRAFFIC、ULTRASOUND、HEALTHYで同じ意味を維持できるか
- [ ] 保存済みArtifactとの互換性に影響するか
- [ ] Schema Versionの更新が必要か

---

## 19. 移行原則

1. 旧フィールドを直ちに削除せず、新フィールドと一定期間併記する。
2. 旧フィールドはDeprecatedとしてSchemaと文書へ明記する。
3. 同一Artifact内で旧・新フィールドが矛盾する場合は、ArtifactをInvalidとする。
4. SummaryおよびAcceptanceは新フィールドを正本として計算する。
5. `exact`の互換変換は、Algorithmごとの推測で自動変換せず、元の意味を判別できる場合に限る。
6. 移行完了後も、旧Artifact ReaderはVersion付き互換層として隔離する。
