# 開発タスク

> Status: Informative  
> Applies To: BRIDGE v0.15.x

BRIDGEの標準検査は次です。

```text
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
```

CLIは`src/products/cli/cmd/bridge`からビルドします。

```text
go build -o bridge ./src/products/cli/cmd/bridge
./bridge scenario validate tests/scenarios/<scenario>.json
./bridge benchmark run tests/scenarios/<scenario>.json
```

Scenarioの公開仕様は`docs/contracts/BENCHMARK_SCENARIO_CONTRACT.md`、成果物仕様は`docs/contracts/BENCHMARK_ARTIFACT_CONTRACT.md`を参照します。

詳細な利用手順は`docs/operations/DEVELOPMENT_OPERATION.md`に集約します。
