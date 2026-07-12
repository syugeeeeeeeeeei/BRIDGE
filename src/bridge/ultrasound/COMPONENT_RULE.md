# ULTRASOUND コンポーネント規則

**対象package:** `src/bridge/ultrasound`  
**状態:** 規範文書

## 1. 目的

ULTRASOUNDは、開発・検証用の観測・保存・分析を担当する。

## 2. 所有する責務

- BEARING event収集
- sequence、elapsed、delta付与
- Observation Modeによる選別
- Sink配送とtruncation管理
- Discard／Memory／Writer／File／Callback／Multi Sink
- JSONL等への保存
- metrics集計
- schema検証
- replayと分析

## 3. 禁止する責務

- solver選択
- budget再配分
- candidate選択
- graph変更
- 本番必須依存化

## 4. 依存規則

`BEARING`とread-onlyな`CORE` schemaに依存できる。

`others/legacy/bridge_py`へ依存してはならない。package間循環依存を作ってはならない。

## 5. Go実装規則

- 公開型・関数にはGoDocを付ける
- errorをpanicへ変換しない
- 大規模処理で不要なallocationを増やさない
- map iteration順に結果を依存させない
- WorkとStepは`docs/WORD_DEFINITION.md`の意味で計測する
- cancellationとdeadlineを区別する

## 6. 不変条件

- budget超過を発生させない
- 同一入力では決定論的な結果を返す
- observer有効・無効で探索結果を変えない
- public contractにprivate stateを漏らさない

## 7. 必須テスト

- 単体テスト
- budget境界テスト
- cancellationテスト
- 決定論性テスト
- architecture dependencyテスト
- 該当する場合はPython-Go paired test

## 8. 関連文書

- `docs/ARCHITECTURE_RULE.md`
- `docs/WORD_DEFINITION.md`
- `docs/architecture/BRIDGE_architecture_spec_v0.0.1.md`

## 9. Trace公開契約

- Traceは単なる診断logではなく、探索状態を復元できる棋譜でなければならない。
- 外部visualizerはBRIDGE内部packageをimportせず、保存済みJSONLのみでReplayできなければならない。
- Eventはsequence、経過時間、component、phase、Action、Work前後およびState Deltaを持つ。
- ULTRASOUNDは表示UIを所有しない。表示アプリは別projectが自由に実装できる。
- schema変更はversion管理し、未知Eventを無視できる後方互換性を優先する。
- Trace保存I/OをWorkへ含めてはならない。

## 10. Observation Mode

- `off`: event収集を行わない。
- `summary`: 主要counterの再構成に必要なeventのみを扱う。
- `trace`: replay可能なevent列を扱う。
- `profile`: profilingを伴う詳細観測用であり、純粋速度比較から分離する。
- 旧`metrics`および`debug` modeはv0.14.1で廃止する。

## Observation Mode contract
- `off`: eventを収集しない。
- `summary`: control/candidate eventを集計し、event stream I/Oを行わない。
- `trace`: replayに必要なcontrol/candidate/detail eventを保存する。
- `profile`: traceに加えて高頻度profile eventと観測コストを保存する。
- `overhead_ns`および`sink_write_ns`は探索Workへ含めない。
