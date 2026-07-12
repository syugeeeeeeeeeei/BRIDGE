# Go移植完了の定量基準

## 必須ゲート

| 指標 | しきい値 |
|---|---:|
| valid path rate | 1.000 |
| connected found rate | 0.990以上 |
| 平均Distance Ratio | 1.05以下 |
| p95 Distance Ratio | 1.15以下 |
| worst Distance Ratio | 1.35以下 |
| exact mode exact率 | 1.000 |
| budget violation率 | 0.000 |
| repeatability率 | 1.000 |
| Python-Go複合傾向相関 | 0.70以上 |
| topology coverage | 0.90以上 |
| found判定一致率 | 0.99以上 |

## 評価方針

完全な値一致ではなく、algorithm、architecture、data structureが同等の問題特性へ反応しているかを評価する。Work定義が異なる期間は絶対値ではなく順位相関を使用する。

## 現行v0.5.0

- 75 paired cases
- valid path rate: 1.0000
- connected found rate: 1.0000
- 平均Distance Ratio: 1.002094
- p95: 1.005402
- worst: 1.007180
- topology coverage: 1.0000
- found一致率: 1.0000
- 複合傾向相関: 0.7075

判定: `migration_complete: true`
