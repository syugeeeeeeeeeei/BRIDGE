# Simulator Operation

> Status: Informative  
> Applies To: BRIDGE Simulator v0.15.x

```bash
python src/simulator/simulate.py <execution-id>.zip
```

出力先を変更する場合:

```bash
python src/simulator/simulate.py <execution-id>.zip --output-dir simulation-output
```

検証時など、生成量を制限する場合:

```bash
python src/simulator/simulate.py <execution-id>.zip \
  --frames 8 \
  --max-runs 2 \
  --duration-ms 80
```

- `--frames`: Runごとの最大GIFフレーム数。既定値は24。
- `--max-runs`: 処理するRun数の上限。`0`は全Run。
- `--duration-ms`: GIFの1フレーム表示時間。

Graph、Trace、Algorithm、Run directoryはBenchmark ArchiveのExecution manifest、`runs.jsonl`、Run referencesから自動検出されます。入力契約は`docs/contracts/SIMULATION_ARTIFACT_CONTRACT.md`を参照します。
