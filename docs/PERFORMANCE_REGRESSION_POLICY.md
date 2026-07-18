# 性能回帰運用基準

## 1. 位置付け

本書は、BRIDGE v0.15系列における性能回帰試験の中期運用ルールを定める。

本ルールは現時点の実装規模、CI所要時間、実測結果に基づく運用上の正本であり、アーキテクチャ上の永久不変条件ではない。ただし、正式な見直しを行うまでは、本書に定める実行区分と判定基準を使用する。

性能基準の更新によって既存の退行を隠してはならない。閾値、Scenario、Baselineを変更する場合は、変更理由、変更前後の実測結果、影響範囲を監査記録へ残す。

## 2. 基本方針

- Smoke Benchmarkは機能確認専用とし、性能回帰判定には使用しない。
- Mediumを通常開発における性能回帰の正本とする。
- LargeをRelease Candidateにおけるスケーラビリティ監査とする。
- Smallは短時間の補助検査として使用し、Mediumの代替にはしない。
- 性能判定は、Baselineと環境フィンガープリントが一致する場合に限って自動判定する。
- Work、経路成立、最適性主張などの意味論基準は、時間計測より厳格に扱う。

## 3. 階層と責務

| 階層 | 正規Scenario | 規模 | 運用上の責務 | 測定数/組合せ |
|---|---|---:|---|---:|
| Small | `benchmark-regression-small-v1.json` | 400〜500 | Pull Requestで利用可能な補助検査。固定オーバーヘッドや明白な退行の早期検出 | 60 |
| Medium | `benchmark-regression-medium-v1.json` | 1,000 | 通常の性能回帰正本。Main統合、性能関連変更、通常リリース判定の基準 | 10 |
| Large | `benchmark-regression-large-v1.json` | 10,000 | Release Candidate監査。スケーラビリティ、メモリ、トポロジー依存の弱点を確認 | 3 |

MediumとLargeの測定数は現段階の中期基準である。測定時間、分散、CI資源を継続観察し、必要な場合に正式な見直し手順を経て変更する。

## 4. 運用ゲート

### 4.1 通常のPull Request

- Go、SDK、契約テストを必須とする。
- 性能に影響しない文書修正や明白に非実行系の変更では、性能回帰試験を省略できる。
- 探索、計測、データ構造、メモリ確保、シリアライズ、オーケストレーションに影響する変更では、SmallまたはMediumを実行する。
- Smallを実行した場合でも、性能影響が疑われる変更をMainへ統合する前にはMediumを実行する。

### 4.2 Main統合・通常リリース

Mediumを必須の性能回帰ゲートとする。

次をすべて満たさなければならない。

- Medium Scenarioが完走する。
- Baselineとの環境条件が一致する。
- 共通意味論基準を満たす。
- Medium用許容幅を超える退行がない。
- 失敗をBaseline更新だけで解消していない。

### 4.3 Release Candidate

Mediumに加えてLargeを必須とする。

Largeは次を確認する監査である。

- 10,000ノード規模で完走する。
- メモリ使用量と実行時間に重大な非線形退行がない。
- GridとMazeで既知の性能特性が維持される。
- BRIDGEの強い条件と弱い条件の双方で意味論が維持される。
- Large用許容幅を超える退行がない。

Largeの失敗は、通常のMain開発を常に停止するものではない。ただし、Release Candidateの承認条件を満たさないため、原因の解明、修正、または明示的なリリース判断が必要である。

## 5. Scenario構成

### 5.1 Small

- Grid Open: 400ノード
- Random Geometric: 500ノード
- Seed: 1, 7, 19
- Warmup: 3
- Repetition: 20

### 5.2 Medium

- Grid Open: 1,000ノード
- Random Geometric: 1,000ノード
- Seed: 7, 19
- Warmup: 1
- Repetition: 5

### 5.3 Large

- Grid Open: 10,000ノード
- Maze: 10,000ノード
- Seed: 7
- Warmup: 1
- Repetition: 3

Random Geometric 10,000ノードは、現行Generatorの生成コストが探索計測を支配するため、正規Largeから除外する。Generator改善後に再評価する。

## 6. 共通意味論基準

