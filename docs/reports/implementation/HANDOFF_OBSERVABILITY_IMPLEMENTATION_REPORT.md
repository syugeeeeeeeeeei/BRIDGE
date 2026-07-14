# Handoff・ボトルネック可観測性 実装レポート

## 実装内容

`core.HandoffMetrics`、`core.HandoffRecord`、`core.BottleneckProfile`を追加した。BRIDGE Route Result、GATE公開Result、TRAFFIC Benchmark Run、debug_summaryへ同一の型付き情報を伝播する。

実行成果物には`handoffs.csv`を追加し、同一Scenario・Graph Instance・Query・Seed・Repetitionの`weighted_astar` RunをBOLTS単体基準として自動結合する。

## 重要な修正

監査中に、BOLTSのWorkがBudget Ledgerでは消費される一方、BRIDGEの最終Workへ加算されていない会計不整合を発見した。BOLTS Work Metricsを集約Workへ追加し、Component別Work、Total Work、Budget Ledgerを一致させた。

## 出力項目

- handoff count / reason
- ANCHOR work at handoff
- BOLTS work / time
- available / transferred / reused state units
- pre-handoff waste work
- BOLTS standalone work / time
- additional work / time versus BOLTS standalone
- ANCHOR / BOLTS / TRUSS work
- ANCHOR / BOLTS / orchestration time
- epochs, max frontier, candidate updates, stagnation work
- dominant work / time component

## 検証

- `go test ./...`: 全成功
- debugモード、1000ノード、6トポロジー、6アルゴリズム: 全36 Run完走
- Handoff発生: maze、random-geometric、community、grid-wall、grid-u-shape
- `handoffs.csv`生成成功
- `runs.jsonl`の型付きHandoff/Bottleneckデータ確認済み

## 解釈上の注意

`available_state_units`、`transferred_state_units`、`reused_state_units`は現時点ではNode/Queue/Path単位の論理状態数であり、バイト数ではない。`pre_handoff_waste_work`はHandoff時点のANCHOR Workから、実際に再利用された状態単位を差し引いた診断指標である。完全なg-score/Open/Closed移管が実装された場合は、再利用Work換算モデルをさらに精密化する必要がある。
