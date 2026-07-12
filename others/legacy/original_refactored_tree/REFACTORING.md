# BRIDGE refactoring record

## 実施内容

- packageを`src/bridge_py`へ移行し、テスト・ツール・生成物との境界を固定
- `Graph`と`PathResult`等の共通型を`core`へ集約
- 旧`bridge_py.graph`と`bridge_py.types`は互換re-exportへ縮小
- TRUSSの単一巨大処理を以下へ分割
  - `truss/profile.py`: query profile
  - `truss/models.py`: task trace schema
  - `truss/orchestrator.py`: portfolio orchestration
- GATEのrequest組立を`gate/request_factory.py`へ分離
- benchmark実装を`traffic`へ移動し、旧`bench`は互換shim化
- CABLEの重複query profilerを除去し、TRUSS profilerの互換re-exportへ変更
- root直下のbenchmark・profiling scriptを`tools/`へ移動
- reportを`docs/reports/`、測定結果を`artifacts/evaluation_results/`へ移動
- testを`architecture`、`integration`、`unit`へ分類
- `.pytest_cache`、`__pycache__`、`.pyc`を成果物から除外

## 互換性

- `bridge_py.route`、`bridge_py.path`、旧solver importを維持
- `bridge_py.graph`、`bridge_py.types`を互換moduleとして維持
- `bridge_py.bench.*`を`bridge_py.traffic.*`への互換shimとして維持
- CABLE metadataは既存利用者向けに残すが、制御実装はTRUSSのみが所有

## 検証

- pytest: 20 passed
- compileall: 成功
- architecture dependency test: 合格
- ULTRASOUND ON/OFF非干渉test: 合格
