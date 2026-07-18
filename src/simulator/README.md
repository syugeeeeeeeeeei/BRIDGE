# BRIDGE Simulator

TRAFFICが生成したBenchmark Archiveを直接入力します。

```text
python src/simulator/simulate.py <execution-id>.zip
```

SimulatorはExecution manifest、`runs.jsonl`、各Runの`references`からGraph snapshotとTraceを自動検出します。個別パスやAlgorithm名の指定は主経路ではありません。

詳細:

- `docs/operations/SIMULATOR_OPERATION.md`
- `docs/contracts/SIMULATION_ARTIFACT_CONTRACT.md`
