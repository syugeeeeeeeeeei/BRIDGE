# BRIDGE プロジェクト知識統合版

> 本文書は、BRIDGEの思想・目的・責務境界・不変条件・用語・評価原則・変更判断基準を、ChatGPTプロジェクト等の情報源へ単一ファイルで登録するための統合版である。

# BRIDGE プロジェクト知識ベース

## 目的

本ディレクトリは、BRIDGEについて長期間維持される思想、目的、責務境界、不変条件、評価原則、変更判断基準を、ChatGPTその他の支援システムへ提供するための知識源である。

ここでは、頻繁に変わる実装詳細、バージョン固有の進捗、個別ファイル名、コマンド、実測結果を原則として扱わない。それらはリポジトリ内の現行コード、仕様書、利用手順、変更履歴、実装報告を参照する。

## 読み順

1. `01_IDENTITY_AND_PURPOSE.md`
2. `02_ARCHITECTURE_AND_RESPONSIBILITIES.md`
3. `03_INVARIANTS_AND_RULES.md`
4. `04_TERMINOLOGY_AND_CONCEPTS.md`
5. `05_RESEARCH_AND_EVALUATION_PRINCIPLES.md`
6. `06_CHANGE_DECISION_GUIDE.md`

## 情報の扱い

### 恒久原則

本ディレクトリに記載された、目的、責務分離、依存方向、予算所有権、観測非干渉、決定論性、実行境界、評価の公正性は、原則として維持する。

### 変更可能な設計

アルゴリズム内部、データ構造、公開API、ファイル構成、実装言語、CLI、SDK、計測項目、個別solver構成は改善の対象である。ただし、恒久原則を侵害してはならない。

### 時点依存情報

バージョン番号、実装済み機能、未実装機能、性能値、テスト件数、対応環境、具体的コマンドは本知識ベースだけで断定しない。必ず現行リポジトリを確認する。

## 規範上の位置づけ

本知識ベースは、BRIDGEの思想と判断軸を要約する長期参照資料である。コード変更に対する直接の規範は、現行のアーキテクチャ規則、用語集、コンポーネント規則、公開契約を優先する。

内容が競合する場合は、次の原則に従う。

1. 現行の最上位規範文書
2. 現行の正式用語集
3. 現行のコンポーネント規則と公開契約
4. 本知識ベース
5. 実装報告、計画、議論記録

報告書や過去の議論は、その時点の事実であり、恒久仕様ではない。

---

# BRIDGEの思想・目的・成功条件

## 1. BRIDGEとは何か

BRIDGEは、有限の探索予算のもとで、できるだけ早く有効な経路を提示し、追加予算に応じて経路品質または品質証明を改善する、予算管理型Anytime経路探索システムである。

BRIDGEは単一の探索アルゴリズム名ではない。主探索、補助solver、予算管理、公開境界、観測、評価、健全性検証を、明確な責務境界で組み合わせる探索スタックである。

## 2. 中心思想

BRIDGEが目指すのは、あらゆる条件で常に単一指標の最高値を得ることではない。制約された予算、時間、計算資源のもとで、次の価値を同時に成立させることである。

- 早期に有効解を返せる
- 追加予算によって解を改善できる
- 必要に応じて品質または最適性を証明できる
- 消費した探索量を説明できる
- 実行を再現できる
- 比較評価を公正に行える
- 実装の変更後も意味が崩れない

この思想は「最高より最善を目指す」という表現に集約される。ただし、これは曖昧な妥協を意味しない。予算、品質、Work、Step、終了理由を明示し、その制約下で最善の結果を選択することを意味する。

## 3. 解決しようとする問題

一般的なexact探索は、最適性を得るまで結果を返せない場合がある。単純な近似探索は、早く結果を返せても、品質の改善過程や保証を説明できない場合がある。

BRIDGEは、次の間にある設計課題を扱う。

- 初期解の速さと最終品質
- 探索Workと並列Step
- 主探索の独自性とexact solverの信頼性
- 実運用の低オーバーヘッドと研究評価の観測可能性
- 柔軟なportfolio制御と厳格な責務分離

## 4. BRIDGEが守る価値

### 4.1 予算を第一級の契約として扱う

探索予算は単なる設定値ではない。誰が所有し、どこへ割り当て、何を消費として数え、なぜ終了したかを説明できなければならない。

### 4.2 初期解と品質改善を分離して評価する

最初の有効経路を得る性能と、最終的に高品質またはexactへ到達する性能は別の能力である。両者を単一の実行時間だけで評価しない。

### 4.3 独自アルゴリズムと既存アルゴリズムを役割で組み合わせる

ANCHORはBRIDGE固有の主Anytime探索を担う。BOLTSはexact、fallback、reachability、repair、certification等の補助能力を提供する。両者の優劣ではなく、portfolio内での責務が異なる。

### 4.4 観測可能性と非干渉を両立する

内部挙動を詳細に観測できなければ研究評価は成立しない。一方、観測の有無によって探索結果、Work、Step、予算配分、solver選択が変わるなら、観測は正当な測定ではない。

