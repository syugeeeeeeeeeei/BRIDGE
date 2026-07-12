# BRIDGE TypeScript SDK

BRIDGE 0.14.0の実行バイナリを静的に同梱し、Node.jsの子プロセスとして利用します。バイナリの自動ダウンロード、APIサーバー、暗黙のファイル生成は提供しません。

```ts
import { BridgeClient } from "@bridge/route-sdk";
const client = await BridgeClient.local();
const response = await client.route(request);
```

解決順は`binaryPath`、`BRIDGE_BINARY`、同梱バイナリ、PATHです。
