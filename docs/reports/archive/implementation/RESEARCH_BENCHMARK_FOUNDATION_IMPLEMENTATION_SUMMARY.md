# BRIDGE v0.14.1 研究ベンチマーク基盤化 改修総まとめ

## 1. 文書の目的

本書は、`V0.14.1_RESEARCH_BENCHMARK_FOUNDATION_PLAN.md`で定義された研究ベンチマーク基盤化計画について、Phase 1からPhase 5までに実装された変更を総括するものである。

特に、各変更について次を明確にする。

- なぜ変更が必要だったのか
- 何を実装したのか
- 何が解決されたのか
- どのコンポーネントへ影響したのか
- どの契約・schema・文書・テストが更新されたのか

本計画の目的は、論文執筆用の完成済み研究プロトコルを提供することではない。BRIDGE、ANCHOR、BOLTSおよび既存探索アルゴリズムを、同一条件で繰り返し実行し、取得したraw dataから性能差、品質差、Work差、内部挙動差を客観的に確認できる開発・評価基盤を構築することである。

---

# 2. 改修前に存在した主要な問題

改修前にも、BRIDGEにはbenchmark実行機能とULTRASOUNDによる観測機能が存在していた。しかし、研究・開発評価の基盤としては次の不足があった。

## 2.1 実験単位が曖昧だった

- seed、query、repetition、warm-upの区別がartifact上で十分に保持されていなかった。
- Scenarioと個々の実行Runの境界が曖昧だった。
- 平均値中心で、後から再集計できるraw observationsが不足していた。
- 実行順randomizationを再現するためのmanifestがなかった。

このため、同じ条件で再実行したか、異なるqueryが平均へ混在していないか、warm-upが集計へ入っていないかを確認しにくかった。

## 2.2 Workと時間の内訳が不足していた

- Total Workだけでは、どの操作が増減したか分からなかった。
- BRIDGE固有のorchestration、fallback、certification等の時間をsolver時間から分離できなかった。
- allocation、malloc、GC、heap等のruntime指標を標準artifactへ保存できなかった。

このため、アルゴリズム改善なのか、実装オーバーヘッド増加なのかを切り分けられなかった。

## 2.3 観測モードの意味が統一されていなかった

- `metrics`、`debug`等の旧modeが混在していた。
- summary、trace、profileの保存対象とオーバーヘッドの境界が曖昧だった。
- 観測によって探索結果が変化しないことをBRIDGE全体で検証できていなかった。
- traceのsampling、欠落、truncation、checksum等をmanifestへ十分に保存していなかった。

このため、traceを取得した結果だけが遅くなったのか、観測自体が探索挙動を変えたのか判断しにくかった。

## 2.4 Anytime挙動と失敗理由を説明できなかった

- 最終距離だけでは、最初の経路をいつ見つけたか、何回改善したか分からなかった。
- `Found=false`だけでは、非連結、予算切れ、timeout、fallback失敗等を区別できなかった。
- fallbackやcertificationの効果を同一Scenario系列で比較する設定が不足していた。

このため、ANCHORとBRIDGEが「どのように」結果へ到達したかを説明できなかった。

## 2.5 graph familyとdatasetの一般性が不足していた

- gridやrandom geometricだけでは、特定topologyに依存した結論となる可能性があった。
- 実データを再現可能な契約で取り込む仕組みがなかった。
- datasetの出典、ライセンス、SHA-256、前処理履歴をraw runへ残せなかった。
- raw observationsから統計比較を再生成する標準scriptがなかった。

このため、異なるgraph条件へ結果が一般化するかを検証しにくかった。

---

# 3. 改修後の全体構造

Phase 1からPhase 5の実装後、研究ベンチマーク基盤は次の責務分担となった。

