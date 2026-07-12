# BRIDGE 用語定義

**文書ID:** BRIDGE-WORD-DEFINITION-001  
**版:** v1.2  
**対象:** Python参照実装、Go実装、ベンチマーク、trace、設計文書

## 1. 目的

本書は、BRIDGEで使用する独自用語と計測用語の意味を固定する。コード、テスト、ベンチマーク、設計文書で別の意味を使用してはならない。

## 2. 用語管理規則

### 独自用語

一般的な技術用語と同じ表記であっても、BRIDGE内で意味、責務、計測範囲、使用可能なコンポーネント、または判定条件を限定している語を独自用語として扱う。

独自用語をコード、schema、CLI、Scenario、trace、結果artifact、テスト、設計文書へ追加または変更する場合、同一変更で本書へ次を記載する。

- 正式表記と、必要な場合は日本語表記
- BRIDGEにおける定義
- 含むものと含まないもの
- 所有または算出するコンポーネント
- 混同しやすい関連語との差異
- 外部公開値である場合は対応するfield名または値

本書に定義されていない略語や、文脈によって意味が変わる無修飾語を新しい公開契約へ導入してはならない。既存の一般語をBRIDGE固有の意味で使用し始める場合も、新語追加と同様に本書を更新する。

### 正式表記

コード識別子やschema fieldは各契約の表記を使用し、文書本文では本書の見出し表記を使用する。`Run`と`run`、`Work`と`work`のように大文字小文字で規範語と一般語を区別している場合、その区別を維持する。

### 用語変更

定義変更は単なる文言修正として扱わない。影響するschema、実装、テスト、Scenario、trace replay、既存artifact互換性を確認する。意味を破壊的に変更する場合は、schema versionまたは契約versionの更新要否を明示する。

## 3. 探索量・並列性

### Work

経路探索中に実行された、意味を持つ探索アクションの総数。CPU命令数、関数呼出し数、経過時間ではない。

標準Actionは次のとおり。

| Action | 定義 |
|---|---|
| `SELECT` | frontier等から次の探索対象を選択する |
| `EXPAND` | 選択した探索状態を展開する |
| `EVALUATE` | node、edge、候補、制約を評価する |
| `RELAX` | 距離またはcost更新を試行する |
| `ENQUEUE` | 探索候補をfrontierへ追加する |
| `REJECT` | stale候補、非改善候補、制約違反候補、探索枝を棄却する |
| `BACKTRACK` | 親状態または分岐点へ戻る |
| `CONNECT` | forward/backward laneや部分経路を接続する |
| `CANDIDATE` | 完全経路または部分経路候補を生成する |
| `REPAIR` | 既存経路の区間を置換・改善する |
| `BOUND` | upper boundまたはlower boundを有効に更新する |
| `TERMINATE` | 探索laneまたはtaskの終了判断を確定する |

`Total Work`は全Action数の合計である。telemetry、JSON/CSV出力、GC、mutex待機、ファイル読込み、外部ID変換は探索Workへ含めない。

### Step

依存関係を保ちながら同時実行可能な複数Workを一つにまとめた論理実行段階。

- `Logical Step`: 無制限の並列資源を仮定した最小段階数
- `Scheduled Step`: 指定worker数で配置した段階数
- 逐次実装では `Work = Logical Steps = Scheduled Steps`
- 常に `Logical Steps <= Scheduled Steps <= Work`

### Lane

独立または半独立に進行できる探索系列。例としてforward lane、backward lane、corridor lane、hypothesis laneがある。

### Worker

同時に一つ以上のWorkを実行できる実行資源の論理単位。CPU thread、goroutine実行枠、process、GPU execution unit等へ対応し得るが、Worker数と物理core数は同義ではない。

### Work Budget

TRUSSが所有する、portfolio全体で消費可能なWork上限。solverは割り当てられたsliceを超えてはならない。

### Step Budget

許容されるScheduled StepまたはLogical Stepの上限。Work Budgetとは独立した制約である。