### 4.5 再現性を性能と同等に重視する

同一条件で結果を再現できない実験は、比較研究や回帰判定に使用できない。決定論性、seed、実行条件、raw observationを管理する。

## 5. 成功条件

BRIDGEの実装は、経路を返すだけでは成功とみなさない。少なくとも次が成立する必要がある。

- 返却経路が妥当である
- 予算上限を超えない
- WorkとStepの意味が一貫している
- 終了理由を区別できる
- 観測有無で意味的結果が変化しない
- 同一条件で再現できる
- exact baselineとの正しさ比較が可能である
- performance baselineとの性能比較が可能である
- コンポーネント責務と依存方向が保たれている
- 文書、契約、コード、テストの意味が一致している

## 6. BRIDGEが目的としないもの

- 単一のトポロジーだけに最適化された競技用アルゴリズム
- exact solverを隠して独自アルゴリズムの性能として見せる仕組み
- 実行時間だけを最適化し、探索量や品質を説明しない実装
- benchmark専用経路を本番経路と別物にする設計
- telemetryやtraceを探索制御へ帰還させる設計
- テスト結果に合わせて定義を後付け変更する運用
- 既存アルゴリズムの単なるラッパー

## 7. 長期的な研究方向

BRIDGEの研究価値は、単純な最短経路時間の競争だけではない。次を統合的に扱える点にある。

- Anytime品質曲線
- First PathまでのWorkとStep
- 予算配分の妥当性
- 並列化可能性
- topology別の適性
- fallbackとcertificationの費用対効果
- 観測可能で再現可能な探索過程
- 独自主探索と既存solverのportfolio設計

---

# BRIDGEのアーキテクチャと責務境界

## 1. 基本構造

BRIDGEは、各コンポーネントが一つの主要責務を所有し、他コンポーネントの判断を代行しない構造を採用する。

```text
External Caller
      |
    GATE
      |
    TRUSS
   /     \
ANCHOR  BOLTS
   \     /
   BEARING
      |
 ULTRASOUND

TRAFFIC  -> 公開経路を通じた評価
HEALTHY  -> 生成済みartifactのread-only検証
CORE     -> 全体が共有する中立契約
```

この図は概念上の責務関係を示す。具体的なpackage依存や呼出し構造は現行規範を確認する。

## 2. CORE

### 役割

COREは、中立的な共有値型とデータ契約を提供する。

### 所有するもの

- Graph、Node、Edge等の基本契約
- RouteRequest、RouteResult
- Budget、Bounds、WorkMetrics
- cancellation、deadline、errorの中立表現
- コンポーネント間で共有するが、特定実装へ偏らない型

### 所有しないもの

- solver選択
- portfolio制御
- algorithm固有heuristic
- trace保存
- CLIや外部形式への変換

### 原則

COREは最下層の中立契約であり、上位コンポーネントの事情を取り込まない。

## 3. GATE

### 役割

GATEは、BRIDGEの外部公開境界である。

### 所有するもの

- 入力検証
- 既定値適用
- 外部IDと内部IDの変換
- 公開API用の要求・結果表現
- 外部向けerror mapping
- observer契約の受け渡し

### 所有しないもの

- ANCHORやBOLTSの直接起動
- solver選択
- portfolio予算配分
- 品質終了判断
- trace collectorや保存先の生成
- CLI、HTTP、ファイル等の具体的I/Oそのもの

### 原則

外部利用者はGATEを通じてBRIDGEを利用する。評価経路も、可能な限り同じ公開経路を使用する。

## 4. TRUSS

### 役割

TRUSSは、portfolio制御と予算の唯一の所有者である。

### 所有するもの

- query特性の評価
- task生成
- Budget Sliceの割当
- strategyとsolverの選択
- shared bounds
- fallbackとcertificationの開始判断
- 候補間の最終採用判断
- portfolio全体の終了判断

### 所有しないもの

- frontier操作
- edge relaxation
- solver private stateへの依存
- 観測結果を根拠とする制御変更

### 原則

ANCHORとBOLTSは互いを直接起動しない。TRUSSが両者を調停する。portfolio全体の予算、順序、継続、切替えはTRUSSだけが判断する。

## 5. ANCHOR

### 役割

ANCHORは、BRIDGE固有の主Anytime探索である。

### 所有するもの

- 複数の探索Hypothesis
- Corridor等を用いた主探索
- First Path生成
- Candidate生成と比較
- 局所Repair
- 自身が消費したWorkとStepの報告
- 進捗および緊急状態の報告

### 所有しないもの

- portfolio全体の予算変更
- fallback solverの選択
- 未証明のexact主張
- ULTRASOUNDへの直接依存
- BOLTSの直接起動

### 原則

ANCHORは主探索を担うが、全体最適な運用判断は行わない。必要な支援はTRUSSへ報告し、TRUSSがBOLTSを含む対応を決定する。

