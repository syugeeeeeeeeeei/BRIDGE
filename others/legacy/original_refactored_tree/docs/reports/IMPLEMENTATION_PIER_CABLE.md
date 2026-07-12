# BRIDGE PIER・CABLE初期実装報告

## 実装版

- BRIDGE Python reference: 0.2.0-development
- PIER: 0.1.0
- CABLE: 0.1.0
- Legacy baseline: MRPC-DG6

## 今回実装した範囲

### BRIDGE Core

- `RouteRequest`を追加
- `QualityBounds`を追加
- `PathResult`へ以下を追加
  - `lower_bound`
  - `certified_ratio`
  - `quality_certified`
  - `first_path_work`
  - `first_path_time_ms`
  - `solver_trace`
  - `fallback_used`
  - `budget_exhausted`
  - `deadline_exceeded`
  - `error_code`
- 公開モードを`fast / balanced / quality / exact`へ統一
- `auto`を`balanced`の互換別名へ変更
- 旧solver名による明示実行を維持

### PIER v0.1

- MRPC-DG6をlegacy baselineとして保持
- `bridge_py.solvers.pier.pier`を追加
- PIER名称・バージョン・legacy名をtelemetryへ記録
- first-pathとrefinementのtelemetryを正規化
- conservative lower boundを導入
- certified ratioを導入
- 予測品質と証明品質を分離
- pathからdistanceを再計算し、DG6の距離不一致を補正

### CABLE v0.1

- query profilerを追加
- PIERを初期probeとして実行
- modeとPIER結果に応じて双方向DijkstraまたはA*を起動
- solver traceを記録
- shared upper boundを記録
- exact結果を品質証明済みとして統合
- deadline超過状態を記録
- portfolio overheadを記録

### 評価・テスト

- PIER telemetry test
- balanced mode contract test
- quality mode certification test
- exact mode certification test
- 既存12件を含む全16件のpytestを通過
- Python compileallを通過

## 検出・修正した既存不具合

MRPC-DG6の一部結果で、返却された`distance`と`path`の辺重み合計が一致しないケースを検出した。PIERでは全経路の距離を再計算し、不一致時は実経路距離へ補正する。

この補正がない場合、distance ratio、upper bound、certified ratioが不正確になるため、PIERの契約上は必須処理とした。

## スモーク評価

random geometric graph、各規模5 seed、balancedモードで確認した。

| ノード数 | found | 平均距離比 | 平均work比 / bidir | certified件数 |
|---:|---:|---:|---:|---:|
| 100 | 5/5 | 1.0421 | 0.1024 | 2/5 |
| 500 | 5/5 | 1.0593 | 0.0494 | 0/5 |
| 1,000 | 5/5 | 1.0715 | 0.0336 | 0/5 |

これは動作確認用の15 queryであり、仕様の受け入れ試験ではない。正式判定には30 seed以上、各graph 30 query以上、複数トポロジー、p95・p99、holdoutを使用する必要がある。

## 現時点の制限

- PIER内部のfirst path/refinementは、まだDG6内部実装を物理モジュール分割していない
- CABLEは逐次reference schedulerであり、solverの真の同時実行ではない
- deadlineは監視・記録のみで、実行中solverを強制中断できない
- work budgetはPIERへ反映されるが、portfolio全体の厳密なhard limitではない
- shared lower boundは未実装
- sampled geometric lower boundは安全性を優先して0を返すため、品質証明率が低い
- balancedモードの正式品質基準は未達成または未評価
- weighted-noise、scale-free、実道路の正式回帰評価は未実施
- memory budget enforcementは未実装
- solver progressに基づく途中再配分は未実装

## 次の実装単位

1. DG6本体を`first_path.py`、`refinement.py`、`hypotheses.py`、`repair.py`へ物理分割
2. 共通work schemaを全solverへ適用
3. budget-aware exact probeを実装
4. cancellation tokenとdeadline-aware停止を実装
5. shared lower boundとfrontier進捗を実装
6. 30 seed・複数query評価を標準化
7. weighted-noiseとscale-freeの回帰ケースを固定
8. CABLEの進捗ベース再配分を実装