| コンポーネント | 研究ベンチマーク基盤での責務 |
|---|---|
| TRAFFIC | Scenario展開、Run生成、実験実行、metadata生成、raw result保存、summary集計 |
| ULTRASOUND | mode別観測、summary、trace、profile、replay用artifact生成 |
| BEARING | canonical typed event vocabularyとevent classの定義 |
| CORE | Work、時間、system metrics、ablation等の共有型と公開契約 |
| GATE | 公開API境界と結果整形。研究上の合否判断は行わない |
| TRUSS | BRIDGE全体の実行制御、fallback、certification、orchestrationの測定事実を提供 |
| ANCHOR | 中核探索とanytime改善履歴の測定事実を提供 |
| BOLTS | 比較・fallback・certification用solverの測定事実を提供 |
| compatibility scripts | raw dataからの研究準備性評価、統計report再生成 |

基本フローは次のとおりである。

```text
Scenario
   ↓
TRAFFIC
   ↓
GATE.Route / GATE.ExecuteOnce
   ↓
TRUSS / ANCHOR / BOLTS
   ↓
ULTRASOUNDによる観測
   ↓
Benchmark Raw Result
   ↓
再集計・統計比較・研究準備性評価
```

TRAFFICはsolver private stateを直接参照せず、GATE公開API、CORE公開schema、ULTRASOUND公開artifact APIを通じて実験を構成する。

---

# 4. Phase 1: 実験単位と集計単位の標準化

## 4.1 なぜ必要だったのか

従来のbenchmarkは平均値中心であり、どのseed、query、repetitionから値が生成されたかを後から完全に追跡しにくかった。

研究・開発評価では、平均値だけでは次を確認できない。

- 特定queryだけで性能が悪化していないか
- seed依存の偶然な改善ではないか
- warm-upが測定結果へ混入していないか
- 同じRun順序を再現できるか
- summary値をraw dataから再計算できるか

そのため、Scenarioを宣言単位、Runを記録単位として分離する必要があった。

## 4.2 実装内容

### Scenario拡張

Scenarioへ次を追加した。

- `warmup_runs`
- 複数`queries`
- `randomize_order`
- `artifact_id`
- environment capture
- output metadata

旧`endpoints`形式は、`query_id=default`として正規化し、既存Scenarioとの互換性を維持した。

### Run識別子

Runを次の組で一意化した。

```text
scenario_id
× algorithm
× graph_instance_id
× query_id
× repetition
```

各raw runへ次を保存した。

- Run ID
- run ordinal
- seed
- query ID
- repetition
- warm-up状態
- algorithm
- graph instance ID

### raw observations

各Runの結果をトップレベル`raw_runs`へ保存した。

平均値だけでなく、後からraw observationsを読み取り、別の集計規則で再分析できるようになった。

### metadata

TRAFFICで次のmetadataを生成した。

- graph metadata
- query metadata
- quality metadata
- environment metadata
- output metadata

solver本体へ評価専用metadata計算を持ち込まない構造を維持した。

### summary statistics

次を標準集計として追加した。

- count
- mean
- sample standard deviation
- min
- p50
- p95
- max
- 95% confidence interval

### query別summary

集計単位を次へ変更した。

```text
Scenario × Algorithm × Query
```

異なるquery難易度が一つの平均へ混ざる問題を解消した。

### execution manifest

randomization後の実行順について次を保存した。

- randomizeの有無
- shuffle seed
- shuffle algorithm
- 展開後Run IDの実行順

## 4.3 解決されたこと

- 各測定値の由来をRun単位で追跡できるようになった。
- warm-upを保存しつつ、summaryとacceptanceから除外できるようになった。
- query別の性能差を分析できるようになった。
- raw observationsからsummaryを再計算できるようになった。
- 実行順による偏りを確認・再現できるようになった。
- 平均値だけでは隠れるp95やworstの悪化を確認できるようになった。

## 4.4 影響範囲

### 主なコンポーネント

- TRAFFIC
- COREのbenchmark artifact型
- GATEの公開結果整形

### 主な契約・文書

