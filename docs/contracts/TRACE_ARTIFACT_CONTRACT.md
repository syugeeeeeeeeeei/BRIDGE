# Trace Artifact Contract

> Status: Normative  
> Applies To: Trace Event v1 / Trace Manifest v1  
> Producer: ULTRASOUND through TRAFFIC  
> Consumers: HEALTHY, Simulator

## 構造

```text
traces/run-000001/
├── manifest.json
├── graph.json
└── trace.jsonl
```

Traceは正式測定Runで`observation.mode=trace`の場合にのみ保存します。

## Event順序

`trace.jsonl`は1行1JSONです。Eventは`sequence`の昇順で保存し、同一Run内で一意でなければなりません。`elapsed_ns`はRun開始からの経過時間、`logical_step`はアルゴリズム上の論理順序です。

必須の意味フィールドは、`sequence`、`elapsed_ns`、`logical_step`、`component`、`kind`、`action`です。Work変化を伴うEventは`work_before`と`work_after`を記録します。

## 完全性

Trace manifestはEvent件数、切り捨て、Dropped event、Sampling情報、Trace SHA-256を記録します。`truncated=true`またはDropped eventがあるTraceを完全な探索履歴として扱ってはなりません。

## 禁止事項

- フォルダ名からRun属性を推測しません。
- Eventを並べ替えて保存しません。
- 欠落EventをConsumer側で捏造しません。
- Traceを正式な性能時間の正本として扱いません。

## 機械契約

- `src/contracts/json-schema/trace-event-v1.schema.json`
- `src/contracts/json-schema/trace-manifest-v1.schema.json`