## 6. BOLTS

### 役割

BOLTSは、交換可能な補助solver群である。

### 所有するもの

- exactまたは準exact探索
- reachability
- fallback
- detourとrepair支援
- lower bound
- certification
- 自身が消費したWorkとStepの報告

### 所有しないもの

- portfolio scheduling
- ANCHORの継続判断
- 他solverの無制限な連鎖起動
- GATEへの直接公開

### 原則

BOLTSはアルゴリズムの格納場所ではなく、共通契約に従う補助能力の集合である。各solverは、予算境界、決定論性、計測契約を守る。

## 7. BEARING

### 役割

BEARINGは、探索層と観測層をつなぐ非干渉のevent契約である。

### 所有するもの

- typed event
- phase、step、lane等の共通語彙
- Observer interface
- Null Observer
- Safe Observer

### 所有しないもの

- event保存
- 集計
- replay
- bottleneck分析
- 予算変更
- cancellation指示

### 原則

BEARINGは制御チャネルではない。探索から観測へ事実を一方向に伝える。

## 8. ULTRASOUND

### 役割

ULTRASOUNDは、開発・検証・研究のための観測、保存、再生、分析を担う。

### 所有するもの

- event収集
- sequenceや経過時間等の付与
- Observation Modeによる選別
- Sinkへの配送
- trace保存
- metrics集計
- schema検証
- replayと分析

### 所有しないもの

- solver選択
- 予算再配分
- candidate採用
- graph変更
- 本番探索の必須依存

### 原則

ULTRASOUNDは探索結果を観測するが、探索を変えない。観測コストは探索Workへ含めず、別のoverheadとして扱う。

## 9. TRAFFIC

### 役割

TRAFFICは、テスト、benchmark、回帰、比較研究を担う評価基盤である。

### 所有するもの

- graphとqueryの生成・読込み
- ScenarioとRunの生成
- exact baselineとperformance baselineの設定
- benchmark、stress、回帰試験
- raw observationと統計集計
- 受入基準の判定

### 所有しないもの

- solver内部状態の変更
- 実行中の探索制御への介入
- private APIへの依存
- 本番route処理への組込み

### 原則

TRAFFICはBRIDGEを測定する側であり、BRIDGEの探索能力を補助してはならない。baselineは評価にのみ使用し、通常探索へ注入しない。

## 10. HEALTHY

### 役割

HEALTHYは、TRAFFICやULTRASOUNDが生成したartifactをread-onlyで検証する健全性評価層である。

### 所有するもの

- artifactの整合性確認
- Work保存則の検証
- Budget Ledgerとの照合
- traceからのWork再構成
- invalid Runの識別
- 比較・回帰結果の妥当性確認

### 所有しないもの

- solver制御
- artifact書換え
- exact referenceの探索注入
- 独自Work定義
- invalid Runを性能集計へ混入させる判断

### 原則

HEALTHYの評価結果は、同一Runの探索判断へ戻さない。事後検証に限定する。

## 11. 製品系と評価系の分離

### 製品系

- CORE
- GATE
- TRUSS
- ANCHOR
- BOLTS
- BEARING

### 任意観測・評価系

- ULTRASOUND
- TRAFFIC
- HEALTHY

製品系は、評価系が存在しなくても正しく動作できなければならない。評価系は製品系の公開契約を使用し、private implementationへ侵入しない。

---

# BRIDGEが守るべき不変条件とルール

## 1. 予算所有権

- portfolio全体のWork BudgetはTRUSSだけが所有する。
- 各solverは、割り当てられたBudget Sliceを超えてはならない。
- task単位、component単位、portfolio全体のWorkを混同しない。
- deadline、cancellation、Work Budget到達、品質達成を別の終了理由として記録する。
- 強制できない予算を保証済みと表現しない。

## 2. WorkとStepの意味

- Workは意味的な探索Actionの総数である。
- CPU命令数、関数呼出し数、経過時間、I/O時間、GC、mutex待機はWorkではない。
- Stepは、依存関係を保ちながら同時実行できるWorkをまとめた論理実行段階である。
- 並列性を主張する場合、Work、Logical Step、Scheduled Step、Worker数を区別する。
- Work定義は全アルゴリズムで共通でなければならない。
- アルゴリズムごとに有利な独自カウントを導入してはならない。

## 3. 観測非干渉

Observation Modeやobserverの有効・無効によって、次を変化させてはならない。

- path
- distance
- found、exact、quality-certified等の意味的状態
- WorkとStep
- tie-breaking
- seed消費順
- solver選択
- budget配分
- candidate採用

観測失敗は記録するが、探索結果を破壊しない。観測I/Oやprofile overheadは探索Workへ含めない。

## 4. 決定論性

同一の実装版、Graph、Query、Mode、Budget、Worker数、Seedでは、timing以外の意味的結果を再現可能にする。

