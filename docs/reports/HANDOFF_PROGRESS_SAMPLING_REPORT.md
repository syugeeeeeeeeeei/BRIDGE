# Handoff早期化データ取得 実装・検証レポート

## 実装範囲

- debug実行時だけ、64 Work間隔の低頻度進捗サンプルを収集
- サンプル項目は以下に限定
  - work
  - candidate_found
  - works_since_candidate_update
  - frontier_size
  - reject_rate
  - best_heuristic
  - lower_bound
- へ型付き保存
- Scenarioのを追加
- 明示閾値指定時のみepochを閾値境界で分割
- minimumではサンプル収集なし
- Trace Event全件保存、Queue内容、Node履歴は追加していない

## 検証

- ?   	github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing	[no test files]
ok  	github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/anchor	(cached)
ok  	github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bolts	(cached)
ok  	github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core	(cached)
ok  	github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate	(cached)
ok  	github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/healthy	0.038s
ok  	github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic	0.074s
ok  	github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/truss	(cached)
ok  	github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/ultrasound	(cached)
ok  	github.com/syugeeeeeeeeeei/BRIDGE/src/internal/yamlmini	(cached)
ok  	github.com/syugeeeeeeeeeei/BRIDGE/src/products/cli/cmd/bridge	(cached): 全成功
- debug Smoke Benchmark: 全8 Run成功
- 閾値Sweep: 64 / 128 / 256 Workで全3 Run成功

## 閾値Sweep結果

| Scenario | Handoff時Work | BOLTS Work | Total Work | Sample数 | Path Found |
|---|---:|---:|---:|---:|---:|
| grid-wall-100-th64 | 64 | 63 | 639 | 8 | true |
| grid-wall-100-th128 | 128 | 63 | 704 | 10 | true |
| grid-wall-100-th256 | 256 | 63 | 830 | 11 | true |

早いHandoffほど、この条件ではTotal Workが小さいことを確認した。ただし、これは単一条件の結果であり、採用閾値はトポロジー・規模・Seed・Queryを広げたSweepで決定する必要がある。

## 取得例

64 Work時点:

- Candidate未発見
- Frontier Size: 6
- Reject Rate: 0.353
- Best Heuristic: 9.220
- Lower Bound: 21.720

これにより、Handoff前の単一最終値だけでなく、進捗の変化率を事後計算できる。

## 完了判定

必要十分な計測項目の実装と、望むデータが取得できることの検証は完了した。Handoff判定ロジック自体の最適化は、本実装で得られるSweepデータを用いる次段階の課題とする。
