# 離散Work・横比較評価報告

## 概要

ANCHOR、A*、双方向ダイクストラを同一の離散Action定義で計測した。対象は100、484、1024、2025、4900ノード、open、wall、U-shape、cul-de-sac、disconnected、seed 1～3である。

## 実装変更

- `Total Work`は標準Actionの整数合計を維持した。
- `0.45N`はANCHORの`EXPAND`上限として分離した。
- TRUSSのTotal Work BudgetとANCHORのExpand Budgetを独立させた。
- ANCHORをfirst-path優先の離散Anytime探索へ変更した。
- A*、双方向ダイクストラにも同じWorkMetricsを適用した。
- ULTRASOUNDへsearch開始・終了、node展開、incumbent更新のtraceを追加した。
- ULTRASOUNDへ決定論的な集計・validation機能を追加した。

## 接続グラフ60ケースの総合結果

| solver | 発見率 | 平均距離比 | 中央値 | p95 | 最悪 | 平均Total Work | 平均EXPAND |
|---|---:|---:|---:|---:|---:|---:|---:|
| ANCHOR | 100% | 1.007559 | 1.007354 | 1.010566 | 1.016102 | 917.9 | 70.6 |
| A* | 100% | 1.000000 | 1.000000 | 1.000000 | 1.000000 | 20,751.8 | 1,445.1 |
| 双方向ダイクストラ | 100% | 1.000000 | 1.000000 | 1.000000 | 1.000000 | 21,200.1 | 1,517.1 |

ANCHORはA*に対してTotal Workを95.6%、EXPANDを95.1%削減した。双方向ダイクストラに対してTotal Workを95.7%、EXPANDを95.3%削減した。代償は平均0.756%、最悪1.610%の距離超過である。

## 4900ノード

| solver | 発見率 | 平均距離比 | p95 | 平均Total Work | 平均EXPAND | 平均時間ms |
|---|---:|---:|---:|---:|---:|---:|
| ANCHOR | 100% | 1.008541 | 1.011364 | 1,836.0 | 139.0 | 0.266 |
| A* | 100% | 1.000000 | 1.000000 | 60,564.1 | 4,179.5 | 4.574 |
| 双方向ダイクストラ | 100% | 1.000000 | 1.000000 | 61,850.3 | 4,389.9 | 7.108 |

## Work内訳

ANCHORの平均Workでは`EVALUATE`と`RELAX`が各257.9で最大であり、次いで`ENQUEUE`156.9、`REJECT`102.0であった。A*と双方向ダイクストラでも`EVALUATE`、`RELAX`、`REJECT`が主要ボトルネックである。

この結果から、今後の最適化対象は係数調整ではなく、以下の離散Action削減へ一本化できる。

- 不要なedge評価を減らす。
- 非改善RELAXを事前に除外する。
- stale候補と重複ENQUEUEを減らす。
- corridor・detour仮説によりEXPAND対象を限定する。

## disconnected

ANCHORは`0.45N`展開上限で停止し、到達不能を証明しない。A*と双方向ダイクストラは探索空間を消尽して到達不能を判定した。したがって、到達不能証明はBOLTSの責務として維持する。

## 制約

- 現行実装は逐次実行であり、`Logical Steps = Scheduled Steps = Total Work`である。
- 今回のA*は位置情報を利用できるグラフを対象としている。
- ANCHORのfirst-path後のrepair・bound改善は限定的であり、完全なAnytime品質曲線は今後の課題である。