- adjacency順を正規化する。
- 同順位candidateのtie-breakingを固定する。
- map等の非決定的iteration順へ依存しない。
- candidate適用順を固定する。
- Stable Digestへ経過時間、address、非決定的IDを含めない。
- randomizeが必要な場合、seedと消費規則を明示する。

## 5. 正しさと品質保証

- Graph上に存在しないedgeを含むpathを返してはならない。
- 到達不能と未発見を区別する。
- exactは、exact solverまたは同等の証明がある場合だけ主張する。
- Distance Ratioは評価値であり、実行中の品質証明ではない。
- Certified RatioはUpper BoundとLower Boundに基づく証明値である。
- First Path、Best Path、Final Pathを区別する。

## 6. 責務分離

- TRUSSはfrontier操作を行わない。
- ANCHORはBOLTSを直接起動しない。
- BOLTSはportfolio制御を行わない。
- GATEはsolver選択を行わない。
- BEARINGは観測事実を制御指示へ変換しない。
- ULTRASOUNDは探索判断を変更しない。
- TRAFFICはprivate stateを利用して評価対象を有利にしない。
- HEALTHYはartifactを変更せず、評価結果を同一Runへ戻さない。
- COREは特定アルゴリズムや外部I/Oへ依存しない。

## 7. 本番経路とbenchmark経路

- benchmark専用の特別な探索ロジックを製品経路とは別に実装しない。
- 同じ公開契約と同じsolver実装を使用する。
- benchmarkでは低オーバーヘッドの一回実行経路を使用できるが、意味的な探索経路は通常利用と共通にする。
- 評価の都合で本番コードにshortcut、正解注入、特別caseを追加しない。

## 8. BRIDGE実行境界の原則

BRIDGEの処理およびbenchmark計測は、実際にBRIDGEを外部システムまたはアプリケーションへ組み込んだ場合の責務境界に基づき、`Preparing`、`Running`、`Finalizing`の三つへ分類する。

- `Preparing`は、BRIDGEへ渡す入力を外部で準備する区間である。Scenario解釈、Graph生成、Query生成、実行条件の確定を含む。外部システム、地図アプリケーション、生成script、TRAFFICによるGraph生成はBRIDGEの実行時間へ含めない。
- `Running`は、BRIDGEが入力を受理してから、経路結果および要求されたTrace結果を外部利用可能な形式で返却するまでの区間である。入力検証、内部Graph構築、Graph分析、特徴抽出、方針決定、経路探索、結果構築、Trace生成・直列化・返却を含む。
- `Finalizing`は、BRIDGEが返却した結果を外部で集計、分析、評価、保存または可視化する区間である。複数Run集計、HEALTHY評価、性能回帰判定、bottleneck分析、artifact生成、外部SimulatorによるTrace可視化を含む。

処理の所属は、実装コンポーネントではなく、実運用上の責務と公開interface境界によって決定する。benchmark実装上の都合を計測境界の正本としてはならない。

- Graph生成時間をBRIDGEの実行時間として扱わない。
- 外部GraphをBRIDGE内部表現へ変換する時間をBRIDGEの実行時間から除外しない。
- Graph分析および特徴抽出はRunningに含めるが、RouteまたはSolverの時間とは分離して記録する。
- Trace取得、構築、直列化および返却はRunningに含める。Traceを用いた描画、動画化、比較分析はFinalizingに含める。
- 経路探索性能、BRIDGE全体性能、benchmark全体性能を別の計測値として保持する。

## 9. baselineの隔離

- Exact Baselineは正解との比較に使用する。
- Performance Baselineは性能比較に使用する。
- Regression Baselineは変更前との比較に使用する。
- Reference Implementationは意味的整合性の確認に使用する。
- baselineをANCHORのcandidate生成、TRUSSの通常判断、BOLTSの探索順へ入力しない。

## 10. traceの完全性

- replay可能と主張するtraceは、状態遷移を再構成できる必要がある。
- sampling、drop、truncation、破損、digest不一致があるtraceを完全traceとして扱わない。
- trace schemaはversion管理する。
- 未知eventを安全に無視できる後方互換性を優先する。
- traceは診断logと区別する。

## 11. テストと検証

変更時には、変更対象に応じて次を確認する。

- 単体テスト
- path妥当性
- budget境界
- cancellationとdeadline
- 決定論性
- observer非干渉
- dependency rule
- exact baseline一致
- Work保存則
- trace再構成
- 既存実装またはReference Implementationとのpaired comparison
- topology、規模、seedを変えた回帰

## 12. 文書と用語

- BRIDGE固有の意味を持つ用語は正式用語集へ定義する。
- schema field、CLI option、trace vocabulary、公開resultへ未定義語を追加しない。
- 用語変更は文言変更ではなく契約変更として扱う。
- 文書とコードの不一致は、どちらかを暗黙に正しいものとして扱わず、欠陥として修正する。
- 報告書の時点依存事実を恒久仕様へ昇格させない。

## 13. 禁止される設計上の近道

