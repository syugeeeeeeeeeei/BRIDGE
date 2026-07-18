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
- 要求されていない高頻度Eventの属性を生成してはならない
- `off`またはDetailObserverが拒否したEventは、探索WorkごとのMap・Slice・文字列生成およびHeap Allocationを発生させない

## 4. 必須テスト

- Null/Safe Observerテスト
- Observer非干渉差分テスト
- event vocabulary validationテスト
- Observation無効時のAllocation非増幅テスト

## Lifecycle observation contract

- BEARING owns lifecycle event contracts, subscription checks, span identifiers, timestamp attachment through observers, and delivery.
- Producers MUST use `BeginLifecycle`/`EmitLifecycle` for low-frequency operation boundaries.
- The disabled path MUST NOT generate span identifiers, timestamps, payload maps, or trace objects.
- BEARING MUST NOT aggregate durations or alter algorithm decisions.

## Lifecycle emission cost

Lifecycle boundaries SHOULD use the stack-friendly `StartLifecycle` handle where practical. The disabled path MUST avoid timestamp lookup, span identifier creation, payload allocation, and dispatch. `BeginLifecycle` remains a compatibility helper and MUST NOT be introduced into high-frequency loops.