### Budget Slice

TRUSSが一つのSolverTaskへ事前割当てするWork、deadline、worker、memoryの部分予算。

### First Path Work

最初の有効経路が確定するまでにportfolio全体で消費したWork。

### First Path Step

最初の有効経路が確定するまでのScheduled Step。

### Parallelism

`Total Work / Logical Steps`。アルゴリズムが持つ理論的な平均並列度を表す。

## 4. 経路・品質

### Path

sourceからtargetへ連続するedge列。全edgeがgraphに存在する場合のみ有効。

### Candidate

最終採用前の完全経路または部分経路。Candidate生成は最適性を意味しない。

### First Path

実行中に最初に発見された有効なsource-target path。

### Best Path

現時点で既知の有効Candidateのうち、目的costが最小のpath。

### Exact

返却pathが指定目的関数に対する最適解であることを、exact solverまたは同等の証明手段で確定した状態。

### Anytime

予算が少なくても有効解を返し、追加予算に応じて品質または証明を改善できる性質。

### Upper Bound

既知の有効path cost。最適cost以下ではなく、最適cost以上の境界。

### Lower Bound

最適costを超えないことが保証された境界。

### Certified Ratio

`Upper Bound / Lower Bound`。1.0はexactを示す。lower boundが0または未確定の場合は算出しない。

### Max Suboptimality

利用者が許容するCertified Ratio上限。この値以下を証明できれば品質目標を達成したとみなす。

### Quality Certified

返却結果が指定された品質比以下であることをboundまたはexact solverで証明した状態。

### Baseline

TRAFFICによるbenchmark、性能評価、回帰試験において、評価対象を比較するために選択された参照結果、参照アルゴリズム、参照version、または参照実装。

Baselineは評価上の役割名であり、特定のsolverやアルゴリズムの固有属性ではない。DijkstraがBOLTSに存在するだけではbaselineとは呼ばず、TRAFFICが比較基準として選択した場合にのみbaselineとなる。

BaselineをANCHOR、BOLTS、またはTRUSSの通常探索制御へ入力してはならない。通常探索が使用できるのは、自身の探索状態から得られるUpper Bound、Lower Bound、Certified Ratio、進捗、残予算等である。

無修飾の `baseline` は意味が曖昧なため、コード、出力、文書では原則として次の限定語を使用する。

| 用語 | 定義 | 主な用途 |
|---|---|---|
| `Exact Baseline` | 最適costまたは到達不能という正解を提供する参照結果または参照solver | Distance Ratio、exact一致率、到達性一致率 |
| `Performance Baseline` | 時間、Work、Step、memory、初期解品質、改善曲線等を比較する参照アルゴリズム | アルゴリズム間性能比較 |
| `Regression Baseline` | 変更前のversion、commit、設定、または保存済み評価結果 | 性能退行・品質退行の検出 |
| `Reference Implementation` | 移植先や別実装の意味的整合性を確認する参照実装 | Python-Go parity、差分試験 |

### Exact Baseline

評価対象と同一のGraph、RouteRequest、目的関数、制約条件で求めた最適costまたは到達不能判定。原則として、正しさが確認されたexact solverから取得する。

Exact BaselineはTRAFFIC等の評価経路でのみ使用する。ANCHORの候補生成、進行判断、停止判断、fallback判断、またはTRUSSの通常運用判断へ参照させてはならない。

### Performance Baseline

評価対象の探索性能やAnytime性能を比較するための参照アルゴリズム。最適解を返す必要はない。Dijkstra、bidirectional Dijkstra、A*、Weighted A*、Anytime Weighted A*、ARA*等が、評価目的に応じて選択され得る。

### Regression Baseline

変更による退行を判定するための過去version、commit、設定、または保存済みbenchmark結果。正解値ではなく、変更前の性能水準を表す。

### Reference Implementation

別言語実装、移植元実装、または独立実装との意味的整合性を確認するための参照実装。性能上の優劣を示す語ではない。

### Distance Ratio

