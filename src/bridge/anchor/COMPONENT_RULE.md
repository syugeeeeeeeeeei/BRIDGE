# ANCHOR コンポーネント規則

**対象package:** `src/bridge/anchor`  
**状態:** 規範文書

## 1. 目的

ANCHORは、BRIDGE固有の主Anytime探索を担当する。

## 2. 所有する責務

- HypothesisとCorridor探索
- first path生成
- Candidate生成と比較
- local Repair
- Work/Step報告

## 3. 禁止する責務

- portfolio全体予算の変更
- fallback solver選択
- 未証明exactの主張
- ULTRASOUND直接依存

## 4. 依存規則

`CORE`と`BEARING`に依存できる。補助機能は注入されたportを介する。

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

## Coordination rule

ANCHOR reports progress and emergency evidence to TRUSS through CORE coordination
contracts. ANCHOR does not select, own, or directly invoke BOLTS in production orchestration.
Legacy injected connector support is retained temporarily for compatibility only and MUST
not be wired by TRUSS.
