# Handoff・ボトルネック可観測性 実装チェックリスト

## Handoff
- [x] 発生回数を型付きデータで出力
- [x] 発生理由をRun単位・Handoff単位で出力
- [x] Handoff時点のANCHOR Workを出力
- [x] BOLTS Workを出力
- [x] BOLTS実行時間をns/msで出力
- [x] 引継ぎ可能状態量を出力
- [x] 実際に転送した状態量を出力
- [x] 再利用された状態量を出力
- [x] Handoff前の無駄Workを出力
- [x] 同一Graph・Query・Seed・反復のBOLTS単体Workと比較
- [x] BOLTS単体との追加Work・追加時間を出力
- [x] HandoffなしRunでは不要な空レコードを出力しない

## ボトルネック
- [x] ANCHOR/BOLTS/TRUSS別Workを出力
- [x] ANCHOR/BOLTS/Orchestration別時間を出力
- [x] Epoch数を出力
- [x] 最大Frontierを出力
- [x] Candidate更新回数を出力
- [x] Candidate更新後の停滞Workを出力
- [x] 支配的Work Componentを判定
- [x] 支配的Time Componentを判定
- [x] debug_summaryへ型付き情報を反映

## 成果物・会計
- [x] runs.jsonlへhandoff_metricsを保存
- [x] runs.jsonlへbottleneck_profileを保存
- [x] handoffs.csvを自動生成
- [x] BOLTS WorkをBRIDGE Total Workへ正しく加算
- [x] Budget LedgerとTotal Workを一致
- [x] 全Goテスト成功
- [x] 1000ノード6トポロジーで再ベンチマーク完走