評価時の `result distance / exact baseline distance`。アルゴリズム内部の証明値ではなく、TRAFFICがExact Baselineとの比較で計算する評価値。通常探索中のANCHOR、BOLTS、TRUSSはこの値を参照しない。

## 5. 探索構造

### Frontier

今後処理される探索状態の集合。priority queue、beam、bucket等で表現される。

### Corridor

source-target間の幾何学的または構造的仮説に基づき、探索対象を制限した部分領域。

### Portal

離れた領域間を接続する可能性が高いnodeまたはedge endpoint。長距離edge、community境界等から選定される。

### Hub

degree、centralityまたは接続性が周囲より高く、経路接続点として利用価値が高いnode。

### Hypothesis

ANCHORが試行する探索方針。geometric、weighted、bidirectional、hub、portal、community、diverse、repair等を含む。

### Detour

局所障害、高cost区間、barrier等を回避する部分経路。

### Repair

既存pathの一部を再探索し、より短い、確実な、または制約適合したsubpathへ置換する処理。

### Fallback

ANCHORによる候補生成が不十分な場合に、TRUSSがBOLTSへ明示的に切り替える処理。ANCHOR内のalternate hypothesisはFallbackに含めない。

### Certification

exact solverまたはlower bound providerによって、最適性または品質比を証明する処理。

### Reachability

sourceからtargetへ到達可能かを判定する性質または専用BOLTS task。距離最適化とは区別する。

### Stagnation

一定WorkまたはStepの間、upper bound、lower bound、candidate生成、frontier進展に有意な改善がない状態。

## 6. 実行・制御

### Portfolio

一つのRouteRequestを処理するためにTRUSSが運用するANCHOR/BOLTS task全体。

### SolverTask

TRUSSが作成する一つの探索実行単位。purpose、budget slice、deadline slice、worker slice、quality targetを持つ。

### Session

中断、再開、cancel、snapshotが可能なsolver実行状態。

### Probe

少量予算でgraphまたはqueryとの相性、進展率、到達性等を評価する短いtask。

### Deadline

経過時間に基づく実行終了制約。Work Budgetとは独立し、観測I/Oの影響を受け得るため結果比較では別管理する。

### Cancellation

外部要求またはTRUSS判断による実行中断。deadline超過とは区別する。

### Memory Budget

portfolioまたはtaskが保持可能な探索用memory上限。単なる希望値ではなく、強制できる実装のみ保証済みと表記する。

### Scenario

TRAFFICにおいて、再現可能な評価条件を宣言する設定単位。graph条件または入力graph、source、target、execution targets、budget、timeout、stop condition、heuristic、seed、repetitions、warm-up、observation mode、実行順序、出力設定、合否基準等を含み得る。

Scenario自体は1回の探索実行を意味しない。1つのScenarioから複数のRunが生成され得る。

### Generator

TRAFFICがbenchmark用入力graphを生成する方法の種類。Generatorは、nodeの配置方法、edgeの張り方、weightの決め方、接続構造の基本形を定義する。

例:

- `grid`: 2次元格子状にnodeを配置する
- `random_geometric`: 2次元平面上へ点を配置し、幾何的近傍で接続する

Generatorはgraphの基本骨格を決める語であり、障害物配置や通行可能領域の変形そのものはTopologyが担当する。

### Topology

Generatorが作るgraph骨格に対して適用する位相的・構造的な形状条件。Topologyは、障害物、遮断、袋小路、分断、回り込み構造など、探索難易度に影響する空間構造を定義する。

例:

- `open`: 障害物なし
- `wall`: 壁で空間をほぼ二分する
- `u_shape`: U字型障害物を置く
- `culdesac`: 袋小路を作る
- `disconnected`: 到達不能な分断を作る

GeneratorとTopologyの違いは次のとおり。

- Generator: graphの基本骨格を決める
- Topology: その骨格の上で障害物や構造的難しさを与える

たとえば `grid + wall` は、「格子状generatorに wall topology を適用したgraph」を意味する。