- Path Found Rate: 100%
- Dijkstra・A*: Optimality Proven Rate 100%
- BRIDGE: 根拠のないOptimality Proven Rate 0%
- Work p50: Baselineから増加不可

意味論基準の失敗は、時間やAllocationが改善していても許容しない。

## 7. 性能許容幅

| 階層 | Time p50 | Time p95 | Allocation p50 | Allocation p95 |
|---|---:|---:|---:|---:|
| Small | +35% | +50% | +15% | +20% |
| Medium | +20% | +30% | +10% | +15% |
| Large | +15% | +20% | +10% | +15% |

Mediumを通常判定の正本とするため、性能改善・退行の通常判断ではMediumの結果を優先する。Smallの成功をもってMediumの失敗を上書きしてはならない。

Largeはスケーラビリティ監査であり、Mediumより厳しい時間許容幅を用いる。Largeでのみ発生する退行は、規模依存の問題として別途扱う。

## 8. 環境条件

以下がBaselineと一致しない場合は比較を失敗とする。

- Go Version
- GOOS
- GOARCH
- 論理CPU数

異なる環境では、その環境専用のBaselineを作成する。異なるCPU、OS、Go版の絶対時間を直接比較しない。

## 9. Baseline更新ルール

Baselineは次の場合に限り更新できる。

- 意図した性能改善が確認された場合
- Go、OS、CPUなど実行環境を正式に変更した場合
- Scenarioの正当な見直しを承認した場合
- 計測方法の欠陥を修正した場合

Baselineを更新する前に、旧Baselineに対する結果を保存し、退行が存在しないこと、または退行を受容する理由を記録する。

次の目的でBaselineを更新してはならない。

- 原因不明の失敗を通すため
- 一時的な環境負荷を吸収するため
- Work増加や意味論不整合を隠すため
- Release Candidate監査を形式的に通過させるため

## 10. 中期ルールの見直し条件

本ルールは、次のいずれかが発生した場合に見直す。

- Mediumの実行時間が通常CIに対して過大または過小になった場合
- Largeの規模が実運用・研究対象を代表しなくなった場合
- BRIDGEの対象グラフ特性が大きく変わった場合
- Generatorの性能改善により新しいLarge Scenarioが実用化した場合
- 計測分散が現在の閾値と整合しなくなった場合
- Goランタイム、CI環境、対応プラットフォームを大きく変更した場合
- v0.16以降で探索・観測・Artifact契約に重大な変更が入った場合

見直し時は、少なくとも以下を実施する。

1. 旧ルールによる最終Baselineを保存する。
2. 新旧Scenarioを同一環境で実行する。
3. 閾値の根拠となる分散と実行時間を記録する。
4. 現行監査文書を更新する。
5. 変更をバージョン管理された文書として残す。

## 11. 操作

### Small補助検査

````bash
bridge benchmark run tests/examples/benchmark-regression-small-v1.json
python tasks/performance_regression.py check \
  tests/performance/baselines/v0.15.3-small-linux-amd64-go1.23.2.json \
  <artifact-dir>
````

### Medium通常ゲート

````bash
bridge benchmark run tests/examples/benchmark-regression-medium-v1.json
python tasks/performance_regression.py check \
  tests/performance/baselines/v0.15.3-medium-linux-amd64-go1.23.2.json \
  <artifact-dir>
````

### Large Release Candidate監査

````bash
bridge benchmark run tests/examples/benchmark-regression-large-v1.json
python tasks/performance_regression.py check \
  tests/performance/baselines/v0.15.3-large-linux-amd64-go1.23.2.json \
  <artifact-dir>
````

Baseline作成時は`create-baseline`と`--tier small|medium|large`を使用する。

## 12. 性能分析の必須構造

性能試験の結果は、単純な速度比較ではなく、次の順序で分析する。

1. 正当性
2. Work効率
3. 実時間性能
4. ボトルネック
5. スケーラビリティ
6. リリース判定

前段の検査が失敗した場合、後段の性能比較を有効な評価として扱ってはならない。特に、経路不正、根拠のない最適性主張、Run欠損がある結果について、時間改善を成果として採用してはならない。

## 13. 正当性分析

性能分析の前提として、Scenario・Algorithm・Seed単位で以下を確認する。

