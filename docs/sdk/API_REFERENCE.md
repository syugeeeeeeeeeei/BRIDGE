# SDK APIリファレンス

## 共通Client

- `solver` / `local`: ローカルBRIDGEバイナリを利用
- `server` / `http`: HTTP Serverへ接続
- `version`: 接続先BRIDGEのversionを取得
- `capabilities`: 対応Schema・機能・アルゴリズムを取得
- `route`: 同期またはPromiseベースのRoute実行
- Python `route_async`: asyncio向けRoute実行

## Python

### `BridgeClient.solver`

- `binary_path`: BRIDGEバイナリの明示パス
- `default_timeout`: 既定処理時間、秒
- `verify_compatibility`: Capabilities検証

### `BridgeClient.server`

- `base_url`: `http://`または`https://`で始まるURL
- `default_timeout`: 既定通信時間、秒
- `headers`: 全HTTPリクエストへ追加するヘッダー
- `verify_compatibility`: Capabilities検証

### `BridgeServer.start`

- `binary_path`
- `host`
- `port`
- `config_path`
- `startup_timeout`
- `env`

## TypeScript

### `BridgeClient.solver`

`SolverOptions`:

- `binaryPath`
- `defaultTimeoutMs`
- `verifyCompatibility`

### `BridgeClient.server`

`ServerOptions`:

- `defaultTimeoutMs`
- `verifyCompatibility`
- `headers`

### `route`

`RouteOptions`:

- `timeoutMs`
- `signal`

### `BridgeServer.start`

`BridgeServerStartOptions`:

- `binaryPath`
- `host`
- `port`
- `configPath`
- `startupTimeoutMs`
- `env`