- `docs/architecture/BENCHMARK_SCENARIO_SPEC_v1.md`
- `benchmark-scenario-v1.schema.json`
- `benchmark-result-v1.schema.json`
- `docs/WORD_DEFINITION.md`

### 主なテスト

- Scenario validation
- Run ID一意性
- warm-up除外
- query別集計
- raw resultからのsummary再計算
- execution manifest一致
- deterministic randomization

---

# 5. Phase 2: Work内訳・時間内訳・system metrics

## 5.1 なぜ必要だったのか

Total Workと平均実行時間だけでは、性能差の原因を説明できない。

例えば実行時間が増加した場合でも、次のどれが原因か区別できなかった。

- ANCHOR探索量が増えた
- BOLTS fallbackが増えた
- TRUSS orchestrationが増えた
- GATE変換処理が増えた
- allocationやGCが増えた
- 観測I/Oが増えた

そのため、探索Work、phase別時間、system metricsを別々の型付き契約として記録する必要があった。

## 5.2 実装内容

### WorkMetrics

COREの型付き契約としてWork内訳を公開した。

主要な対応は次のとおりである。

| 研究上の指標 | WorkMetrics上の位置づけ |
|---|---|
| node expansion | Expand Action |
| edge scan | Evaluate Action |
| relaxation attempt | Relax Action |
| queue insertion | Enqueue Action |

時間計測、runtime memory取得、serialization、artifact保存、trace I/OはWorkへ含めない。

### TimeBreakdown

次の時間内訳を型付きで追加した。

- total time
- solver time
- TRUSS time
- ANCHOR time
- BOLTS time
- fallback time
- Supervisor time
- Arbiter time
- orchestration time
- GATE time

TRUSS、ANCHOR、BOLTS側が測定事実を出し、TRAFFICが比較可能なartifactへ保存する構造とした。

### SystemMetrics

各Runの前後でGo runtime metricsを取得し、次を保存した。

- alloc bytes
- malloc count
- GC count
- heap alloc before
- heap alloc after
- heap boundary max
- profile時のheap sampled peak

`heap_alloc_boundary_max`と`heap_alloc_sampled_peak`を分離し、厳密な瞬間peakであるかのような誤解を防止した。

### summary拡張

次についても標準summary statisticsを生成するようにした。

- Expand、Evaluate、Relax、Enqueue等のWork内訳
- ANCHOR、BOLTS、fallback等のphase時間
- alloc bytes
- malloc count
- GC count

## 5.3 解決されたこと

- WorkがどのActionで増減したか分析できるようになった。
- BRIDGE固有のorchestration時間をsolver時間から分離できるようになった。
- fallbackやcertificationの追加コストを把握できるようになった。
- 速度改善とallocation増加を同時に確認できるようになった。
- 観測I/Oやserializationを探索Workへ混入させずに済むようになった。
- 実装最適化とアルゴリズム上のWork削減を区別しやすくなった。

## 5.4 影響範囲

### 主なコンポーネント

- CORE
- TRUSS
- ANCHOR
- BOLTS
- GATE
- TRAFFIC

### 主な契約・文書

- Work定義
- TimeBreakdown契約
- SystemMetrics契約
- benchmark result schema
- 各component rule

### 主なテスト

- serializationとzero value
- phase時間の非負性
- Workと非探索overheadの分離
- memory metrics保存
- summary statistics再計算

---

# 6. Phase 3: ULTRASOUND mode再設計と内部挙動trace

## 6.1 なぜ必要だったのか

観測には、軽量な集計だけが必要な場合と、詳細な状態遷移を保存したい場合がある。

旧実装ではmodeの意味が重複し、次の問題があった。

- summaryでも不要なI/Oが発生する可能性がある。
- traceとprofileの保存対象が曖昧だった。
- sampling設定がschemaに存在しても実動作へ反映されていない箇所があった。
- 観測modeによって探索結果が変化しないことを十分に検証できなかった。
- Collectorと旧Recorderの二重実装が存在した。

そのため、観測契約を単純化し、非干渉性とartifact完全性を検証可能にする必要があった。