- Path Found Rate
- 経路の連続性と辺の存在
- 経路コストと辺重み合計の一致
- Dijkstra・A*の最適性証明率
- BRIDGEの最適性主張の妥当性
- 到達不能判定
- Budget終了、Timeout、Cancel、Errorの混入
- 無効Run数

正当性に失敗したRunは性能統計から黙って除外してはならない。除外する場合は件数、理由、影響範囲を報告し、通常の回帰判定は失敗として扱う。

## 14. Work効率分析

Workは環境ノイズの影響を受けにくい中核指標として扱う。以下をScenario・Algorithm単位で分析する。

- Work p50 / p95
- Work / node
- Work / edge
- Work / path length
- BRIDGE Work / Dijkstra Work
- BRIDGE Work / A* Work
- Medium / SmallのWork増加率
- Large / MediumのWork増加率

全Scenarioを単純平均して、トポロジーごとの差を隠してはならない。Grid、Random Geometric、Mazeは個別に評価する。

Workが増加している場合は、時間が改善していてもアルゴリズム上の退行候補として原因を調査する。

## 15. 実時間分析

実時間は平均値だけで判断せず、以下を使用する。

- Solver Time p50 / p95
- End-to-End Time p50 / p95
- 初回解時間
- 最終解時間
- Time / Work
- Time / node

次の差分を周辺処理の概算値として確認する。

```text
Non-Solver Overhead = End-to-End Time - Solver Time
```

p50は通常時の代表性能、p95は尾部遅延と不安定性の指標として扱う。p50が改善していてもp95が悪化している場合は、GC、Allocation、特定Seed、Handoff変動を調査する。

## 16. ボトルネック分析

### 16.1 位置付け

ボトルネック分析は、回帰判定とは独立した必須分析である。

- 回帰判定は「基準を満たすか」を判断する。
- ボトルネック分析は「時間・メモリ・Workを支配する箇所」を特定する。

性能変更、Medium回帰失敗、Large監査失敗、原因不明のp95悪化がある場合は、ボトルネック分析を省略してはならない。

### 16.2 分離すべき処理領域

可能な範囲で、少なくとも以下を分離する。

- Graph Generation
- Input Validation
- GATE変換
- TRUSS Orchestration
- ANCHOR探索
- BOLTS呼び出し
- Handoff
- ULTRASOUND Observation
- HEALTHY Evaluation
- Artifact書き込み
- JSON Serialization
- ZIP Compression
- SDK・HTTP通信

Benchmark全体時間、経路探索時間、Artifact生成時間を同一指標として扱ってはならない。Graph GenerationはSetup参考指標であり、通常のBenchmark Run経過時間、End-to-End時間、性能回帰判定へ直接含めてはならない。

### 16.3 必須時間指標

利用可能な計測値から、以下を収集・報告する。

- total_time
- graph_generation_time（Setup参考指標。通常の回帰判定対象外）
- validation_time
- solver_time
- orchestration_time
- observation_time
- artifact_write_time
- evaluation_time

実装が計測可能な場合は、以下も分離する。

- anchor_time
- bolts_time
- handoff_time
- serialization_time
- compression_time

### 16.4 必須比率

```text
Solver Ratio = Solver Time / End-to-End Time
Orchestration Ratio = Orchestration Time / End-to-End Time
Observation Ratio = Observation Time / End-to-End Time
Artifact Ratio = Artifact Write Time / Benchmark Total Time
```

比率はSmall・Medium・Large、およびトポロジーごとに比較する。

### 16.5 CPU分析

CPU時間の支配要因を、Go Benchmarkまたは開発用Profileで確認する。

主な確認対象は以下とする。

- Priority Queue操作
- Map・Sliceアクセス
- Heuristic計算
- 経路復元
- Handoff判断
- Evidence生成
- JSON変換
- 圧縮処理

Profileを取得した場合は、CPU上位関数、累積比率、Scenario、規模、取得条件を監査記録へ残す。

### 16.6 メモリ・GC分析

以下を確認する。

- Allocation Count
- Allocation Bytes
- Peak Heap
- Live Objects
- GC回数
- GC Pause
- 一時Slice・Mapの再確保
- 経路・Evidence・Traceのコピー

