# BEARING コンポーネント規則

**対象package:** `src/bridge/bearing`  
**対象版:** v0.15.0以降  
**状態:** 規範文書

## 1. 責務

- 型付きEvent envelope
- component、phase、epoch、task、action vocabulary
- Observer、DetailObserver
- NullObserver、SafeObserver

## 2. 禁止事項

- event収集・保存・replay・分析
- budget変更、cancel指示、solver選択
- ULTRASOUNDへの依存

## 3. 不変条件

- Observerは探索の副作用を持たない
- event sequenceは同一論理実行で再構成可能である
- Observer失敗を探索結果へ伝播させない。ただし観測失敗自体は診断可能にする
- Trace保存I/OをWorkへ含めない

## 4. 必須テスト

- Null/Safe Observerテスト
- Observer非干渉差分テスト
- event vocabulary validationテスト
