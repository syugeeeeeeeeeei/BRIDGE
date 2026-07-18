# リポジトリ構造

```text
BRIDGE/
├── src/
│   ├── bridge/       # Go本体
│   ├── products/     # 正式CLI等の配布製品
│   ├── sdk/          # Python／TypeScript SDK
│   ├── contracts/    # JSON Schema等の公開契約
│   └── internal/     # 現行製品だけが使う非公開実装
├── docs/             # 規範、長期原則、仕様、計画、報告、利用手順
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
- 長期維持する思想、責務境界、不変条件、評価原則は`docs/project-knowledge/`へ置く。
- テスト入力、Scenario、互換評価は`tests/`へ置く。
- 旧実装および正式経路から参照しない履歴資産は`others/`へ置く。
- 現行製品がimportするGo internal packageは、Goの可視性規則に従い`src/internal/`へ置く。
- ルートにはmodule定義、環境設定、入口READMEだけを置く。

## コンポーネントとサブコンポーネント

- `src/bridge/<name>`の大文字名で識別される責務境界をメインコンポーネントとする。`CORE`は共有基盤として規則管理する例外である。
- メインコンポーネント内部の責務別DirectoryまたはFile群をサブコンポーネントとする。サブコンポーネントは親の`COMPONENT_RULE.md`を継承する。
- `src/products/`はCLI・HTTP Server等の製品境界であり、メインコンポーネントではない。
- `src/contracts/`は機械可読な公開契約、`src/internal/`はZIP、設定codec、build情報等の非公開技術基盤を配置する。
- 責務分割だけの変更では公開APIと探索挙動を変更せず、変更前後の決定的ベンチマーク結果を一致させる。

## 現行の責務別物理分割

- CLI: app、route、benchmark、serve、scenario、artifact、metadata
- TRAFFIC: scenario model、generator、runner、metrics、artifact
- GATE: contracts、router、observation、graph mapper
- HEALTHY: profile、analysis service、validation、oracle、comparison、evaluation
- Server: config、lifecycle、handlers、middleware
- CORE: graph、route、errors、work、metrics、solver

詳細は`docs/architecture/RESPONSIBILITY_REFACTORING_RULES.md`と`docs/operations/RESPONSIBILITY_REFACTORING_RESULT.md`を参照する。