## 6.2 実装内容

### Observation Mode

現行modeを次の4つへ固定した。

| Mode | 動作 |
|---|---|
| `off` | observerを接続しない |
| `summary` | control・candidate eventをメモリ集計し、event streamを保存しない |
| `trace` | control・candidate・detail eventを保存する |
| `profile` | trace対象に加え、高頻度Action eventとprofile計測を保存する |

旧`metrics`、`debug`は現行公開契約から削除した。

### BEARING event vocabulary

BEARINGへcanonical event vocabularyとevent classを追加した。

代表的なeventは次のとおりである。

- search start／finish
- component start／finish
- frontier selection
- node expansion
- edge evaluation
- relaxation
- budget extension
- candidate submission
- incumbent update
- fallback start／finish
- certification start／finish
- state reuse用予約event
- profile Action event

BEARINGは語彙と型のみを所有し、保存・集計・制御判断を持たない。

### ULTRASOUND Collector

Collectorを唯一の現行観測実装へ統一した。

旧Recorderは現行packageから削除し、legacyへ移行した。

Collectorへ次を追加した。

- mode別event選別
- event kind別count
- phase別count
- dropped event count
- truncation状態
- observer overhead
- sink write time
- quality history
- budget history

### deterministic sampling

`sample_rate`を実際のevent採否へ適用した。

固定seed、event ordinal、event kindから決定論的に採否を決めるため、同一条件では同一event集合を再現できる。

### trace manifest

manifestへ次を保存した。

- mode
- sample rate
- sampling algorithm
- event count
- dropped event count
- truncation
- observer overhead
- sink write time
- Stable Digest
- trace SHA-256
- trace file名

### Observation Resultのraw run統合

TRAFFICのraw runへ次を保存した。

- observation mode
- event count
- dropped count
- truncation
- overhead
- sink write time
- summary
- observation overhead ratio

### Stable Digest非干渉

BRIDGE全体について、`off`、`summary`、`trace`、`profile`でStable Digestが一致するテストを追加した。

## 6.3 解決されたこと

- 軽量なsummaryと詳細traceを使い分けられるようになった。
- summaryではtrace I/Oを発生させずに主要counterを取得できるようになった。
- traceからquality-time／quality-work履歴を再構成できるようになった。
- profileだけが高頻度Action eventを保存する構造になった。
- 観測による探索結果への干渉を検出できるようになった。
- trace欠落、truncation、改変、sampling条件をartifactから確認できるようになった。
- 現行観測経路がCollectorへ一本化され、二重実装によるmode不整合が解消された。

## 6.4 影響範囲

### 主なコンポーネント

- BEARING
- ULTRASOUND
- TRUSS
- ANCHOR
- BOLTS
- TRAFFIC
- GATE

### 主な契約・文書

- ULTRASOUND trace data contract
- trace event schema
- trace manifest schema
- Route Request／Result schema
- component rules
- 用語集

### 主なテスト

- mode別event保存
- summary非I/O
- deterministic sampling
- Stable Digest一致
- replay互換
- trace checksum
- legacy Recorder非使用

---

# 7. Phase 4: Anytime品質評価・失敗分類・アブレーション

## 7.1 なぜ必要だったのか

ANCHORとBRIDGEはAnytime探索を志向しているため、最終距離だけでは性能を十分に表現できない。

必要なのは、次のような進行過程である。

- 最初の有効経路をいつ見つけたか
- 最良経路へいつ到達したか
- 何回改善したか
- どのWork時点で品質が改善したか
- 予算変更によって何が起きたか

また、`Found=false`だけでは失敗原因を区別できず、改善対象を特定しにくかった。

そのため、anytime履歴、failure reason、機構別アブレーション、BRIDGE固有overhead指標を追加する必要があった。

## 7.2 実装内容

### Anytime metrics

各raw runへ次を保存した。

- time to first path
- time to best found
- improvement count
- quality history
- budget history