- exact solverの結果をANCHORの性能として計上する。
- 観測eventからsolver切替えを行う。
- timeout内に収めるためWorkを未計上にする。
- benchmarkでのみWork定義を変更する。
- 失敗Runを黙って集計から除外する。
- warm-up Runを通常Runと混ぜる。
- private implementationへ依存した評価を行う。
- 実行時間だけで優劣を結論づける。
- 特定seedまたは特定topologyだけで一般性を主張する。

---

# BRIDGEの中核用語と概念

本書は、ChatGPTがBRIDGEを説明・分析する際に意味を取り違えやすい中核用語を要約する。完全な定義は現行の正式用語集を参照する。

## 1. 探索量

### Work

意味を持つ探索Actionの総数。時間やCPU命令ではない。SELECT、EXPAND、EVALUATE、RELAX、ENQUEUE、REJECT、CONNECT、CANDIDATE、REPAIR、BOUND、TERMINATE等を共通語彙として数える。

### Step

依存関係を保ちながら同時実行できるWorkをまとめた論理段階。

### Logical Step

無制限の並列資源を仮定した最小Step数。

### Scheduled Step

指定Worker数で実際に配置したStep数。

### Worker

同時にWorkを実行できる論理実行資源。物理CPU coreと同義ではない。

### Parallelism

概念上は `Total Work / Logical Steps` で表す。Workが少ないこととStepが少ないことは別の性能である。

### Work Budget

portfolio全体で消費可能なWork上限。TRUSSが所有する。

### Budget Slice

一つのSolverTaskへ割り当てられた部分予算。

## 2. Anytime品質

### First Path

最初に発見された有効なsource-target path。

### Best Path

現時点で既知のCandidateのうち最小costのpath。

### Exact

返却pathが最適であると証明された状態。

### Anytime

少ない予算でも有効解を返し、追加予算で品質または証明を改善できる性質。

### Upper Bound

既知の有効path cost。最適cost以上の境界。

### Lower Bound

最適costを超えないことが保証された境界。

### Certified Ratio

`Upper Bound / Lower Bound`。探索中に得られる品質保証。

### Distance Ratio

評価時の `result distance / exact baseline distance`。TRAFFICが事後計算する比較値であり、探索中の証明ではない。

### Quality Certified

指定した品質比以下であることをbound等により証明した状態。

## 3. 探索構造

### Portfolio

一つのRouteRequestを処理するため、TRUSSが運用するANCHORとBOLTSのtask全体。

### SolverTask

TRUSSが作成する一つの探索実行単位。目的、Budget Slice、品質目標等を持つ。

### Lane

独立または半独立に進行できる探索系列。

### Hypothesis

ANCHORが試行する探索方針。単なる推測ではなく、明示的な探索戦略単位。

### Corridor

source-target間の構造的・幾何学的仮説に基づいて探索対象を絞る領域。

### Candidate

最終採用前の完全経路または部分経路。Candidateであることは最適性を意味しない。

### Repair

既存pathの一部を再探索し、より良いsubpathへ置換する処理。

### Fallback

ANCHORによる進行が不十分な場合、TRUSSがBOLTSへ明示的に切り替える処理。ANCHOR内部の別Hypothesisへの切替えはFallbackとは呼ばない。

### Certification

exact solverまたはlower bound providerにより、最適性または品質比を証明する処理。

### Stagnation

一定のWorkまたはStepの間、candidate、bound、frontier進展等に有意な改善がない状態。

## 4. 評価単位

### Scenario

TRAFFICにおける再現可能な評価条件の宣言単位。graph、query、target、budget、seed、repetition、observation、合否基準等を含み得る。

Scenarioは一回の探索実行そのものではない。

### Run

Scenarioから生成される一回の探索実行。評価・記録の最小単位。

### Query

一つのGraphに対するsource、target、目的、制約の組。

### Benchmark Suite

共通目的と基準を持つ複数Scenarioの集合。

### Generator

graphの基本骨格を生成する方法。

### Topology

障害物、袋小路、分断、回り込み等、探索難易度に影響する構造条件。

GeneratorとTopologyは区別する。Generatorは骨格、Topologyはその上に与える構造的難しさである。

## 5. 実行フェーズ

### Phase（フェーズ）

実行過程に含まれる処理を、実運用上の責務とBRIDGEの公開interface境界に基づいて分類した区間。

Phaseは、処理が実装されているコンポーネント、呼び出される関数、または内部的な実行順序だけでは決定しない。処理が外部入力の準備、BRIDGEによる入力処理と結果返却、または返却後の分析・評価のいずれを担うかによって、`Preparing`、`Running`、`Finalizing`のいずれかへ分類する。

Phaseは、処理責務、計測境界、および性能指標の意味を明確にするための分類であり、単なる進捗状態を意味しない。

### Preparing

BRIDGEへ渡す入力を外部で準備するフェーズ。Scenario解釈、Graph生成、Query生成、実行条件の確定を含む。地図アプリケーション、外部script、TRAFFIC等によるGraph生成はPreparingに属する。

### Running

