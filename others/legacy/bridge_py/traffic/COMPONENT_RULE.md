# TRAFFIC COMPONENT RULE

## 1. 定義

TRAFFICはBRIDGEへ再現可能な試験負荷を投入し、機能、性能、耐久性、回帰、観測整合性を評価する開発・検証専用基盤である。solverではなく、実ユーザーrequestを処理しない。

## 2. 所有する責務

- graph/query/scenario/seed生成
- unit、contract、golden、benchmark、stress、soak、burst、concurrency、fault、replay、regression
- warm-up、本計測、実行順管理
- baselineとのpaired comparison
- p50/p95/p99/worst、confidence interval、verdict
- failure bundle、manifest、environment fingerprint
- ULTRASOUND artifactの公開reader経由評価

## 3. 所有してはならない責務

- solver内部ロジック、TRUSS判断、budget配分の変更
- ANCHOR/BOLTS private APIの呼び出し
- BEARING eventの捏造
- 同一実行中のbenchmark結果による探索制御
- 本番RouteResultの改変
- 本番サービスへの組込み

## 4. 依存規則

### 許可

- GATE公開API（end-to-end試験の原則経路）
- CORE公開schema
- ULTRASOUND公開artifact/validator API
- BOLTS公開baseline registry（solver単体評価に限定）
- composition rootでGATEへTRUSS/observerを注入するための公開constructor

### 制限付き例外

- 現行`TrafficRunner`のTRUSS直接importは、観測付きGATEを構成するdependency injectionに限る。
- route実行自体は必ず`Gate.route_request()`経由とする。
- TRUSSのprivate method、state、moduleへアクセスしてはならない。
- 将来GATEにobserver factoryが追加された場合、この例外は除去する。

### 禁止

- ANCHOR/BOLTS/TRUSS private modules
- BEARING adapter internals
- ULTRASOUND internal buffers
- production configurationの暗黙変更

## 5. 再現性規則

- scenario ID、seed、graph spec、query、mode、budget、version、environmentを保存する。
- 同一config/seedでcase集合を再生成できる。
- randomized execution orderを保存する。
- failure caseを単独再実行できる。
- timing比較ではwarm-up、反復、分散を考慮する。

## 6. 判定規則

- invalid path、budget violation、semantic trace errorは1件でもfailとする。
- connected/disconnectedを分離集計する。
- distance ratioは有効なbaseline found caseのみ計算する。
- 単発wall timeだけで性能回帰を断定しない。
- observer ON/OFFのpath、distance、work、trace-independent result一致を確認する。

## 7. Artifact規則

- raw recordとaggregate summaryを分離する。
- manifestにcase-to-trace対応を保存する。
- traceはULTRASOUND validator合格後のみ有効とする。
- verdict理由を機械可読形式で保存する。

## 8. 必須テスト

- seed再現性
- valid path/oracle comparison
- budget/deadline/cancellation
- disconnected
- non-interference
- trace semantic validation
- paired others/legacy/new regression
- stress/soak resource drift
- 禁止依存検査

## 0. 文書の効力

この文書は当該コンポーネントの実装・レビュー・テスト・将来変更に対する規範である。ルートのアーキテクチャ仕様と矛盾する場合は、より厳格な規則を採用し、矛盾を放置してはならない。

変更時には、コード、公開契約、テスト、観測意味論、本書を同一変更単位で更新する。文書と実装の不一致は不具合として扱う。
