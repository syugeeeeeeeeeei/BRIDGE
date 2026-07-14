# Timing Regression Verification

## 条件

- graph: 10 x 10 open grid
- node count: 100
- algorithms: BRIDGE, ANCHOR, A*, Bidirectional Dijkstra
- observation: off
- warm-up: 2
- measured repetitions: 10
- seed: 7

## 合格条件

- 全本計測で `solver_time_ns > 0`
- 全本計測で `end_to_end_time_ns > 0`
- 全本計測で `timing_valid = true`
- `total_ns >= solver_ns`
- `gate_ns >= total_ns`
- `end_to_end_time_ns >= gate_ns`
- BRIDGEでは `solver_ns = anchor_ns + bolts_ns`
- ミリ秒値は対応するナノ秒値から導出される

## 結果

全条件に合格した。