BRIDGEが公開interfaceで入力を受理してから、経路結果および要求されたTrace結果を外部利用可能な形式で返却するまでのフェーズ。入力検証、内部Graph構築、Graph分析、特徴抽出、方針決定、探索、結果構築、Trace生成・直列化・返却を含む。

### Finalizing

BRIDGEが返却した結果を外部で集計、分析、評価、保存または可視化するフェーズ。複数Run集計、HEALTHY評価、性能回帰判定、artifact生成、外部SimulatorによるTrace可視化を含む。

フェーズは処理の実装場所ではなく、実運用上の責務とBRIDGEの公開interface境界によって決定する。

## 6. baseline

### Baseline

TRAFFICによる評価で比較基準として選択された参照結果、アルゴリズム、version、実装の役割名。特定solverの固有属性ではない。

### Exact Baseline

最適costまたは到達不能という正解を提供する参照。

### Performance Baseline

時間、Work、Step、品質曲線等を比較する参照アルゴリズム。

### Regression Baseline

変更前のversion、commit、設定、保存済み結果。

### Reference Implementation

別言語実装や移植元との意味的整合性を確認する参照実装。

## 7. 観測

### BEARING Event

探索層から観測層へ渡す型付き事実。制御指示ではない。

### Observation Mode

観測の詳細度と保存範囲を定める設定。概念上、off、summary、trace、profileを区別する。

### Trace

探索状態の遷移を再構成できるevent列。単なるlogではない。

### Replay

保存済みTraceから探索過程を再構成すること。

### Stable Digest

timing等の非決定的情報を除き、意味的実行結果の同一性を確認するdigest。

## 8. 健全性

### Reported Work

solverまたはportfolioが実行時に報告したWork。

### Reconstructed Work

完全なAction Traceから事後再構成したWork。

### Budget Ledger

TRUSSがtask、component、portfolioの予算割当と消費を記録する台帳。

### Invalid Run

契約違反、artifact破損、trace不完全、予算超過等により、正当な性能比較へ使用できないRun。

---

# BRIDGEの研究・性能評価原則

## 1. 評価の目的

BRIDGEの評価は、単に「速いか」を測るものではない。次を分離して検証する。

- 正しさ
- 初期解性能
- 最終品質
- Anytime改善能力
- Work効率
- Step効率と並列性
- 実行時間
- memory使用量
- 予算遵守
- 再現性
- topologyと規模に対する一般性
- 観測オーバーヘッド
- fallback、repair、certificationの寄与

## 2. 正しさを先に確立する

性能比較へ進む前に、各Runが正当であることを確認する。

- pathがGraph上で連続している
- distance計算が正しい
- 到達可能性判定が一致する
- budget超過がない
- Work保存則が成立する
- artifactが破損していない
- trace完全性が要求水準を満たす
- 実行条件が比較対象間で同一である

Invalid Runを性能集計へ混ぜてはならない。除外した場合は件数と理由を明示する。

## 3. 比較対象

BRIDGE全体、ANCHOR単体、BOLTS solverは別の研究対象として扱う。

### BRIDGE全体

portfolio制御、fallback、certification、overheadを含むシステム性能を測る。

### ANCHOR単体

独自主探索そのものの能力を測る。TRUSSやBOLTSの支援をANCHORの成果として混ぜない。

### BOLTS solver

既存アルゴリズムの実装品質と参照性能を測る。

比較対象には、研究目的に応じてexact、bidirectional、heuristic、weighted、anytime系のアルゴリズムを含める。各アルゴリズムは同じGraph、Query、Work定義、停止条件、観測条件で比較する。

## 4. 測定指標

### 正しさ

- found率
- 到達性一致率
- exact一致率
- path妥当率

### 品質

- Distance Ratio
- worst Distance Ratio
- Certified Ratio
- quality-certified率
- 最終解品質

### 初期解

- First Path Work
- First Path Step
- First Path Time
- First Path Distance Ratio

### Anytime

- 予算ごとのBest Path品質
- 品質改善曲線
- initial-to-final improvement
- area under quality curve
- exactまたは品質証明までのWork、Step、Time

### 探索効率

- Total Work
- Work内訳
- Logical Step
- Scheduled Step
- Parallelism
- Worker utilization

### システム性能

- wall-clock time
- CPU time
- memory peak
- allocation
- GC
- orchestration overhead
- observation overhead

時間は環境依存性が高いため、WorkやStepと併記する。

## 5. 実行フェーズと計測境界

benchmarkは、実運用時のBRIDGE公開interface境界を再現し、処理時間を`Preparing`、`Running`、`Finalizing`へ分離して記録する。

### Preparing

BRIDGE外部で入力を準備する時間。Scenario解釈、Graph生成、Query生成、実行計画を含む。Graph生成時間はbenchmark全体時間には含めてよいが、BRIDGE Running Time、Route Time、Solver Timeには含めない。

### Running

BRIDGEが入力を受理してから、経路結果および要求されたTrace結果を返却するまでの時間。少なくとも次を分離して記録する。