quality historyには、candidateまたはincumbent更新ごとに次を保存した。

- sequence
- elapsed time
- Work
- distance

summary modeでもcandidate eventをメモリ保持し、trace fileを保存しなくても主要履歴をartifactへ残せるようにした。

### Failure Reason

error codeとalgorithmic outcomeを分離し、次を分類できるようにした。

- disconnected
- budget exhausted
- timeout
- fallback failure
- invalid request
- no path

query別summaryでfailure reasonの件数を集計する。

### Ablation Options

Scenarioから次の型付きoptionを宣言できるようにした。

- disable fallback
- disable certification
- disable detour
- disable budget reallocation
- disable state reuse

fallbackとcertificationはTRUSSの実制御へ反映した。

独立機構が未実装のoptionは、後続のHEALTHY改修で黙って無視せずvalidation errorとする方針へ強化された。

### BRIDGE固有指標

次を追加した。

- Bridge Overhead Ratio
- Duplicated Work Ratio
- State Reuse Ratio

Bridge Overhead RatioはTRUSS orchestration時間をBRIDGE全体時間で割った指標であり、solverとの差分を推測するのではなく、型付きTimeBreakdownから算出する。

Duplicated Work Ratioは、ANCHORとBOLTS間で重複して観測されたnode・edgeをportfolio Workへ対して診断する。

State Reuse Ratioは、現行でreuseがない場合は0として記録するが、未計測・未実装と混同しないよう用語上の制約を定義した。

## 7.3 解決されたこと

- 最終結果だけでなく、品質改善の時間・Work推移を分析できるようになった。
- 「最初の経路は速いが最良品質が遅い」等の特性を区別できるようになった。
- 非連結、予算不足、timeout、fallback失敗を分けて集計できるようになった。
- fallbackやcertificationの効果を同じScenario系列で比較できるようになった。
- BRIDGE固有のorchestrationや重複探索を定量化できるようになった。
- 機構を無効化した場合の品質・Work・時間差を追跡できる基礎が整った。

## 7.4 影響範囲

### 主なコンポーネント

- CORE
- TRUSS
- ANCHOR
- TRAFFIC
- ULTRASOUND
- GATE

### 主な契約・文書

- Scenario ablation schema
- benchmark result schema
- ANCHOR／TRUSS component rules
- 用語集
- research readiness script

### 主なテスト

- first path／best found保存
- improvement count集計
- failure reason分類
- fallback／certification無効化
- raw history保存
- overhead ratio統計

---

# 8. Phase 5: graph family拡張・dataset・統計再現性

## 8.1 なぜ必要だったのか

限られたtopologyだけで評価すると、特定構造に過適合した改善を一般的な性能向上と誤認する可能性がある。

また、実データを使用する場合、出典や前処理が不明確では同じ実験を再現できない。

そのため、人工graph familyを増やし、実データ取り込み契約と統計report再生成機能を追加する必要があった。

## 8.2 実装内容

### Community Graph

局所的に密なcommunityと、community間の疎なbridgeを持つgraph generatorを追加した。

局所探索と長距離接続のバランス、community間移動、bridge依存性を評価できる。

### Maze Graph

randomized depth-first carvingによるperfect-maze型graphを追加した。

長い迂回、一本道、分岐後の戻り等を評価できる。

### Adversarial Graph

次を持つstress graphを追加した。

- 低costに見えるdead end
- 高cost shortcut
- 誤誘導枝

ヒューリスティックや局所判断が不利になる条件を評価できる。

全generatorは固定parameterとseedに対して決定論的である。

### Dataset Contract

`bridge.dataset.v1`を追加した。

主要項目は次のとおりである。

- dataset ID
- source
- license
- directedness
- nodes
- weighted edges
- optional positions
- default query
- preprocessing history

Dataset readerはTRAFFIC内に限定した。

GATE、CORE、TRUSS、ANCHOR、BOLTSへ外部file readerを追加していない。

### Dataset Provenance

