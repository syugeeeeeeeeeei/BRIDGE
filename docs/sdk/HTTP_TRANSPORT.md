# HTTP Transport

起動済みの`bridge serve`へ接続します。Routeは`POST /v1/routes`、互換性確認は`GET /v1/capabilities`を使用します。

- Python: `BridgeClient.server(base_url, ...)`
- TypeScript: `BridgeClient.server(baseUrl, ...)`

タイムアウトはSDK側の通信期限です。探索リクエスト内の`budget.timeout_ms`とは別であり、短い側で処理が終了する場合があります。

カスタムヘッダーは、Pythonの`headers`、TypeScriptの`headers`で指定できます。認証情報をログへ出力しないでください。
