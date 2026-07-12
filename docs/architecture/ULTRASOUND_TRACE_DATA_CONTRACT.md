# ULTRASOUND Trace Data Contract v1

## 1. 目的

本仕様は、BRIDGEとは独立した外部プロジェクトが、ULTRASOUNDの出力だけを入力として探索過程を視覚的に再生できるようにするための公開データ契約である。表示UI、描画方式、再生エンジン自体はBRIDGEに含めない。

## 2. run directory

```text
<output>/<run-id>/
├── manifest.json
├── events.jsonl
├── metrics.json
├── result.json
└── validation.json
```

## 3. Event envelope

各行は独立したJSON objectであり、`schema_version`は`bridge.trace.v1`である。

主要フィールド:

- `run_id`: 実行識別子
- `sequence`: run内で単調増加するEvent番号
- `elapsed_ns`: run開始からの経過時間
- `delta_ns`:直前Eventからの経過時間
- `logical_step`: 理論的探索段階
- `scheduled_step`: 実行スケジュール段階
- `component`: `TRUSS`、`ANCHOR`、`BOLTS`
- `phase`: component内の実行phase
- `kind`: Event種別
- `action`: 離散Work Action
- `work_before` / `work_after`: Action前後のTotal Work
- `attributes`: Event固有のState Delta

## 4. Replayに必要なEvent

- `frontier_enqueued`
- `frontier_selected`
- `node_expanded`
- `edge_evaluated`
- `relaxation`
- `incumbent_updated`
- `candidate_submitted`
- `component_started`
- `component_finished`
- `emergency_reported`
- `directive_issued`
- `search_finished`

## 5. State Delta規約

### frontier_enqueued

- `node`
- `from`（任意）
- `priority`
- `distance`（任意）

### node_expanded

- `node`
- `distance`
- `frontier_size`

### edge_evaluated

- `from`
- `to`
- `weight`

### relaxation

- `from`
- `to`
- `old_distance`
- `new_distance`
- `accepted`

### incumbent_updated

- `distance`
- `path`
- `first_path_work`
- `first_path_expand`

## 6. 外部シミュレーターの最低実装

外部アプリは`events.jsonl`をsequence順に読み、以下の集合・mapへState Deltaを適用するだけで基本表示を構築できる。

- frontier node集合
- expanded node集合
- evaluated edge集合
- node distance map
- parent map
- candidate path列
- component / phase状態
- Workと経過時間

外部アプリはGo packageをimportする必要がなく、JSONLを読める任意の言語で実装できる。

## 7. 時間軸

再生アプリは次の軸を選択できる。

- `sequence`: 1 Eventずつ
- `work_after`: 1 Workずつ
- `logical_step`
- `scheduled_step`
- `elapsed_ns`: 実測時間に比例

## 8. 互換性

- Eventの意味を変更する場合はschema versionを更新する。
- 未知の`kind`は無視できなければならない。
- 既存フィールドを削除せず、追加は後方互換とする。
- 実時間値は決定論的digestへ含めない。

## v0.14.1 Phase 3 mode semantics

`summary`はevent stream artifactを生成せず、control/candidate event由来の集計のみを返す。`trace`はstate replayとquality/work history再構成に必要なeventを保存する。`profile`はtrace eventに加えて高頻度Action eventを保存し、`overhead_ns`と`sink_write_ns`を記録する。

追加されたcanonical control eventは`fallback_started`、`fallback_finished`、`certification_started`、`certification_finished`である。既存の`bridge.trace.v1` envelopeを維持し、attributes追加として後方互換に扱う。