各raw runへ次を保存した。

- dataset ID
- source
- license
- dataset path
- SHA-256
- preprocessing history

SHA-256はdataset fileの正確なbyte列から算出する。

### 研究Scenario

次を含む研究Scenarioを追加した。

- community
- maze
- adversarial
- dataset fixture
- 複数seed
- 複数repetition
- warm-up
- randomized order
- BRIDGE、ANCHOR、Dijkstra等の比較

### Statistics Report

raw runsを直接読み取る統計scriptを追加した。

出力内容は次のとおりである。

- observation count
- mean
- median
- standard deviation
- bootstrap 95% confidence interval
- Mann–Whitney U近似
- Cliff's delta

warm-upは除外し、metricとgroupをCLIから指定できる。

固定bootstrap seedでは同じreportを再生成できる。

## 8.3 解決されたこと

- 特定topologyだけで改善していないか確認できるようになった。
- community、maze、adversarial等の異なる難しさを比較できるようになった。
- 実データをproduction routeへ混入させず研究用途で使用できるようになった。
- datasetの出典、ライセンス、内容、前処理を追跡できるようになった。
- 同じraw observationsから統計reportを再生成できるようになった。
- 平均値だけでなく分布差とeffect sizeを確認できるようになった。

## 8.4 影響範囲

### 主なコンポーネント

- TRAFFIC
- compatibility／statistics scripts

### 影響を受けないproduction components

- GATE
- CORE
- TRUSS
- ANCHOR
- BOLTS

これらへdataset I/Oや統計分析ロジックは追加していない。

### 主な契約・文書

- `BENCHMARK_DATASET_SPEC_v1.md`
- benchmark Scenario schema
- benchmark Result schema
- 用語集
- TRAFFIC component rule

### 主なテスト

- generator決定性
- dataset validation
- provenance保存
- SHA-256保存
- preprocessing順序保持
- Scenario実行
- statistics report生成

---

# 9. Phase横断で実施したガバナンス改善

## 9.1 用語集の正本化

BRIDGE固有の意味を持つ語を`docs/WORD_DEFINITION.md`へ集約した。

追加・整理された主な語は次のとおりである。

- Scenario
- Run
- Raw Run
- Query ID
- Graph Instance ID
- Warm-up
- Execution Manifest
- Observation Mode
- Summary／Trace／Profile
- Time Breakdown
- System Metrics
- Quality History
- Budget History
- Failure Reason
- Ablation
- Bridge Overhead Ratio
- Duplicated Work Ratio
- Dataset Provenance
- Graph Family
- Effect Size
- Reproduction Manifest

新しいschema field、CLI option、event vocabulary、benchmark metricを追加する場合、同一変更で用語集を更新する規則を追加した。

## 9.2 schemaと実装の同期

各Phaseで次を同期した。

- Go構造体
- JSON Schema
- 仕様書
- component rule
- Scenario fixture
- acceptance test

公開fieldだけ存在して実行経路へ接続されていない状態を最終監査で確認した。

## 9.3 Legacy整理

旧ULTRASOUND Recorderを現行経路から除外し、legacyへ移行した。

現行の観測実装はCollectorへ一本化した。

旧modeや曖昧な互換層を残すことによる契約分岐を防止した。

## 9.4 最終gap audit

各Phase完了後に、次を確認した。

- Scenario fieldが実際のコードへ接続されている。
- raw resultからsummaryを再構計算できる。
- dataset I/OがTRAFFIC外へ漏れていない。
- 統計分析がsolverやGATEへ混入していない。
- 観測I/OがWorkへ混入していない。
- Stable Digestがmode間で一致する。
- schemaとGo contractが一致する。
- 用語集に新規独自語が存在する。
- legacy観測経路が現行実行へ戻っていない。

---

# 10. コンポーネント別の影響範囲

## 10.1 TRAFFIC

### 主な変更

