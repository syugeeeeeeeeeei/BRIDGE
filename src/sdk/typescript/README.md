# BRIDGE TypeScript SDK

Node.js 18以上で、BRIDGEをローカルSolverまたはHTTP Serverとして利用できます。

## インストール・ビルド

````bash
cd src/sdk/typescript
npm install
npm run build
````

## Local Solver

````typescript
import { BridgeClient } from "@bridge/route-sdk";
import { request } from "./examples/common.js";

const client = await BridgeClient.solver({ defaultTimeoutMs: 10000 });
const response = await client.route(request);
console.log(response.result.path);
````

## 起動済みServerへ接続

````typescript
const client = await BridgeClient.server("http://127.0.0.1:8080", {
  defaultTimeoutMs: 10000,
  headers: { "X-API-Key": "example" },
});
````

## SDK管理Server

````typescript
import { BridgeServer } from "@bridge/route-sdk";

const server = await BridgeServer.start();
try {
  const response = await (await server.client()).route(request);
} finally {
  await server.stop();
}
````

## タイムアウト・キャンセル

````typescript
const controller = new AbortController();
const promise = client.route(request, {
  timeoutMs: 10000,
  signal: controller.signal,
});
controller.abort();
await promise;
````

## 互換性

`verifyCompatibility`が正式オプションです。Capabilitiesに必要なRoute Schemaがあるかを確認します。

## Observation Level

正式値は`minimum`、`debug`、`trace`です。

## サンプル

`examples/`にSolver、HTTP、Managed Server、キャンセル、エラー処理の例があります。

共通仕様は`docs/sdk/SDK_OVERVIEW.md`を参照してください。
