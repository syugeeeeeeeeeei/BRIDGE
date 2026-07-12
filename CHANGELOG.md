## v0.14.3

- TRUSS Budget Ledgerを公開artifactへ追加し、HEALTHYのLevel 2 Work照合を実装。
- HEALTHYがprofile trace manifestとJSONLを自動読込みし、SHA-256・sampling・drop・truncation検査後にWorkを再構成。
- Bidirectional DijkstraとReachabilityへprofile Action Eventを追加し、全solverでReported/Reconstructed Work一致を検証。
- Reachabilityの返却distanceを実path weightへ修正。
- 旧Work診断fieldをDerived Compatibility Counterとして位置付け、WorkMetricsを唯一の正本に固定。
- 固定fixture、profile E2E、budget ledger、trace digestの受入テストを追加。

## v0.14.2

- HEALTHY analysis and validation component.
- Work action contract and conservation validation.
- Path, distance, Exact reference, paired comparison, and regression policy.
- Unsupported ablations now fail validation.

## Repository structure refactor

- ルートを`src/`、`docs/`、`tests/`、`others/`へ再編
- BRIDGE本体、SDK、contracts、productsを`src/`へ集約
- examples、scenarios、compatibilityを`tests/`へ集約
- legacyを`others/`へ隔離
- 現行CLIが利用するGo internal packageは可視性規則に従い`src/internal/`へ配置
- import、CLIビルドパス、互換性スクリプト、文書内パスを新構造へ追従

## v0.14.0

- Python SDKとTypeScript SDKを追加
- 5プラットフォーム向けBRIDGEバイナリを両SDKへ静的同梱
- 明示パス、`BRIDGE_BINARY`、同梱版、PATHのバイナリ解決を実装
- Route Request/Result型、終了コード別例外、timeout、TypeScript AbortSignalを実装
- SDKのバイナリ自動ダウンロードを明示的に不採用
- BRIDGE自身はAPIサーバーを提供せず、SDK利用者が構築する方針へ更新

# CHANGELOG

## v0.13.1

- Collector／SinkをBEARINGからULTRASOUNDへ移動
- GATEの観測具体実装生成を廃止し、Observer契約注入へ変更
- graph file読込みをGATEからCLI入力アダプターへ移動
- TRAFFICとCLIの保存実装依存先をULTRASOUNDへ修正
- 観測Close失敗で探索結果を破棄しない挙動へ修正
- package依存行列を検査するarchitecture testを追加
- 規範文書と現行ディレクトリ構成を同期

# 変更履歴

## v0.5.1 - 文書刷新

- `docs/ARCHITECTURE_RULE.md`をGo本番実装向けに全面改訂
- 全`src/bridge/*/COMPONENT_RULE.md`を日本語化し、責務・依存・禁止事項を具体化
- architecture、algorithm、migration、report、repository文書を日本語へ統一
- 旧Python版の設計内容を履歴として保持しつつ、現行Go仕様と区別
- 規範文書の優先順位を更新

## v0.5.0 - 研究準備性評価

- 75 paired casesのPython-Go比較
- topology coverageと傾向相関による移植完了判定
- `migration_complete: true`

## v0.4.0

- semantic Work Action
- 双方向Dijkstra
- Python-Go semantic parity

## v0.3.0

- 決定論的benchmark
- Stable Digest

## v0.2.0

- ANCHOR複数strategy移植

## v0.1.0

- Go architecture基盤

## v0.6.0
- 離散semantic Action Workを維持し、ANCHORの0.45N目標をEXPAND上限へ分離。
- ANCHOR、A*、双方向ダイクストラの同一Work計測と横比較CLIを追加。
- ULTRASOUNDへ詳細trace集計・validationを追加。

## v0.8.0 experimental
- Added checkpoint-based long-range BOLTS connector starts.
- Added investigated node/edge coverage metrics and candidate/path counts.
- Kept BEARING as an observation-only intermediate contract.