### Run

TRAFFICがScenarioから生成する1回の探索実行。原則として `Scenario ID × Execution Target × Seed × Repetition` の組合せで識別する。

Runは評価・記録の最小単位であり、必要に応じてRun ID、execution order、warm-upフラグ、effective configuration digest、observation mode、target kind、execution path等を持つ。

### Benchmark Suite

共通の評価目的、条件、合否基準に基づく複数Scenarioの集合。TRAFFICはBenchmark Suiteを読み込み、Scenarioを検証し、各ScenarioからRun群を生成して集計する。

## 7. 研究ベンチマーク実行

### Query

一つのGraph Instanceに対するsource、targetおよび経路要求条件の組。TRAFFICが評価入力として所有し、solver内部の探索候補をQueryとは呼ばない。

### Query ID

Scenario内でQueryを一意に識別する安定した文字列。sourceとtargetが同じでも、制約や目的関数が異なるQueryには異なるQuery IDを付ける。

### Graph Instance

Generator、Topology、seed、入力dataset、前処理条件等から確定した、Runで実際に使用する一つのgraph実体。同じGenerator名でもseedまたは生成条件が異なれば別Graph Instanceである。

### Graph Instance ID

Graph InstanceをRun間で識別する安定した文字列。Generator名だけをGraph Instance IDとしてはならない。

### Repetition

同一のGraph Instance、Query、Execution Targetおよび有効設定を、測定ばらつきの取得を目的として繰り返す番号付き実行。seedを変える実行とは区別する。

### Warm-up

集計対象Runの前に、初期化、JIT相当処理、cache、allocation状態等の初回影響を分離する目的で実行する非集計Run。Go solverの探索結果を事前学習させる処理ではない。Warm-up Runはraw記録してよいが、通常のsummary、acceptance、統計検定へ含めない。

### Execution Target

TRAFFICが一つのRunで評価対象として指定する公開実行対象。例はBRIDGE全体、ANCHOR単独、またはBOLTSの特定solverである。Execution Targetは比較対象の指定であり、Baselineであることを自動的には意味しない。

### Target Kind

Execution Targetの分類を表す公開値。BRIDGE全体経路とsolver単独経路を区別し、TRAFFICが適切なGATE公開APIを選択するために使用する。アルゴリズム名そのものではない。

### Execution Path

Runが通過した公開呼出し経路の分類。通常のBRIDGE運用を表す`Route`と、研究比較用の単一solver実行を表す`ExecuteOnce`等を区別する。内部packageのcall graphを意味しない。

### Run ID

一つのRunを一意に識別する文字列。少なくともScenario ID、Execution Target、Graph Instance ID、Query ID、Repetitionを識別可能にする。表示順序や時刻だけから生成してはならない。

### Raw Run

一回のRunについて保存する未集約の観測record。入力識別子、実効設定、経路結果、Work、Step、時間、system metrics、error、環境情報等を含み得る。Raw Runからsummaryを再計算できなければならない。

### Raw Observation

統計集約前の測定値。Raw Run全体またはその中の個別測定値を指す。平均値やp95等のsummary値はRaw Observationではない。

### Summary Statistics

Raw Observation群からTRAFFICが算出する記述統計。平均、標本標準偏差、最小値、分位点、最大値、信頼区間等を含む。summaryだけを正本としてraw値を破棄してはならない。

### Artifact

Scenario、raw result、summary、trace、manifest等、再利用または検証のために保存される出力物。探索中のCandidateや内部stateは、保存契約を持たない限りArtifactとは呼ばない。

### Artifact ID

ArtifactをScenarioおよび他Artifactから参照するための安定した識別子。ファイルパスと同義ではない。

### Environment Metadata

Runの再現性判断に必要な実行環境情報。BRIDGE version、commit、Go version、OS、architecture、CPU、worker設定等を含み得る。探索アルゴリズムの入力には使用しない。

### Effective Configuration Digest

