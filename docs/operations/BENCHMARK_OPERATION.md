# Benchmark Operation

> Status: Informative  
> Applies To: BRIDGE v0.15.x

## 実行

```bash
go build -o bridge ./src/products/cli/cmd/bridge
./bridge scenario validate tests/scenarios/<scenario>.json
./bridge benchmark run tests/scenarios/<scenario>.json
```

完了時、CLIは展開済み成果物の`artifacts:`とZIPの`archive:`を表示します。成果物の意味は`docs/contracts/BENCHMARK_ARTIFACT_CONTRACT.md`を参照します。

## 評価

- `result.json`と`runs.jsonl`を正本として使用します。
- `summary.csv`は閲覧・表計算用の派生物です。
- `healthy.json`のfatal判定があるBundleを比較評価へ使用しません。
