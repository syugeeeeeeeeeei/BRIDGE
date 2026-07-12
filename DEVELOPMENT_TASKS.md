# 開発タスク

BRIDGEの開発コマンドはmiseを入口とし、複雑な処理はクロスプラットフォームなPythonスクリプトで実行します。shell scriptおよびPowerShell scriptは使用しません。

## 構成

```text
mise.toml
├── toolsのバージョン
└── tasks/*.tomlのinclude

tasks/
├── mise.tasks.setup.toml
├── mise.tasks.build.toml
├── mise.tasks.sdk.toml
├── mise.tasks.test.toml
└── mise.tasks.verify.toml

others/tools/dev_tasks.py
└── OS差、クロスビルド、コピー、manifest生成、完全検査
```

## 初期設定

```text
mise install
mise run setup
```

## BRIDGEバイナリ

```text
mise run build
mise run build:debug
mise run build:release
```

出力はWindowsでは`build/bridge.exe`、LinuxおよびmacOSでは`build/bridge`です。

## SDK

```text
mise run sdk:binaries
mise run sdk:python:package
mise run sdk:typescript:package
mise run sdk:package
```

`sdk:binaries`は5環境分をクロスビルドし、Python SDKとTypeScript SDKの`bin/`および`binary-manifest.json`を更新します。

## テスト

```text
mise run test
mise run test:go
mise run test:race
mise run test:vet
mise run test:cli
mise run test:python
mise run test:typescript
mise run test:compatibility
mise run test:benchmark
mise run test:benchmark:smoke
mise run test:benchmark:algorithms
mise run test:benchmark:system
mise run test:benchmark:robustness
```

## ベンチマーク

ベンチマークは専用taskから、シナリオファイルとCLIオプションをそのまま渡します。

```text
mise run benchmark ./tests/scenarios/smoke.yaml
mise run benchmark ./tests/scenarios/algorithm-benchmark.yaml --format csv --output ./build/algorithm-benchmark.csv --overwrite
mise run benchmark ./tests/scenarios/system-benchmark.yaml --format json --output ./build/system-benchmark.json --overwrite
mise run benchmark ./tests/scenarios/robustness.yaml --trace-dir ./build/traces --trace-overwrite
mise run benchmark ./tests/scenarios/stress-large.yaml --format csv --output ./build/stress-large.csv --overwrite
```

用意済みのシナリオは `tests/scenarios/` にある。大規模ストレス用は `stress-large.yaml` を使用する。

scenarioの公開仕様は `docs/architecture/BENCHMARK_SCENARIO_SPEC_v1.md` を参照する。

## 完全検査

```text
mise run verify
mise run verify:quick
```

`verify`はSDK同梱バイナリを更新してから、Goテスト、race detector、vet、CLIスモークテスト、Python SDK、TypeScript SDK、互換性検証を実行します。`verify:quick`はSDKバイナリ再生成だけを省略します。

## その他

```text
mise tasks
mise tasks deps verify
mise run clean
```
## Windowsでのnpm実行

開発タスクはPythonから外部コマンドを実行する前にPATHを検索し、Windowsでは`npm.cmd`や`npx.cmd`の実体を解決します。`shell=True`、PowerShellスクリプト、cmdスクリプトには依存しません。

タスク一覧の表示には、引数なしの`mise run`ではなく次を使用してください。

```text
mise tasks
```

タスク実行時は名前を明示します。

```text
mise run sdk:typescript:package
```

