# BOLTS リファレンス最適化 実装・監査報告

## 目的

BOLTS の Dijkstra、Bidirectional Dijkstra、A*、Weighted A*、Reachability を、名称に対応する標準的な探索規則と Work Action 契約を維持したまま高速化した。アルゴリズムを BRIDGE に有利な別物へ変更せず、参照実装として説明可能な最適化だけを適用した。

## 実装内容

### Dijkstra / A* / Weighted A*

- Queue Entry の `distance` と現在の `dist[node]` を比較する Stale Entry 判定へ統一した。
- debug または state-delta 観測が不要な場合、調査 Edge ID 用 Map を作成しない。
- 初期 Queue Push を QueuePushes へ含めた。
- Found、Unreachable、Budget Exhausted の終了契約を明示した。
- A* weight=1 の完走結果を Exact として扱うよう結果契約を修正した。

### Bidirectional Dijkstra

- 前向き・後ろ向きを固定交互に展開する方式から、Queue 最小 Key が小さい側を展開する標準的な選択へ変更した。
- 距離ベース Stale Entry 判定へ統一した。
- debug 不要時の Edge 診断 Map を省略した。
- 終了状態、到達不能証明、Budget 終了契約を統一した。

### Reachability

- FIFO Queue を Graph Node 数で事前確保し、成長時 Allocation を削減した。
- 到達不能を完全探索で証明した場合、LocalExecutor が `ProofUnreachable` Evidence を生成できるよう修正した。

## 正しさ検証

40 Seed、8〜31 Node、非負重みの有向・無向ランダム Graphを生成し、各 Graph 12 Queryについて Dijkstra を Oracle として比較した。

- A*: Found 判定・距離とも全件一致
- Bidirectional Dijkstra: Found 判定・距離とも全件一致
- 到達不能終了契約: Dijkstra、A*、Bidirectional Dijkstraで一致
- Reachability Evidence: 到達不能時に ProofUnreachable を生成

`go test -count=1 ./...` は全パッケージ成功した。

## Microbenchmark

条件:

- 32x32 無向 Grid、1024 Node
- Source=0、Target=1023
- Edge Weight=1
- NullObserver
- 旧版と最適化版で同一 Benchmark Code
- Dijkstra/A*/Weighted A*/Bidirectional: 100回 × 3測定
- Reachability: 1000回 × 5測定

| Solver | 旧 ns/op | 新 ns/op | 時間改善 | 旧 B/op | 新 B/op | Memory改善 |
|---|---:|---:|---:|---:|---:|---:|
| Dijkstra | 1,672,347 | 787,109 | 52.93% | 676,395 | 449,665 | 33.52% |
| A* | 1,581,180 | 798,192 | 49.52% | 676,908 | 450,105 | 33.51% |
| Weighted A* | 1,395,924 | 731,895 | 47.57% | 615,256 | 392,876 | 36.14% |
| Bidirectional Dijkstra | 2,086,607 | 1,108,733 | 46.86% | 1,215,731 | 690,788 | 43.18% |
| Reachability | 122,136 | 120,511 | 1.33% | 19,579 | 10,754 | 45.07% |

主要な時間改善は、minimum/NullObserver 経路で不要な診断用 Map と大量の Edge ID Allocation を行わなくした効果である。Work Action 数や探索規則を軽く見せる変更ではない。

## リファレンス性の判断

今回の最適化後も各 Solver は次の規則を維持する。

- Dijkstra: `priority=g`
- A*: `priority=g+h`
- Weighted A*: `priority=g+w*h`
- Bidirectional Dijkstra: 前後 Dijkstra と `minForward+minBackward>=best` 停止
- Reachability: FIFO BFS

したがって、BOLTS 内部参照実装としてのリファレンス性は改善した。成果物では Weight、Heuristic、Tie Break、Reopen/Stale Policy、Instrumentation Modeを必ず記録する必要がある。

## 残課題

通常探索と Seeded Weighted A* は依然として別 Loop である。Parity Test は存在するが、長期的には単一 Search Core へ統合すべきである。また Bidirectional Dijkstra の Reverse Graph はRunごとに構築されるため、同一 Graph 多数 Query の測定では前処理込み・探索のみを分離する余地がある。