- Scenario／Run分離
- 複数seed、query、repetition、warm-up
- deterministic execution order
- raw run保存
- query別summary
- graph／query／environment metadata
- Work、時間、system metrics集計
- Observation Result保存
- quality／budget history保存
- failure reason集計
- ablation設定
- graph generator拡張
- dataset loader
- dataset provenance

### 影響

TRAFFICは単なるbenchmark起動処理から、再現可能な実験実行・artifact生成コンポーネントへ拡張された。

ただし、solver private stateの参照、研究上の合否判定、統計的な結論生成は担当しない。

## 10.2 ULTRASOUND

### 主な変更

- `off / summary / trace / profile`
- Collector一本化
- deterministic sampling
- observation overhead
- quality／budget history
- trace manifest
- replay拡張

### 影響

観測の詳細度とオーバーヘッドを用途別に分離できるようになった。

profile modeは高コストな検証・解析用であり、純粋な速度比較とは分離して扱う。

## 10.3 BEARING

### 主な変更

- canonical event vocabulary
- event class
- fallback、certification、state reuse等の語彙

### 影響

各solverとULTRASOUNDの間で、観測eventの意味が統一された。

BEARING自身は保存、集計、分析、制御を行わない。

## 10.4 CORE

### 主な変更

- WorkMetrics
- TimeBreakdown
- SystemMetrics
- Ablation Options
- Failure Reason
- anytime metrics

### 影響

研究用に必要な共有測定契約が型付きで統一された。

COREは研究上の合否判断やdataset I/Oを持たない。

## 10.5 TRUSS

### 主な変更

- phase別時間
- fallback／certification event
- ablation反映
- failure reason
- overhead指標
- anytime結果の統合

### 影響

BRIDGE固有の制御コストと結果形成過程を観測・分析できるようになった。

observer結果によって制御判断を変えない原則を維持した。

## 10.6 ANCHOR

### 主な変更

- Work内訳
- phase時間
- candidate／incumbent event
- quality history
- observation mode非干渉検証

### 影響

ANCHOR単独とBRIDGE組込み時の両方で、同一の観測・測定契約を使用できるようになった。

## 10.7 BOLTS

### 主な変更

- solver時間とWork内訳
- fallback／certification用途の測定
- BRIDGE、ANCHORとの同一artifact比較

### 影響

既存アルゴリズムをPerformance Referenceとして同一Scenario内で比較できる基礎が整った。

## 10.8 GATE

### 主な変更

- 公開結果fieldの追加
- observation mode validation
- TimeBreakdown等の転記

### 影響

外部利用者から研究用測定値へアクセスできるようになった。

ただし、baseline比較、統計分析、合否判断をGATEへ持ち込んでいない。

---

# 11. 実装によって成立した開発評価

研究ベンチマーク基盤化後は、次のような問いへraw dataで回答できる。

## 11.1 品質

- BRIDGEはDijkstraと同じ経路距離を返しているか
- ANCHORのdistance ratioは改善したか
- p95やworst distance ratioが悪化していないか
- first pathの品質とbest foundの品質はどう推移したか

## 11.2 Work

- Total Workは減ったか
- Expandは減ったがEvaluateが増えていないか
- fallbackによる追加Workはどの程度か
- duplicated work ratioは減ったか
- graph familyごとにWork削減傾向が異ならないか

## 11.3 時間

- solver timeは減ったか
- BRIDGE orchestration timeが増えていないか
- p50は改善したがp95が悪化していないか
- observation overheadは許容範囲か

## 11.4 メモリ

- allocationは増えていないか
- GC countが増えていないか
- profile時のsampled heap peakが増えていないか

## 11.5 Anytime挙動

- 最初の経路発見が早くなったか
- 最良品質への到達が早くなったか
- 改善回数が無駄に増えていないか
- 予算再配分と品質改善の関係はどうか

## 11.6 一般性

- gridだけでなくcommunityでも改善しているか
- mazeで極端に悪化していないか
- adversarial graphでfallbackが増えていないか
- 実datasetでも同じ傾向が再現するか

