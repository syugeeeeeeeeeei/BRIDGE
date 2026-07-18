# 観測アーキテクチャ改修 実装報告

## 実装内容

- BEARINGへ型付きLifecycleイベント、Span ID、Parent Span IDを追加した。
- `BeginLifecycle`は購読されている場合だけSpanを生成し、無効時はPayload・時刻・Spanを生成しない。
- ULTRASOUNDへ開始・終了イベントの対応付け、Duration算出、異常Span検出を追加した。
- minimum観測でも低頻度Lifecycleイベントのみを取得できるようにした。
- GATEへRoute、Graph Build、Validation、Dispatch、Result Conversionの境界を追加した。
- TRUSSへRoute、Result Integrationの境界を追加した。
- ANCHORと主要BOLTS探索へSolver境界を追加した。
- 公開Observation結果へSpan集計を追加した。
- BOLTSに残っていた無効時の開始・終了・Node Expanded Payload先行生成を抑止した。

## 互換移行

既存の`TimeBreakdown`直接計測は、新しいSpan値との比較監査が完了するまで維持している。正式な時間集計元は段階的にULTRASOUND Spanへ移行し、差異が許容範囲内であることを確認後、観測目的だけの直接計測を削除する。

## 検証

- `go test -count=1 ./...`: PASS
- minimum観測でLifecycle Spanが完結すること: PASS
- NullObserverでSpanを生成しないこと: PASS
- GATE公開結果へSpan集計が含まれること: PASS
