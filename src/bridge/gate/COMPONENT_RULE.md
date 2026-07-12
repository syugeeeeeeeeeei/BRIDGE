# GATE コンポーネント規則

**対象package:** `src/bridge/gate`  
**状態:** 規範文書

## 1. 目的

GATEは、外部公開境界を担当する。

## 2. 所有する責務

- 入力検証と既定値適用
- 外部IDと`NodeID`の変換
- 公開Go APIとCLI向け結果表現
- error mapping
- BEARING Observer契約の受け渡し

## 3. 禁止する責務

- ANCHOR/BOLTSの直接起動
- solver選択
- budget配分
- quality終了判定
- stdin、ファイル、HTTP等の外部I/O
- Collector、Sink、Trace保存先の生成

## 4. 依存規則

`CORE`、`TRUSS`の公開API、および`BEARING`のObserver契約にのみ依存する。`ULTRASOUND`の具体実装へ依存してはならない。

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