---

# 12. 実装によって解決されなかったもの

本計画は評価基盤を整備するものであり、次を自動的に解決するものではない。

- ANCHORやBRIDGEのアルゴリズム性能そのもの
- 比較対象solverの追加実装
- Scenario条件の十分性
- 実データの収集量
- Python版とGo版のWork傾向差
- タイミングノイズの完全除去
- 研究仮説やRegression Policyの選定

これらは、整備された基盤を利用してPDCAを回し、問題発見、修正、再評価を行う対象である。

また、Phase 1からPhase 5の時点では、path、distance、Exact、Reported Workの独立検証と自動回帰判定は完全ではなかった。

この不足は、後続の`HEALTHY`新設とWork Validation改修によって補完された。

---

# 13. 検証結果

Phase 1からPhase 5の各完了時点で、次の横断検証を実施した。

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `python tests/compatibility/verify.py`
- Python-Go semantic parity
- Scenario validation
- raw result再集計
- Stable Digest mode間一致
- generator決定性
- dataset provenance検証
- statistics report再生成

最終gap auditでは、Phase 1からPhase 5の計画完了を妨げる必須級の未接続field、責務境界違反、現行／legacy経路の混在は確認されなかった。

ただし、Python版とGo版のWork trend correlationは、基盤実装とは別の移植・アルゴリズム評価課題として残った。

---

# 14. 最終的に何が成立したか

研究ベンチマーク基盤化によって、BRIDGEには次の能力が成立した。

## 再現可能な実験

- 複数seed
- 複数query
- 複数repetition
- warm-up
- deterministic randomization
- environment metadata
- Run ID
- execution manifest

## 多角的な測定

- 品質
- Work
- Step
- phase別時間
- memory／runtime
- anytime履歴
- failure reason
- fallback／certification
- observation overhead
- BRIDGE固有overhead

## 観測の非干渉性

- off
- summary
- trace
- profile

これらのmode間で決定論的結果が変化しないことを検証できる。

## 一般化評価

- grid
- random geometric
- community
- maze
- adversarial
- `bridge.dataset.v1`による実データ

## raw dataからの再分析

- summary再計算
- bootstrap confidence interval
- Mann–Whitney U
- Cliff's delta
- group／metric別report

---

# 15. 総括

研究ベンチマーク基盤化以前のBRIDGEは、benchmarkを実行して一部の平均値を取得できる状態であったが、実験単位、観測契約、内部内訳、anytime挙動、dataset provenance、統計再現性が十分に統一されていなかった。

Phase 1からPhase 5の改修によって、次が実現した。

1. ScenarioとRunが分離され、各測定値の由来を追跡できるようになった。
2. raw observationsからsummaryを再計算できるようになった。
3. Work、時間、メモリをcomponent・phase別に分析できるようになった。
4. ULTRASOUNDの観測modeが統一され、非干渉性とtrace完全性を検証できるようになった。
5. ANCHORとBRIDGEのanytime改善過程をquality-time／quality-workで追跡できるようになった。
6. failure reasonとablationにより、失敗原因と機構別効果を分析できるようになった。
7. community、maze、adversarial、datasetを用いて一般性を確認できるようになった。
8. raw dataから統計reportを再生成できるようになった。
9. 用語集、schema、component rule、テストが同じ契約へ統一された。
10. 実験実行責務をTRAFFIC、観測責務をULTRASOUNDへ分離したまま、production routing componentsへ研究専用I/Oや統計判断を混入させずに済んだ。

これにより、BRIDGEの開発では、単一の平均実行時間だけを見るのではなく、品質、Work、Step、時間、メモリ、内部挙動、topology依存性を複数の視点から確認できるようになった。

研究ベンチマーク基盤化は、性能改善そのものを自動的に生み出すものではない。しかし、変更前後を同一条件で実行し、どの値がどの理由で変化したかを確認し、次の改善対象を選択するためのPDCA基盤を成立させた。
