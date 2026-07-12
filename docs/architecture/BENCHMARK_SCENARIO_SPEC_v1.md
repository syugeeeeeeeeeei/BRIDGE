# Benchmark Scenario / Result v1 仕様

**Schema:** `bridge.benchmark.v1` / `bridge.benchmark.result.v1`  
**状態:** v0.14.1研究基盤向けドラフト。v1文字列を維持した破壊的更新を許容する。

## 1. 実験単位

- `Scenario` は宣言単位であり、graph生成条件、query集合、route条件、budgetを定義する。
- `Run` は記録単位であり、`scenario_id × algorithm × graph_instance_id × query_id × repetition` で一意に識別する。
- warm-upもRunとして記録するが、case summary、acceptance判定、統計量から除外する。
- graph seedはrepetitionで変更しない。同一graph instance上で反復差を測定する。

## 2. Execution

- `seeds`: graph instanceを定める。空配列は禁止する。
- `repetitions`: 計測反復数。1以上1000以下。
- `warmup_runs`: query・algorithm・seedごとのwarm-up回数。0以上1000以下。
- `randomize_order`: trueの場合、seed集合から導出した決定論的乱数でRun順序を並べ替える。
- `jobs`: v1では1のみ。
- `timeout`: Run単位のcontext timeout。

## 3. Query

`scenarios[].queries[]` は `id` とselectorを持つ。現在のselectorは次の二つである。

- `opposite-corners`: generatorが返す既定source/targetを使用する。
- `explicit`: `source` と `target` を明示する。

旧`endpoints`は単一query用の互換入力として受理し、内部では`query_id=default`へ正規化する。

## 4. Observation

公開modeは次に固定する。

- `off`: eventを収集しない。
- `summary`: candidate/control系の主要eventのみを収集する。
- `trace`: replay用eventを収集する。
- `profile`: profile採取を伴う詳細観測用。純粋速度比較とは分離する。

`metrics`と`debug`はv1ドラフト更新で廃止する。観測I/OはWorkへ含めない。

## 5. Output

- `artifact_id`: 出力集合の利用者指定識別子。
- `save_raw_results`: 各Runの`raw-result.json`を保存する。
- `save_trace`: 各Runのtraceを保存する。
- `capture_environment`: Go version、OS、architecture、CPU論理数をresultへ保存する。
- `metadata`: 研究条件やcommit等の利用者定義文字列metadata。

## 6. Raw result

トップレベル`raw_runs`と個別`raw-result.json`は、少なくとも次を保存する。

- Run identity、seed、repetition、warm-up状態
- graph instance metadata
- query metadata
- found、exact、distance、quality metadata
- Work内訳
- solver time、end-to-end time
- error code

TRAFFICがgraph/query metadataを計算し、solver private stateは参照しない。

## 7. Summary

case summaryはwarm-upを除外したraw observationsから生成する。Work、solver time、end-to-end timeについて次を保存する。

- count、mean、sample standard deviation
- min、p50、p95、max
- 正規近似による95% confidence interval

平均値互換fieldは保持するが、研究分析ではraw observationsを正本とする。

## 8. 決定論性

`randomize_order`の並べ替えを含め、同一version、Scenario、seed集合ではtiming以外のRun定義と探索結果を再現可能にする。観測modeはpath、distance、found/exact、Work、Step、tie-breaking、seed消費順を変更してはならない。