- 入力検証
- 内部Graph構築
- Graph分析・特徴抽出
- Route制御
- Solver実行
- Pathおよび公開Result構築
- Trace取得・直列化・返却

内部Graph構築およびGraph分析はBRIDGEのRunningへ含める。ただし、アルゴリズム性能と混同しないようRoute TimeおよびSolver Timeとは別項目にする。Prepared Graphを利用する試験では、Graph登録・準備とQuery実行を別操作として測定し、Raw Graph試験と混在させない。

### Finalizing

BRIDGE返却後の集計、分析、評価、artifact生成、Trace可視化に要する時間。Run中のTrace取得はRunning、外部Simulatorによる再生・描画はFinalizingとして扱う。

### 正本となる時間

- BRIDGE全体性能: Runningの開始から結果返却まで
- Route性能: 内部Graphが利用可能になってからRoute結果が確定するまで
- Solver性能: 探索algorithm本体の実行時間
- benchmark全体性能: Preparing、Running、Finalizingの総和

これらを一つの「実行時間」として混同してはならない。

## 6. 実験設計

- ScenarioとRunを区別する。
- 複数seedと複数repetitionを使用する。
- warm-upを記録するが、通常の性能集計から分離する。
- 実行順を固定またはseedにより決定論的にrandomizeする。
- graph、query、環境、version、設定のmetadataを保存する。
- raw observationを保存し、平均値だけを成果物としない。
- timeout、budget、worker、observation modeを比較対象間で揃える。
- 同一Run内でexact baselineを探索制御へ利用しない。

## 7. topologyとdataset

一般性を主張するには、複数のgraph familyと難易度条件が必要である。

- openな単純構造
- wallやbarrier
- U字型回り込み
- cul-de-sac
- disconnected
- maze
- community構造
- adversarial構造
- road network等の実dataset
- 規模、密度、weight分布の変化

Generator、Topology、Datasetを区別し、各条件の由来、seed、preprocessing、license、digestを記録する。

## 8. 統計

- raw Run単位の分布を用いる。
- 平均だけでなく中央値、分散、分位点、worst caseを確認する。
- paired designが可能な場合、同一Graph、Query、Seedで対応付ける。
- 外れ値除外は事前規則に基づき、理由と件数を示す。
- 実務上の効果量と統計的不確実性を分けて解釈する。
- 少数Scenarioで一般性能を断定しない。

## 9. 観測モード

純粋な速度比較では観測を最小化する。内部挙動分析ではtraceまたはprofileを使用する。

- off: 探索本体の低オーバーヘッド測定
- summary: 主要指標の再構成
- trace: replay可能な探索過程
- profile: 詳細なボトルネック分析

mode間で意味的結果が一致することを検証する。profile結果をoffの速度として報告しない。

## 10. アブレーション

BRIDGE固有機構の有用性は、機構を一つずつ無効化して検証する。

- fallbackなし
- certificationなし
- repairなし
- 特定Hypothesisなし
- 特定probeなし
- 予算再配分なし

アブレーションは、異なるコードパスを新規実装するのではなく、同じ実装の明示的な機能切替えとして行う。

## 11. 評価上の禁止事項

- 成功Runだけを選んで集計する。
- BRIDGEだけに有利な停止条件を使う。
- アルゴリズムごとにWork定義を変える。
- exact baseline計算時間を恣意的に含めたり除外したりする。
- trace I/Oを一部アルゴリズムだけに課す。
- 特定seedの結果だけで結論づける。
- ANCHOR単体結果とBRIDGE全体結果を混同する。
- fallbackで得た結果をANCHORのpure performanceとして扱う。
- wall-clock timeだけでアルゴリズムの本質的性能を断定する。

## 12. 研究上の主張に必要な証拠

### 「速い」

同一環境、同一条件、複数Runのtime分布が必要である。

### 「探索効率が高い」

共通Work定義によるTotal Workと内訳が必要である。

### 「並列性が高い」

Logical Step、Scheduled Step、Worker数、schedule規則が必要である。

### 「高品質」

Exact Baselineに対するDistance RatioまたはCertified Ratioが必要である。

### 「Anytime」

複数予算点における品質改善曲線が必要である。

### 「汎用的」

複数Generator、Topology、Dataset、規模、seedで一貫した結果が必要である。

### 「再現可能」

Scenario、Run metadata、seed、raw observation、Stable Digestが必要である。

---

# BRIDGE変更時の判断ガイド

## 1. 変更前に確認する問い

新機能、最適化、リファクタリング、用語変更を行う前に、次を確認する。

1. この変更の目的は何か。
2. どのコンポーネントが責務を所有するべきか。
3. 既存の責務境界を越えていないか。
4. portfolio予算の所有権を侵害しないか。
5. WorkとStepの意味を変えないか。
6. 観測有無で結果が変わらないか。
7. 決定論性を維持できるか。
8. 公開契約、schema、用語集へ影響するか。
9. 製品経路とbenchmark経路を分岐させていないか。
10. 正しさと性能をどのテストで証明するか。

