# 責務分離リファクタリング結果

## 実施内容

新しいメインコンポーネントは追加せず、既存パッケージ内の巨大ファイルを責務別ファイルへ分割した。

- CLI: `app.go`, `route.go`, `benchmark.go`, `serve.go`, `scenario.go`, `artifact.go`, `meta.go`
- TRAFFIC: `scenario_model.go`, `scenario_generator.go`, `scenario_runner.go`, `scenario_metrics.go`, `scenario_artifact.go`
- GATE: `contracts.go`, `router.go`, `observation.go`, `graph_mapper.go`
- HEALTHY: `profile.go`, `analyzer_service.go`, `validation.go`, `oracle.go`, `comparison.go`, `evaluation.go`
- Server: `config.go`, `lifecycle.go`, `handlers.go`, `middleware.go`
- CORE: `graph.go`, `route.go`, `errors.go`, `work.go`, `metrics.go`, `solver.go`

## アルゴリズム不変性

`src/bridge/anchor`, `src/bridge/truss`, `src/bridge/bolts`の既存ファイルは、修正前ZIPとSHA-256単位で同一である。

## ベンチマーク同値性

同一の`minimal` Scenario、seed 42、logical worker 1、Work Budget 10,000,000で変更前後を実行した。

一致確認対象:

- Run ID
- stable digest
- Scenario定義
- Graph profile
- Query profile
- Algorithm configuration
- 返却経路
- 経路コスト
- path found
- search completed
- reachability proof
- optimality proof
- termination reason
- improvement count
- Budget Ledger
- quality claims
- Work metrics

結果: warm-up 1件、measurement 1件の合計2 Runですべて一致。

実時間、割当量、実行日時、Execution IDは非決定的計測値のため同値性判定から除外した。

## 検証

- `go test ./...`: 成功
- `go vet ./...`: 成功
- 主要変更パッケージの`go test -race`: 成功
- `tasks/refactor/verify_benchmark_equivalence.py`: 成功

## 今回変更していない事項

この段階は動作不変の物理分割である。公開契約の`src/contracts`への移動、パッケージ階層の細分化、I/O基盤の`internal`への移動は、次段階で行う。これらは依存関係を変えるため、同じベンチマーク同値性検査を継続して必須とする。
