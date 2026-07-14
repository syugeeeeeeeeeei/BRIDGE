# BRIDGE Go v0.15.0

BRIDGEは、総Work Budgetの下で複数の探索仮説と局所solverをオンライン調停するAnytime経路探索エンジンです。

## 実行アーキテクチャ

```text
GATE → TRUSS → ANCHOR Session
              ↘ BOLTS Local Capability
```

TRUSSはepoch単位でtask、grant、並列度、Handoff、Evidence、終了条件を調停します。ANCHORは中断・再開可能な探索Sessionを提供し、BOLTSは局所接続、脱出、修復、到達不能証明、bound強化、候補認証を提供します。

Reachabilityは到達可能性を証明するsolverであり、重み付き最短路の最適性を証明するsolverではありません。

## リポジトリ構成

```text
BRIDGE/
├── src/       # 本体、製品CLI、SDK、公開契約
├── docs/      # 規範、仕様、監査、評価記録
├── tests/     # examples、scenarios、compatibility
├── others/    # legacy等の履歴資産
├── go.mod
├── mise.toml
└── README.md
```

## 基本コマンド

```bash
go test -count=1 ./...
go test -race -count=1 ./...
go build -o bridge ./src/products/cli/cmd/bridge
./bridge route --request tests/examples/route-request.json
```

## 規範文書

- アーキテクチャ: `docs/ARCHITECTURE_RULE.md`
- 用語: `docs/WORD_DEFINITION.md`
- 利用手順: `USAGE.md`
- コンポーネント規則: `src/bridge/*/COMPONENT_RULE.md`

## SDK

- `src/sdk/python`
- `src/sdk/typescript`

SDKは同梱CLIバイナリのstdin/stdout契約を使用します。BRIDGE本体はAPIサーバーを提供しません。
