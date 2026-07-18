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