既定値適用後にRunで実際に使用した、決定論へ影響する設定から生成するdigest。時刻、出力先、elapsed time等の非決定値を含めない。Stable Digestは結果の意味的同一性を表し、Effective Configuration Digestは入力設定の同一性を表す。

### Acceptance

Scenarioに宣言された合否条件をRaw Runまたはsummaryへ適用した評価結果。solver自身の成功判定、`Found`、`Exact`、`Quality Certified`とは区別する。

## 8. コンポーネント

### BRIDGE

予算、品質、deadline、worker、memory制約に基づき、主探索と補助solverを統合運用するAnytime経路探索stack。

### TRUSS

計画、portfolio予算、task scheduling、solver選択、fallback、certification、終了判断、最終結果選択を所有する制御層。

### ANCHOR

複数Hypothesis、Corridor、Candidate、Repairによりfirst pathの早期発見と改善を行うBRIDGE固有の主探索。

### BOLTS

Dijkstra、bidirectional Dijkstra、A*、reachability、detour、repair、certification等の交換可能な補助solver群。

### BEARING

探索層と観測層の間にある非干渉なevent/metric契約。

### ULTRASOUND

BEARING eventを収集し、trace、replay、分析、可視化を行う開発・検証専用基盤。

### TRAFFIC

scenario生成、benchmark、stress、soak、回帰判定、Exact Baseline・Performance Baseline・Regression Baselineとの比較を行う開発・検証専用基盤。

### GATE

API、CLI、SDK、validation、外部ID変換、serializationを担当する公開境界。

### Core

Graph、RouteRequest、RouteResult、Budget、Bounds、WorkMetrics等の共有data contract。solver判断を持たない。

## 8. 計測内訳

### Time Breakdown

一つの探索結果に付随する責務別の経過時間内訳。COREが値契約を所有し、各コンポーネントが自らの測定事実を記録する。時間値はWorkではなく、observer結果による制御判断にも使用しない。

| field | BRIDGEにおける意味 |
|---|---|
| `total_ms` | 公開実行境界で測定した全体経過時間 |
| `solver_ms` | 選択されたsolver群の実行時間。単独solverでは当該solver時間 |
| `truss_ms` | TRUSSの実行境界全体 |
| `anchor_ms` | ANCHOR solver内部の実行時間 |
| `bolts_ms` | BOLTS solver内部の実行時間合計 |
| `fallback_ms` | fallbackおよびそれに直接付随する処理時間 |
| `supervisor_ms` | Supervisorの判定処理時間 |
| `arbiter_ms` | Arbiterの候補比較・選択処理時間 |
| `orchestration_ms` | solver本体以外のTRUSS制御時間。重複計上を避けた派生値 |
| `gate_ms` | GATEのvalidation、変換、結果整形等の公開境界時間 |

各fieldは可能な限り排他的に定義する。包含関係があるfieldを単純合算して`total_ms`と比較してはならない。

### Solver Time

探索solverが経路候補の生成、改善または証明を行った時間。GATE変換、TRAFFIC集計、artifact I/O、runtime metrics取得は含まない。

### End-to-End Time

評価呼出し開始から公開結果受領までの経過時間。Solver Timeに加え、GATEおよびTRUSS制御等を含み得る。artifact保存や後処理を含める場合は別fieldとして明示する。

### Phase

責務またはアルゴリズム上の意味で区切られた実行区間。単なる関数名や任意のprofiling spanではない。Phase名を公開traceまたはresultへ追加する場合は本書または対応するtrace契約に定義する。

### System Metrics

TRAFFICまたはprofile機構がRun単位で採取するruntime・memory観測値。探索Work、Budget消費、品質判定には使用しない。

| field | BRIDGEにおける意味 |
|---|---|
| `alloc_bytes` | Run前後のruntime累積割当byte差分 |
| `malloc_count` | Run前後のmalloc累積回数差分 |
| `gc_count` | Run前後のGC累積回数差分 |
| `heap_alloc_before` | Run開始直前のlive heap bytes |
| `heap_alloc_after` | Run終了直後のlive heap bytes |
| `heap_alloc_boundary_max` | 現行summary計測では開始前後値の最大。厳密な瞬間peakではない |

