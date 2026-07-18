# HEALTHY Result Contract

> Status: Normative  
> Applies To: `bridge.health.check_result.v1`  
> Producer: HEALTHY

HEALTHYは入力Artifact Bundleをread-onlyで検査し、元成果物を変更しません。

## 検査層

1. Artifact integrity: 必須ファイル、Schema、参照、Checksum、Run ordinal、Trace件数を検査します。
2. Benchmark validity: Timing、Work、Path、Quality claim、Trace completenessを検査します。

## 判定

- `warning`: 解釈上の注意が必要ですが、成果物は利用可能です。
- `error`: 一部の分析結果を信頼できません。
- `fatal`: Bundle全体を有効なBenchmark成果物として扱えません。

HEALTHYはTraceからWorkを再構成する場合、Trace completenessを必ず併記します。切り捨てられたTraceから完全Workを断定しません。

## 機械契約

`src/contracts/json-schema/health-check-result-v1.schema.json`
