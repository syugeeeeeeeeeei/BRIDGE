# BOLTS リファレンス最適化 実装チェックリスト

## 正しさ・参照性
- [x] Dijkstra の優先度規則を累積距離 `g` のみに維持
- [x] A* の優先度規則を `g+h` に維持
- [x] Weighted A* の優先度規則を `g+w*h` に維持
- [x] Bidirectional Dijkstra の前向き・逆向き距離と停止条件を維持
- [x] Reachability を重み非依存 BFS として維持
- [x] 非負重みランダム有向・無向 Graph で A* と双方向 Dijkstra を Dijkstra Oracle と照合
- [x] 到達不能・Budget・終了状態の契約を統一
- [x] Work Action の重み付けおよび定義を変更しない

## 最適化
- [x] 通常 Dijkstra/A*/Weighted A* の Queue Stale 判定を距離ベースへ統一
- [x] debug/state-delta が不要な実行で investigated edge Map を生成しない
- [x] Bidirectional Dijkstra の展開方向を固定交互から最小 Frontier Key 側へ変更
- [x] Bidirectional Dijkstra の Queue Stale 判定を距離ベースへ統一
- [x] Reachability Queue を Node 数容量で事前確保
- [x] 初期 Queue Push を QueuePushes Telemetry に含める
- [x] Reachability の到達不能証明を LocalExecutor Evidence へ正しく昇格

## 監査
- [x] `go test -count=1 ./...` 全成功
- [x] 旧版・最適化版を同一 32x32 Grid、同一 Query、NullObserver で比較
- [x] ns/op・B/op・allocs/op を記録
- [x] 経路発見結果と距離の非回帰を確認
- [x] 未実装事項を完了扱いにしない

## 今回の範囲外
- [ ] 通常探索と Seeded 探索の単一 Core への完全統合
- [ ] Reverse Graph の Graph ライフサイクルキャッシュ
- [ ] 外部ライブラリ実装との Runtime 比較
- [ ] decrease-key 対応 Priority Queue との比較
