# TRAFFIC コンポーネント規則

**対象package:** `src/bridge/traffic`  
**状態:** 規範文書

## 1. 目的

TRAFFICは、開発・検証用のテスト・benchmark基盤を担当する。

## 2. 所有する責務

- graph/query生成
- Golden/paired cases
- benchmark、stress、回帰
- Python-Go比較
- 完了基準判定

## 3. 禁止する責務

- solver内部状態の変更
- 同一実行中の探索制御介入
- private API依存
- 本番route処理への組込み

## 4. 依存規則

`GATE`公開API、`CORE`公開schema、`ULTRASOUND`公開artifact APIに依存できる。

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

## 9. 研究ベンチマーク契約

- Scenarioは宣言単位、Runは記録単位として区別する。
- Run identity、graph/query/environment metadata、raw observations、summary statisticsはTRAFFICが所有する。
- warm-upはraw Runとして記録するが、性能集計とacceptance判定から除外する。
- graph/query特性はTRAFFICが公開Graph契約から計算し、solver private stateを参照しない。
- Run順序のrandomizeはseedから決定論的に導出し、seed消費順を探索側で変更しない。

## Phase 5 dataset and statistics boundary

- TRAFFIC may load development/research datasets through `bridge.dataset.v1`.
- Dataset file access, preprocessing provenance, licensing metadata, and statistical comparison must not be moved into GATE, TRUSS, ANCHOR, BOLTS, or CORE.
- A dataset raw run must retain source, license, SHA-256, and preprocessing records.
- Statistical reports must operate on raw observations rather than pre-aggregated means.
