# BRIDGE TypeScript SDK

Node.js 18以上で、BRIDGEをローカルSolverまたはHTTP Serverとして利用できます。このSDKはGitHub Packagesのnpmレジストリから配布され、対応プラットフォーム向けのBRIDGE CLIバイナリをパッケージ内に同梱します。

## インストール

GitHub Packagesを参照する`.npmrc`をプロジェクトに置きます。

```ini
@syugeeeeeeeeeei:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${GITHUB_TOKEN}
```

GitHub Packagesからインストールします。

```bash
npm install @syugeeeeeeeeeei/bridge
```

公開先は`https://npm.pkg.github.com`のみです。npmjs.comへの公開、`postinstall`による外部バイナリ取得、利用者環境でのGoビルドは行いません。

## 対応OS・CPU

| OS | CPU | 同梱パス |
| --- | --- | --- |
| Linux | amd64 | `bin/linux-amd64/bridge` |
| Linux | arm64 | `bin/linux-arm64/bridge` |
| macOS | amd64 | `bin/darwin-amd64/bridge` |
| macOS | arm64 | `bin/darwin-arm64/bridge` |
| Windows | amd64 | `bin/windows-amd64/bridge.exe` |

未対応のOSまたはCPUでは、SDKは明確な`BridgeBinaryNotFoundError`を返します。SDKは同梱バイナリ、`binaryPath`、または`BRIDGE_BINARY`で明示されたバイナリだけを使用し、PATH上の不明な`bridge`は自動採用しません。

## Local Solver

```typescript
import { BridgeClient } from "@syugeeeeeeeeeei/bridge";
import { request } from "./examples/common.js";

const client = await BridgeClient.solver({ defaultTimeoutMs: 10000 });
const response = await client.route(request);
console.log(response.result.path);
```

## 起動済みServerへ接続

```typescript
const client = await BridgeClient.server("http://127.0.0.1:8080", {
  defaultTimeoutMs: 10000,
  headers: { "X-API-Key": "example" },
});
```

## SDK管理Server

```typescript
import { BridgeServer } from "@syugeeeeeeeeeei/bridge";

const server = await BridgeServer.start();
try {
  const response = await (await server.client()).route(request);
} finally {
  await server.stop();
}
```

## 開発とローカル検査

```bash
mise run sdk:typescript:release-package
```

このタスクは、クリーン、バイナリ生成、依存関係インストール、TypeScriptビルド、テスト、`npm pack`、tarball検査を同じ順序で実行します。検査対象はSDKディレクトリではなく、実際に公開される`.tgz`です。

## 公開手順

`project-version`で`package.json`、`package-lock.json`、BRIDGE本体バージョン、SDK定数を同じSemVerへ更新します。

```bash
mise run project-version
mise run project-version 0.15.4
mise run project-version add patch
mise run project-version sub minor
```

変更を`main`へマージし、`main`上で公開を起動します。

```bash
mise run publish
```

`publish`は現在バージョンから`vX.Y.Z`タグを作成してoriginへpushします。タグpush時のみGitHub Actionsが`GITHUB_TOKEN`でGitHub Packagesへ公開します。タグ、`package.json`、BRIDGEバイナリ、`binary-manifest.json`のバージョンが一致しない場合は失敗します。同一バージョンの再公開は行わず、修正時は新しいバージョンを発行します。

## 互換性

`verifyCompatibility`が正式オプションです。Capabilitiesに必要なRoute Schemaがあるかを確認します。Observation Levelの正式値は`minimum`、`debug`、`trace`です。

共通仕様は`docs/sdk/SDK_OVERVIEW.md`を参照してください。
