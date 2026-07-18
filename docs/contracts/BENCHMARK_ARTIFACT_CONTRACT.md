# Benchmark Artifact Contract

> Status: Normative  
> Applies To: `bridge.benchmark.artifact.v1` / `bridge.benchmark.execution.v1`  
> Producer: TRAFFIC  
> Consumers: HEALTHY, Simulator, analysis tools

## 用語

- Benchmark Execution: Scenarioを一度展開して実行した単位です。
- Execution Directory: 展開済み成果物の保存先です。
- Artifact Bundle: 1回のBenchmark Executionの成果物集合です。
- Benchmark Archive: Artifact Bundleを格納したZIPです。

## 必須構造

```text
<execution-id>/
├── manifest.json
├── scenario.json
├── environment.json
├── result.json
├── runs.jsonl
├── summary.csv
├── handoffs.csv
├── healthy.json
└── traces/
    └── run-000001/
        ├── manifest.json
        ├── graph.json
        └── trace.jsonl
```

`healthy.json`と`traces/`は、該当機能が実行された場合に存在します。ArchiveはExecution Directory内の成果物を相対パスのまま格納します。

## 各成果物の責務

| ファイル | 責務 | 区分 |
|---|---|---|
| `manifest.json` | Execution識別情報と索引 | 正本 |
| `scenario.json` | 正規化済み入力 | 正本 |
| `environment.json` | 実行環境 | 正本 |
| `result.json` | Execution全体の完全な結果 | 正本 |
| `runs.jsonl` | Run単位のストリーム表現 | 正本 |
| `summary.csv` | 人間向け集計 | 派生物 |
| `handoffs.csv` | Handoff集計 | 派生物 |
| `healthy.json` | HEALTHY検査結果 | 検査成果物 |

`summary.csv`や`handoffs.csv`から正本データを復元してはなりません。

## Run識別

- `run_id`: 成果物内の正式な一意識別子です。
- `run_ordinal`: 実際の実行順です。1始まりです。
- Run directory: `run-%06d`です。`run_ordinal`と一致します。
- Run directory名からAlgorithm、Seed、Queryを推測してはなりません。
- Consumerは`references`に記録された相対参照を使用します。

999999を超えるRunを要求するScenarioは、実行前検証で拒否します。

## Warm-up

Warm-upは`run_ordinal`を消費し、`warmup_run=true`として記録できますが、正式統計、成功率、性能集計から除外します。Warm-upのTraceは保存しません。

## 失敗時

一部Runが失敗しても、可能な限り結果と失敗理由を保存します。Execution自体の成否は`run_metadata.execution_succeeded`で示します。Archive作成失敗はCLIエラーとします。

## 参照と完全性

- 参照パスはExecution DirectoryまたはArchive rootからの相対パスを正本とします。
- 外部絶対パスをArchive契約に含めません。
- SHA-256が記録されている成果物は、Consumerが使用前に照合します。
- ZIP entryに絶対パス、`..`、symlinkを含めてはなりません。

## 機械契約

- `src/contracts/json-schema/benchmark-artifact-v1.schema.json`
- `src/contracts/json-schema/benchmark-execution-manifest-v1.schema.json`