## v0.9.0
- First-stage TRUSS separation: Orchestrator, Budget, Supervisor, Arbiter.
- CORE coordination contracts and emergency/directive events.
- Production ANCHOR/BOLTS direct invocation removed from TRUSS wiring.
- Added component Work and investigated-range aggregation.
- Documented future Scheduler and Session Registry separation conditions.

## v0.9.1

- Added component-level runtime breakdown for TRUSS orchestration.
- Added exact portfolio unique and cross-component duplicate investigation metrics.
- Added stable node/edge trace identifiers to ANCHOR and BOLTS telemetry.
- Standardized investigated-edge ratios to directed adjacency slots.
- Optimized trace aggregation for single-component executions.
- Documented that true mid-search emergency handling requires incremental ANCHOR execution and remains a subsequent architectural change.

## v0.10.0 - Replayable ULTRASOUND Trace

- Traceを外部visualizer向けの公開データ契約`bridge.trace.v1`として定義
- Eventへrun ID、sequence、elapsed/delta time、component、Action、Work前後を追加
- frontier、node、edge、relaxation、candidateのState Deltaを記録
- run directoryへmanifest、events、metrics、result、validationを保存
- `bridge-ultrasound record|validate|replay` CLIを追加
- 保存TraceのみからReplayStateを復元するULTRASOUND replay機能を追加
- Trace、Replay、Metrics、Log、State Deltaの用語定義を更新
- 外部シミュレーター向けTrace Data Contractを追加

## v0.12.0

- Added Scenario-driven benchmark execution through `bridge benchmark run`.
- Added strict YAML/JSON scenario validation and `benchmark validate/list`.
- Added console, JSON, JSONL, and CSV output with explicit output-file handling.
- Added acceptance criteria and exit code 5 for acceptance failures.
- Added benchmark scenario/result JSON Schemas and reproducible examples.

## v0.12.1

- `graph.nodes` 指定時に正確なノード数を生成する `GridNodes` を追加。
- Scenario validationを強化し、不正endpoint、Observation Mode、budget、acceptanceを実行前に拒否。
- 未実装の並列実行を黙認せず、`execution.jobs: 1` のみに制限。
- Route Result、Trace Event、Trace ManifestのJSON Schemaを追加。
- Benchmark Result Schemaをcase構造と数値範囲まで厳密化。
- 正式CLIの回帰テストを追加。
- benchmarkのcontext timeout/cancelを終了コード4へ分類。

## v0.13.1

- ULTRASOUND CollectorとEvent Sinkを分離
- FileSink、MemorySink、WriterSink、CallbackSink、MultiSinkを製品利用向けに強化
- `bridge route --trace-output`を追加
- `bridge benchmark run --trace-dir`を追加
- Route ResultへObservation集計を追加
- TraceのNaN／Infinity正規化を追加
- Observation非干渉性テストとSink回帰テストを追加

## v0.14.1-dev - 研究ベンチマーク基盤 Phase 1

- benchmark v1 Scenarioへwarm-up、複数query、決定論的Run順randomize、artifact/environment metadataを追加。
- Run identityをscenario、algorithm、graph instance、query、repetitionへ分解。
- resultへraw runsと再計算可能な分位点・標準偏差・95%信頼区間を追加。
- ULTRASOUND/GATEの観測modeを`off`、`summary`、`trace`、`profile`へ整理。
- 旧単一endpoint Scenarioとraw-result保存pathの互換性を維持。

## v0.14.1 research foundation Phase 2

- Added typed `TimeBreakdown` and `SystemMetrics` contracts.
- Added TRUSS/ANCHOR/BOLTS/fallback/supervisor/arbiter/orchestration/GATE timing separation.
- Added per-run allocation, malloc, GC, and heap observations to TRAFFIC raw results.
- Kept runtime measurement and observation I/O outside search Work accounting.

## v0.14.1 terminology governance

- BRIDGE固有の意味・用法を持つ語について、`docs/WORD_DEFINITION.md`への同時定義を必須化
- 研究benchmark、観測mode、時間内訳、system metricsの正式用語を追加
- 規範文書を設計どおり`docs/`配下へ配置
- 用語集の必須語と用語管理規則を検査する互換性テストを追加