### Telemetry

型付き結果契約に含めにくい補助的観測値を保持する付加情報。主要な研究指標、制御判断、互換性判定をTelemetry mapだけへ依存させてはならない。安定して参照する値はCOREまたは対応schemaの型付きfieldへ昇格する。

## 9. 観測・再現性

### Event

探索中に発生した意味的事象をBEARING契約で表したrecord。

### Observation Mode

ULTRASOUNDがBEARING Eventをどの粒度で収集・保存するかをScenarioまたは公開設定で指定するmode。観測modeはpath、distance、Found、Exact、Work、Step、tie-breaking、seed消費順を変更してはならない。

| 値 | 定義 |
|---|---|
| `off` | eventの収集・保存を行わない。Null Observer相当 |
| `summary` | 研究用counterと集約値を保持し、完全なEvent列は保存しない |
| `trace` | replayおよびquality履歴再構成に必要なEvent列を保存する |
| `profile` | 時間・allocation・memory等の詳細profilingを行う。純粋速度比較とは分離する |

`metrics`および`debug`は現行の正式なObservation Modeではない。

### Trace

sequence、task、phase、lane、logical step、payloadを含むEvent列。

### Replay

保存Traceから実行状態または可視化状態を再構成する処理。

### Deterministic Mode

同一version、graph、request、seed、budget、worker条件でpath、distance、Work、Step、solver traceを同一にする実行mode。

### Stable Digest

実時間等の非決定値を除外し、意味的結果から生成するhash。

### Golden Case

固定graph、request、期待結果を持つ互換性fixture。

### Differential Test

Python参照版とGo版等、複数実装へ同じCaseを入力して意味的結果を比較するtest。

## 10. 禁止される曖昧な用法

- `Work`を展開node数だけの意味で使用しない。
- `Step`を経過時間または単純な `ceil(Work/Workers)`として使用しない。
- alternate hypothesisをFallbackと呼ばない。
- baselineとのDistance RatioをCertified Ratioと呼ばない。
- 推定値を実測Parallel Stepsとして出力しない。
- memory上限を強制していない状態でMemory Budget遵守済みと表記しない。
- Execution Targetを自動的にBaselineと呼ばない。
- Warm-up Runを通常Runのsummaryやacceptanceへ混入させない。
- `heap_alloc_boundary_max`を厳密な瞬間peak memoryと表記しない。
- Stable DigestとEffective Configuration Digestを同じ意味で使用しない。
- typed contractに昇格すべき主要指標をTelemetryだけで公開しない。

## Work内訳と非探索オーバーヘッド

- `Work` は探索アルゴリズムが実行した離散的な意味Actionのみを数える。
- `node_expansions` 相当は `expand_actions`、`edge_scans` 相当は `evaluate_actions`、`relax_attempts` 相当は `relax_actions`、`queue_pushes` 相当は `enqueue_actions` として記録する。
- 時間計測、JSON直列化、ULTRASOUNDの保存I/O、runtime memory計測は`Work`へ含めない。
- `TimeBreakdown`および`SystemMetrics`は研究評価用の観測値であり、探索予算の消費量ではない。

## Phase 3 観測語彙

### Event Class
BEARING eventをULTRASOUNDの保存粒度へ対応付ける分類。`control`、`candidate`、`detail`、`profile`の4種を正規分類とする。分類は観測側の都合で探索制御へ影響させてはならない。

### Control Event
探索開始・終了、component開始・終了、budget変更、fallback、certification、TRUSS判断など、実行構造を示す低頻度event。

### Candidate Event
候補経路の提出、incumbent更新、connector成否など、品質推移の再構成に必要なevent。

### Detail Event
frontier、node expansion、edge evaluation、relaxationなど、探索状態のreplayに必要なevent。

