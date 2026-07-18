# CLI Usage

> Status: Informative  
> Applies To: BRIDGE v0.15.x

```bash
go build -o bridge ./src/products/cli/cmd/bridge
./bridge route tests/examples/route-request.json
./bridge scenario validate <scenario.json>
./bridge benchmark run <scenario.json>
./bridge artifact evaluate <result.json>
```

正確なオプションは`./bridge <command> --help`で確認します。
