# BRIDGE Go v0.14.0

BRIDGEは、予算管理型Anytime経路探索エンジンです。

## リポジトリ構成

```text
BRIDGE/
├── src/       # BRIDGE本体、製品CLI、SDK、公開契約
├── docs/      # 規範、仕様、利用手順、計画、履歴
├── tests/     # examples、scenarios、compatibility等の評価資産
├── others/    # legacy等の非正式・履歴資産
├── go.mod
├── mise.toml
└── README.md
```

## 実行

```bash
go test ./...
go build -o bridge ./src/products/cli/cmd/bridge
./bridge route --request tests/examples/route-request.json
```

## SDK

- `src/sdk/python`: Python SDK
- `src/sdk/typescript`: Node.js向けTypeScript SDK
- 両SDKにLinux、Windows、macOS向けBRIDGEバイナリを静的同梱
- バイナリの自動ダウンロードなし
- APIサーバーはSDK利用者が任意のフレームワークで構築

規範は`docs/ARCHITECTURE_RULE.md`、利用手順は`docs/USAGE.md`を参照してください。

## 開発タスク

開発コマンドはmiseへ統一しています。ツール定義は`mise.toml`、タスク定義は`tasks/mise.tasks.*.toml`へ分離しています。

```text
mise install
mise run setup
mise run build
mise run sdk:binaries
mise run test
mise run verify
```

詳細は`docs/DEVELOPMENT_TASKS.md`を参照してください。
