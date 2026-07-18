# BRIDGE Go v0.15.x

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
├── docs/      # 規範契約、利用手順、履歴・評価記録
├── tests/     # examples、scenarios、compatibility
├── go.mod
├── mise.toml
└── README.md
```

## 基本コマンド

```bash
go test -count=1 ./...
go test -race -count=1 ./...
go build -o bridge ./src/products/cli/cmd/bridge
./bridge route tests/examples/route-request.json
./bridge serve
```

## 規範文書

- アーキテクチャ: `docs/ARCHITECTURE_RULE.md`
- 用語: `docs/WORD_DEFINITION.md`
- 文書索引: `docs/README.md`
- 利用手順: `USAGE.md`
- 成果物契約: `docs/contracts/BENCHMARK_ARTIFACT_CONTRACT.md`
- コンポーネント規則: `src/bridge/*/COMPONENT_RULE.md`

## SDK

- `src/sdk/python`
- `src/sdk/typescript`

SDKは同梱CLIバイナリのstdin/stdout契約を使用します。`bridge serve`により、GATE公開契約を使用するHTTP Serverも起動できます。ServerはBRIDGEのメインコンポーネントではなく、`src/products/server`に置かれる製品アダプターです。

- 性能回帰・分析運用基準（Medium通常正本・Large RC監査・ボトルネック分析）: `docs/PERFORMANCE_REGRESSION_POLICY.md`