### Profile Event
意味的Action単位など高頻度で、観測オーバーヘッド分析に限定して使用するevent。通常のtraceには含めない。

### Observation Overhead
Observerへのevent配送、選別、集計、保存に費やした時間。探索Workには含めない。`overhead_ns`はobserver処理全体、`sink_write_ns`はそのうちsink書込み時間を表す。

### Quality History
candidateまたはincumbent更新を、sequence、elapsed time、Work、distanceの系列として再構成したもの。

### Budget History
budget拡張eventから再構成した、Workと拡張上限の変更系列。

## Phase 1〜3補完で追加・確定した用語

### Execution Manifest
TRAFFICが展開後のRun実行順を記録するartifact内契約。`run_order`、randomizeの有無、shuffle seed、shuffle方式を含む。

### Query Summary
同一Scenario・Algorithm内でもQuery IDごとに分離した集計結果。異なるquery特性を一つの平均へ混合しない。

### Deterministic Sampling
同一seed、event ordinal、event kindに対して常に同じ採否を返すULTRASOUNDのsampling方式。現在の方式識別子は`fnv1a-seed-ordinal-kind-v1`。

### Heap Allocation Boundary Maximum
Run開始直前と終了直後の`HeapAlloc`の最大値。実行中の瞬間最大値ではない。JSON fieldは`heap_alloc_boundary_max`。

### Heap Allocation Sampled Peak
`profile` modeにおいてRun実行中に周期samplingした`HeapAlloc`の最大観測値。sampling間隔外の瞬間最大値を保証しない。JSON fieldは`heap_alloc_sampled_peak`。

### Observation Overhead Ratio
ULTRASOUND observer内部処理時間を当該Runのend-to-end時間で除した比率。探索Workには含めない。

## Phase 4 Anytime評価・失敗分類・アブレーション用語

### Time to First Path
Run開始から最初の有効経路が得られるまでの時間。JSON fieldは`time_to_first_path_ms`。経路未発見時は値を持たない。

### Time to Best Found
Run開始から最終的に採用された最良経路が得られるまでの時間。JSON fieldは`time_to_best_found_ms`。

### Improvement Count
最初の有効経路取得後にincumbentが改善された回数。最初の候補提出自体は改善回数へ含めない。

### Failure Reason
実行エラーとは分離されたalgorithmic outcome分類。正規値は`disconnected`、`budget_exhausted`、`timeout`、`fallback_failure`、`invalid_request`、`no_path`とする。

### Ablation
BRIDGEの機構単位の寄与を同一Scenario系列で比較するため、明示的に機構を無効化する実験設定。production既定動作を変更せず、Scenarioから型付きoptionとして注入する。

### Bridge Overhead Ratio
TRUSSのorchestration時間をBRIDGE全体実行時間で除した比率。選択solverとの差分時間ではなく、型付きTime Breakdownから算出する。

### Duplicated Work Ratio
ANCHORとBOLTS間で重複して調査されたnode・edgeの観測数をportfolio total Workで除した診断比率。厳密なAction重複率ではないため、結果解釈時に定義を明示する。

### State Reuse Ratio
後続solverが先行探索状態を再利用した量を、再利用可能状態量で除した比率。現行実装でstate reuseを行わない経路は`0`を記録する。

## Phase 5 research terms

### Dataset
TRAFFICが研究・検証時に読み込む外部graph artifact。GATEのproduction route入力とは別契約であり、`bridge.dataset.v1`、出典、ライセンス、SHA-256、前処理履歴を必須の再現情報とする。

### Dataset Provenance
Datasetの`id`、取得元、ライセンス、byte列SHA-256、前処理履歴をまとめた由来情報。各Raw RunのGraph Metadataへ複製される。

### Graph Family
同一の生成原理またはデータ由来を共有するgraph群。Phase 5では`grid`、`random_geometric`、`community`、`maze`、`adversarial`、`dataset`を区別する。

### Community Graph
密な局所communityと疎なcommunity間bridgeを持つ決定論的人工graph family。

