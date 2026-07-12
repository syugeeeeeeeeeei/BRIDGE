# Phase 1〜3 完了チェックリスト

この文書は、v0.14.1研究ベンチマーク基盤のPhase 1〜3に対する受入基準の正本である。

## Phase 1: 実験単位と集計単位

- [x] ScenarioとRunを分離し、seed・query・repetition・warm-upをraw runへ保存する。
- [x] Run IDをscenario・algorithm・graph instance・query・repetitionで一意化する。
- [x] graph・query・quality・environment metadataをTRAFFICで生成する。
- [x] raw observationsをトップレベルartifactへ保存する。
- [x] 平均・標準偏差・p50・p95・95%信頼区間を生成する。
- [x] query単位でsummaryを分離する。
- [x] raw runsから保存summaryを再計算する横断テストを持つ。
- [x] randomize後の実行順、shuffle seed、shuffle方式をexecution manifestへ保存する。

## Phase 2: Work・時間・system metrics

- [x] Work内訳をCOREの型付き契約で公開する。
- [x] TRUSS・ANCHOR・BOLTS・fallback・Supervisor・Arbiter・GATEの時間内訳を記録する。
- [x] Workと観測I/O・serialization・runtime計測を分離する。
- [x] alloc bytes・malloc count・GC count・heap境界値をraw runへ保存する。
- [x] profile modeで実行中heap sampled peakを採取する。
- [x] Work内訳、phase時間、system metricsを標準summary statisticsへ含める。
- [x] `heap_alloc_boundary_max`と`heap_alloc_sampled_peak`を区別し、厳密な瞬間peakと誤認させない。

## Phase 3: ULTRASOUND観測契約

- [x] `off`・`summary`・`trace`・`profile`を唯一の現行モードとする。
- [x] summaryは保存先なしでもCollectorを接続し、trace I/Oを行わず集計を返す。
- [x] traceは再構成用eventを保存し、profile専用Action eventを除外する。
- [x] profileは高頻度eventとprofile計測を保存する。
- [x] BEARINGがcanonical typed event vocabularyとevent classのみを所有する。
- [x] fallback・certification・state reuse用event vocabularyを定義する。
- [x] observation resultをTRAFFIC raw runへ保存する。
- [x] observation overhead absolute値とrun時間比を保存する。
- [x] `sample_rate`を決定論的samplingへ実際に適用する。
- [x] trace manifestへsampling・欠落・truncation・overhead・digest・checksumを保存する。
- [x] Stable Digestが全観測モードで一致するBRIDGE全体テストを持つ。
- [x] 旧Recorderを現行packageから削除し、legacyへ移行する。
- [x] Collectorを唯一の現行観測実装とする。

## 横断条件

- [x] 用語集へ新規独自用語と意味を追加する。
- [x] JSON Schemaを実装と同期する。
- [x] `go test ./...`に成功する。
- [x] `go test -race ./...`に成功する。
- [x] `go vet ./...`に成功する。
- [x] compatibility verificationに成功する。
