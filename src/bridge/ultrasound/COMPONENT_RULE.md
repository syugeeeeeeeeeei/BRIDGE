# ULTRASOUND コンポーネント規則

**対象package:** `src/bridge/ultrasound`  
**対象版:** v0.15.0以降  
**状態:** 規範文書

## 1. 責務

- BEARING eventの収集、sequence付与、保存
- replayとschema検証
- Work Model v2再構成
- Anytime曲線`U(W)`、`L(W)`、`U/L`
- state reuse ratio、duplicate work ratio
- truncation、sink、観測コストの記録

## 2. 禁止事項

- solver選択、budget再配分、candidate選択
- graph変更
- 観測結果を探索順序へ返すこと
- 本番必須依存化

## 3. Observation Mode

- `off`: event収集なし
- `summary`: 集計に必要なeventのみ
- `trace`: replay可能なevent列を保存
- `profile`: traceにprofiling情報を追加

## 4. 不変条件

- Observationの有無でpath、proof、termination、Workを変更しない
- Trace保存I/O、sink待ち、serializationをWorkへ含めない
- Work再構成値はResultおよびBudget Ledgerと一致する
- 未計測値を0で偽装しない

## 5. 必須テスト

- Work再構成テスト
- Anytime曲線単調性テスト
- state reuse・duplicate work算出テスト
- replay決定論性テスト
- Observer非干渉テスト

## Lifecycle span aggregation

- ULTRASOUND pairs lifecycle `started` and `completed`/`failed` events by run and span identifiers.
- ULTRASOUND calculates durations from observer-assigned monotonic elapsed timestamps.
- Incomplete, duplicate-start, and orphan-completion spans MUST be reported explicitly.
- ULTRASOUND MUST NOT alter routing, solver selection, Work, or termination decisions.

## Collector lifecycle

- A Collector is a one-run observation object by default.
- Reuse across runs requires `Close` followed by `Reset`.
- Completed spans and event history MUST NOT silently accumulate across independent runs.
- Benchmark and server integrations SHOULD create one Collector per request/run unless explicit reuse is required.
