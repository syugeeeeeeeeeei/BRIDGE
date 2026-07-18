# Benchmark Scenario Contract

> Status: Normative  
> Applies To: `bridge.benchmark.v1`  
> Producer: User / scenario generator  
> Consumer: TRAFFIC

## 目的

Benchmark Scenarioは、TRAFFICが実行計画を決定論的に展開するための入力契約です。旧`endpoints`、`artifact_id`、`save_raw_results`、`save_trace`、`capture_environment`は受理しません。

## 必須構造

- `schema_version`: `bridge.benchmark.v1`
- `suite.id`: 空でない文字列
- `execution.repetitions`: 1以上100000以下
- `execution.seeds`: 1件以上
- `algorithms`: 1件以上
- `observation.mode`: `off`、`minimum`、`debug`、`trace`
- `output.directory`: 成果物の親ディレクトリ
- `scenarios`: 1件以上

## 実行計画

実行計画は、Scenario、Algorithm、Seed、Query、Warm-up、正式反復の直積から生成します。`randomize_order=true`の場合は、展開後の計画をSeedから決定論的にシャッフルします。

- `warmup_runs`: 各組合せに対する予備実行回数です。0以上1000以下です。
- `repetitions`: 正式測定回数です。Warm-upを含みません。
- `run_ordinal`: シャッフル後の実際の実行順です。

## Query

Queryは`source`と`target`を所有します。旧`endpoints`フィールドは使用しません。`selection.method=generator_default`ではGeneratorが既定Queryを提供します。

## Observation

- `off`: 観測を行いません。
- `minimum`: 比較に必要な最小計測のみを記録します。
- `debug`: 集計済み診断情報を記録します。
- `trace`: Event traceとGraph snapshotを保存します。

Warm-upは正式統計から除外します。観測負荷を避けるため、Warm-upのTrace保存は行いません。

## 出力

TRAFFICは正規化後Scenarioを`scenario.json`へ保存します。ベンチマーク完了後、Execution Directoryと同内容のBenchmark ArchiveをZIPとして生成します。

## 機械契約

`src/contracts/json-schema/benchmark-scenario-v1.schema.json`を正本Schemaとします。
