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