### Maze Graph
randomized depth-first carvingをseed固定で実行して生成するperfect-maze型の決定論的人工graph family。

### Adversarial Graph
主経路に加え、局所的に有望に見えるdead endと高cost shortcutを配置したストレス評価用人工graph family。

### Effect Size
差の有無だけでなく差の大きさを表す研究指標。本基盤の標準非母数effect sizeはCliff's deltaとする。

### Reproduction Manifest
Scenario、実行順序、environment、dataset provenance、raw observationsを組み合わせ、第三者が同一実験を再生成するための記録集合。


## HEALTHYと評価検証用語

### HEALTHY
TRAFFICが生成したartifactをread-onlyで検証し、Result Validation、Work Accounting Validation、Exact Reference照合、Paired Comparison、Regression Policy判定を行う開発・検証専用コンポーネント。探索制御へ介入しない。

### Health Check
一つ以上のbenchmark artifactに対してHEALTHYが行う検証・分析・評価処理。

### Health Profile
HEALTHYの検証許容誤差、Candidate、Performance Reference、Exact Reference、Regression Policyを宣言する設定。Scenarioとは別契約である。

### Result Validation
Found、Path、Distance、Exactをgraphとqueryから独立に検証する処理。

### Path Validation
pathの端点、node存在、edge接続、向き、edge weight合計を検証する処理。

### Distance Consistency
reported distanceとpathから再計算したdistanceが許容誤差内で一致する状態。

### Work Accounting
探索中の意味的ActionをCOREのWork契約に従って計上する処理。

### Work Conservation
TotalActionsがAction別内訳の合計と一致し、LogicalSteps <= ScheduledSteps <= TotalActionsを満たす状態。

### Reported Work
solverまたはTRUSSがWorkMetricsとして返したWork。

### Reconstructed Work
samplingなし・欠落なしのprofile Action EventからHEALTHYが再構成したWork。

### Work Accounting Validation
Reported Workの保存則、および利用可能な場合はReconstructed Workとの一致を検証する処理。

### Work Mismatch
Work保存則、component合算、ledger、またはReported WorkとReconstructed Workの不一致。

### Exact Reference
同一graphとqueryについて最適距離または到達不能を提供する参照結果。現在はGATE.ExecuteOnce経由のDijkstraまたはBidirectional Dijkstraを使用する。

### Performance Reference
Candidateと同一タスクで性能比較する既存algorithm、旧version、または保存済みartifact。

### Candidate
HEALTHYが評価対象とするalgorithm、version、commit、configuration。

### Paired Comparison
scenario、graph instance、query、seed、repetitionが同一のCandidateとPerformance Referenceを一対一比較する処理。

### Regression Policy
正当性、品質、Work、時間、memoryについてpass、warning、fail、invalidを判定する規則。

### Invalid Run
正当性または必須Work不変条件に失敗し、性能集計へ使用できないRun。

### Unsupported Feature
公開契約に存在しても現実装が動作を提供しない機構。設定時に黙って無視してはならない。

### Performance Improvement / Performance Regression
正当性と品質を維持したうえでRegression Policyの改善条件を満たす状態／許容範囲を超えて悪化する状態。

## Budget Ledger
TRUSSが各solver taskへのWork予算付与量と実消費量を記録した型付き会計情報。探索Workそのものではなく、Reported Workとtask別・component別消費量の整合性をHEALTHYが検証するための記録である。

## Ledger Validation
Budget Ledgerのtask別消費合計、component別消費合計、ledger全体のused、およびReported WorkのTotal Actionsが一致するかを検証する処理。

## Trace Verifiability
profile traceがWorkの完全再構成に利用可能な状態。`sample_rate=1.0`、dropなし、truncationなし、trace SHA-256一致を必須とする。

## Derived Compatibility Counter
`WorkRelaxations`等、旧契約との互換性のために残る診断値。Workの正本ではなく、必ず`WorkMetrics`から導出される。独立に更新してはならない。
