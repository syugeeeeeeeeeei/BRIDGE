# Benchmark Scenario v3 実装結果

## 実装済み

- Scenario Schemaを`bridge.benchmark.v3`へ更新
- 観測設定を`observation.mode`へ統一
- 観測モードを`minimum`、`debug`、`trace`へ限定
- `save_trace`、`sample_rate`、`output_dir`、`artifact_id`等の旧Scenarioフィールドを削除
- `output.directory`だけを利用者設定として採用
- 実行ごとに`<output.directory>/<suite.id>/<execution_id>/`を新規作成
- 実行IDを時系列順に並べられる一意IDとして自動生成
- `result.json`、`scenario.json`、`manifest.json`、`environment.json`、`runs.jsonl`、`summary.csv`を常時保存
- CLI実行時にHEALTHYを自動実行し、`healthy.json`を保存
- `trace`時にRun別TraceとTrace Manifestを自動保存
- `minimum`時に詳細Collectorを生成しない
- `debug`時に集約Collectorを使用し、Trace I/Oを行わない
- Queryを`queries[].selection.method`へ統一
- Budgetを`work_limit`、`search_time_limit`へ変更
- Execution Timeoutを`run_timeout`へ変更
- Route Modeを`route.mode`へ変更
- v3 JSON Schemaとv3 Scenario例を追加
- 旧v2 Scenario例と旧Scenario Schemaを削除

## 検証結果

- `go test ./...`: 成功
- `bridge benchmark validate tests/examples/benchmark-smoke-v3.json`: 成功
- `bridge benchmark tests/examples/benchmark-smoke-v3.json`: 成功
- Smoke Benchmark: 2 Scenario、4 Algorithm、8 Runすべて完走
- 実行成果物ディレクトリと`healthy.json`の生成を確認

## 補足

Route APIの`observation_config`はBenchmark Scenarioとは別の公開契約であるため、今回のScenario v3改修では維持している。Route API側の観測契約を同じ構造へ変更する場合は、Route Request Schemaの独立した破壊的変更として扱う。