実時間の改善だけを理由に、AllocationまたはPeak Heapの重大な悪化を許容してはならない。

### 16.7 BRIDGE固有の制御分析

以下をScenario・規模単位で確認する。

- Epoch数
- Handoff数
- HandoffあたりWork
- Handoffあたり時間
- 解改善を伴わないHandoff数
- Lower Bound更新回数
- Evidence生成回数
- 同一候補の再評価回数

WorkがDijkstraと同程度で、HandoffやEpochだけが多く、実時間が悪化する場合は、探索器ではなく制御戦略を主要ボトルネック候補とする。

### 16.8 ボトルネックの分類

分析結果は、少なくとも以下のいずれかに分類する。

- Algorithmic Work
- Orchestration
- Allocation / GC
- Observation
- Serialization / Compression
- Graph Generation
- Environment Noise
- Unknown

`Unknown`とした場合は、根拠なくBaselineを更新してはならない。

## 17. スケーラビリティ分析

Largeでは絶対時間だけでなく、Mediumからの増加率を分析する。

```text
Time Scaling = Large Time / Medium Time
Work Scaling = Large Work / Medium Work
Memory Scaling = Large Allocation / Medium Allocation
```

以下の正規化指標を併記する。

- Work / node
- Time / node
- Allocation Bytes / node

Mediumは通常回帰の合否判定に用い、Largeは以下を確認する。

- ノード数増加に対する非線形なWork増加
- Work増加を超える時間増加
- メモリの非線形増加
- GridとMazeでの支配要因の変化
- BRIDGEの強い条件と弱い条件の維持

LargeでBRIDGEが全アルゴリズムより高速であることは要求しない。既知の弱点が悪化していないこと、強みが維持されていること、増加傾向が説明可能であることを要求する。

## 18. Medium分析ルール

Medium結果から、最低限以下を報告する。

- Baseline比較結果
- Scenario・Algorithm別のp50 / p95
- Work回帰
- 解品質と正当性
- Allocation回帰
- Seed別外れ値
- ボトルネック分類
- Pass / Fail
- Baseline更新可否

一部Scenarioの退行を、全体平均の改善で相殺してはならない。

## 19. Large分析ルール

Release CandidateのLarge監査では、最低限以下を報告する。

- 10,000ノード完走性
- Grid・Maze別のWork、時間、メモリ
- MediumからLargeへの増加率
- p95悪化
- Epoch・Handoff特性
- 支配的コンポーネント
- 既知制約と新規退行の区別
- Release Candidate承認可否

Large失敗を受容してリリースする場合は、原因、影響範囲、回避策、次期修正計画を明示的に記録する。

## 20. 性能分析レポートの必須構成

正式な性能分析レポートは、以下の章を持つ。

1. 実行条件
2. 正当性
3. Work効率
4. Medium回帰判定
5. Largeスケーラビリティ監査
6. ボトルネック分析
7. アルゴリズム・トポロジー別評価
8. 原因分類
9. Baseline更新判断
10. Release判断

実行条件には、少なくともVersion、Commit、Go Version、GOOS、GOARCH、CPU、Scenario、Seed、Warmup、Repetition、Observation Levelを記載する。

## 21. 禁止する分析方法

- 平均値だけで合否を判断する。
- 全Scenarioを一括平均して個別退行を隠す。
- Workを確認せず時間だけを比較する。
- Smallだけで正式な性能改善を主張する。
- 異なる環境の絶対時間を直接比較する。
- Generator時間とSolver時間を混同する。
- 解品質悪化を速度改善として許容する。
- p95悪化を無視してp50だけを採用する。
- 原因不明の失敗を閾値またはBaseline更新で解消する。
- BRIDGEがDijkstraより遅いことだけを不具合と断定する。

## 22. 分析ルールの見直し

本分析ルールも中期運用ルールであり、永久不変ではない。次の場合に見直す。

- 新しい計測区間が正式に追加された場合
- Artifact Schemaへ性能内訳が追加された場合
- Medium・Largeの規模またはトポロジーを変更した場合
- Profile取得方式を標準化した場合
- 現行指標では原因特定できない退行が継続した場合

見直しでは、旧ルールによる分析結果と新ルールによる分析結果を比較し、分析精度が低下していないことを確認する。
