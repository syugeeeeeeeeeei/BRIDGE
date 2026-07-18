# Development Operation

> Status: Informative  
> Applies To: BRIDGE v0.15.x

```bash
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
```

契約変更時はSchema検証、Artifact bundle検証、Simulator ZIP入力テストも実行します。