## 2. 責務配置の判断

### GATEへ置く

外部入力検証、外部ID変換、公開API表現、error mappingである場合。

### TRUSSへ置く

solver選択、task生成、予算配分、継続、fallback、certification、最終候補選択である場合。

### ANCHORへ置く

BRIDGE固有の主探索Hypothesis、Candidate生成、局所Repairである場合。

### BOLTSへ置く

交換可能な補助solver、exact探索、reachability、lower bound、certification能力である場合。

### BEARINGへ置く

探索から観測へ渡す中立的なtyped event契約である場合。

### ULTRASOUNDへ置く

event収集、保存、replay、metrics、profile等の観測処理である場合。

### TRAFFICへ置く

Scenario、Run、graph/query生成、benchmark、比較、統計、acceptanceである場合。

### HEALTHYへ置く

生成済みartifactのread-onlyな整合性、Work、Ledger、trace再構成検証である場合。

### COREへ置く

複数コンポーネントが共有し、特定アルゴリズムや外部I/Oへ依存しない中立契約である場合。

## 3. 変更が拒否される条件

次のいずれかに該当する変更は、そのまま採用しない。

- ANCHORがBOLTSを直接起動する。
- BOLTSがportfolio全体の切替えを決める。
- GATEがsolver内部を選択または操作する。
- ULTRASOUNDのeventが探索判断へ帰還する。
- TRAFFICがprivate APIを通じて有利な状態を注入する。
- benchmark専用の別アルゴリズム経路を作る。
- Work未計上の探索Actionを増やす。
- observer有効時だけ結果が改善または悪化する。
- exactでない結果をexactと表記する。
- baselineを通常探索へ入力する。
- 失敗Runを理由なく除外する。
- 未定義用語を公開契約へ追加する。

## 4. アルゴリズム追加

新しいsolverをBOLTSへ追加する場合、少なくとも次を満たす。

- 共通入力・出力契約を使用する。
- Budget Sliceを超えない。
- 共通Work Actionで計測する。
- Step計測規則を明示する。
- cancellationとdeadlineを扱う。
- tie-breakingを決定論的にする。
- path妥当性を検証する。
- observer非干渉を確認する。
- solver固有のprivate metricを共通Workへ混ぜない。
- TRAFFICから同一条件で比較できる。

ANCHORへ新しいHypothesisを追加する場合、さらに次を確認する。

- 主探索の目的に適合する。
- 別solverの隠れた呼出しになっていない。
- Candidate生成とportfolio判断を混同しない。
- 既存Hypothesisとの重複Workを測定できる。
- アブレーション可能である。

## 5. 観測項目追加

新しいeventやmetricを追加する場合、次を確認する。

- 何を表すかが明確である。
- 所有componentが明確である。
- 単位と集計範囲が明確である。
- Workに含むか含まないかが明確である。
- modeごとの取得可否が明確である。
- replayに必要か、profile専用かを区別する。
- schema versionと互換性を検討する。
- event欠損時の扱いを定義する。
- 取得の有無で探索結果が変わらない。

## 6. 用語変更

用語変更時は、次を同時に確認する。

- 正式定義
- 含むものと含まないもの
- 所有component
- 対応するfield名
- 類似語との差異
- schema
- code identifier
- Scenario
- trace
- test
- 既存artifact互換性

既存語の意味を静かに変更しない。破壊的変更であれば、versionまたはmigration方針を明示する。

## 7. 性能最適化

最適化は、意味を変えずに測定可能な改善として行う。

- 最適化前後でpath、distance、状態、Work定義を比較する。
- wall-clock timeだけでなくallocation、memory、Work、Stepを測る。
- mapからslice、object allocation削減、workspace再利用等は、決定論性と安全性を確認する。
- 並列化はWork削減と混同せず、Step削減として評価する。
- instrumentation削減は観測modeの契約内で行う。
- 最適化のためにprivate stateを他componentへ漏らさない。

## 8. 文書更新

変更内容に応じて更新対象を選ぶ。

### 恒久原則が変わる

- 最上位アーキテクチャ規則
- 本知識ベース
- 関連コンポーネント規則
- 用語集

### 公開契約が変わる

- schema
- API仕様
- 用語集
- 互換性試験
- migration情報

### 実装だけが変わる

- code
- test
- CHANGELOG
- 必要に応じて実装報告

### 性能値が変わる

- benchmark artifact
- 評価報告

性能値を本知識ベースへ固定値として書かない。

## 9. 完了判定

変更は、実装が存在するだけでは完了しない。

- 責務境界が正しい
- 依存方向が正しい
- public contractが整合する
- 用語が定義されている
- testが追加されている
- budget違反がない
- Work保存則が成立する
- observer非干渉が成立する
- 決定論性が成立する
- 既存機能に回帰がない
- 文書とコードが一致する

これらを確認して初めて、BRIDGEとして安全に変更されたとみなす。
