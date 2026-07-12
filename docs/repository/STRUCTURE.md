# リポジトリ構造

```text
BRIDGE/
├── src/
│   ├── bridge/       # Go本体
│   ├── products/     # 正式CLI等の配布製品
│   ├── sdk/          # Python／TypeScript SDK
│   ├── contracts/    # JSON Schema等の公開契約
│   └── internal/     # 現行製品だけが使う非公開実装
├── docs/             # 規範、仕様、計画、報告、利用手順
├── tests/
│   ├── examples/     # 入力例
│   ├── scenarios/    # ベンチマークScenario
│   └── compatibility/# Python参照版との比較
├── others/
│   └── legacy/       # 旧実装、旧CLI、履歴資産
├── go.mod
├── mise.toml
└── README.md
```

## 配置規則

- 現行の実行コード、SDK、契約は`src/`へ置く。
- テスト入力、Scenario、互換評価は`tests/`へ置く。
- 旧実装および正式経路から参照しない履歴資産は`others/`へ置く。
- 現行製品がimportするGo internal packageは、Goの可視性規則に従い`src/internal/`へ置く。
- ルートにはmodule定義、環境設定、入口READMEだけを置く。
