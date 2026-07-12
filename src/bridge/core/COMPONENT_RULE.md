# CORE コンポーネント規則

**対象package:** `src/bridge/core`  
**状態:** 規範文書

## 1. 目的

COREは、共有値型と中立的なデータ契約を担当する。

## 2. 所有する責務

- `NodeID`、`Edge`、Graph契約
- `RouteRequest`、`RouteResult`
- `WorkMetrics`、Budget、Bounds
- cancellationとerror codeの中立表現

## 3. 禁止する責務

- solver選択
- portfolio制御
- trace保存
- CLI/JSON変換
- algorithm固有heuristic

## 4. 依存規則

Go標準ライブラリのみを原則とする。ほかのBRIDGE componentへ依存しない。

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
