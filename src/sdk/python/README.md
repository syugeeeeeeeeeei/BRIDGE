# BRIDGE Python SDK

Python 3.10以上で、BRIDGEをローカルSolverまたはHTTP Serverとして利用できます。

## インストール

````bash
python -m pip install -e ./src/sdk/python
````

## 利用方式

### Local Solver

単発処理や簡単なローカル組込み向けです。呼出しごとにBRIDGEプロセスを起動します。

````python
from bridge_sdk import BridgeClient
from examples.common import REQUEST

client = BridgeClient.solver(default_timeout=10)
response = client.route(REQUEST)
print(response.result["path"])
````

### 起動済みServerへ接続

高頻度呼出しや複数クライアント共有向けです。

````python
client = BridgeClient.server(
    "http://127.0.0.1:8080",
    default_timeout=10,
    headers={"X-API-Key": "example"},
)
response = client.route(REQUEST)
````

### SDK管理Server

ローカルテストや一時的な統合処理向けです。

````python
from bridge_sdk import BridgeServer

with BridgeServer.start() as server:
    response = server.client().route(REQUEST)
````

## 非同期処理とキャンセル

````python
response = await client.route_async(REQUEST, timeout=10)
````

呼出しTaskをキャンセルすると、ローカルBRIDGEプロセスも停止します。

## 互換性

`verify_compatibility=True`が既定です。Capabilitiesに必要なRoute Schemaがあるかを確認します。

## エラー

主要例外:

- `BridgeValidationError`
- `BridgeIOError`
- `BridgeTimeoutError`
- `BridgeCancelledError`
- `BridgeAcceptanceError`
- `BridgeProtocolError`
- `BridgeVersionError`
- `BridgeInternalError`

`path_found=false`やBudget到達は例外ではなく、正常なRoute結果です。

## サンプル

`examples/`にSolver、HTTP、Managed Server、async、エラー処理の実行例があります。

共通仕様は`docs/sdk/SDK_OVERVIEW.md`を参照してください。
