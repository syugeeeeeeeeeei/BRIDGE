# SDK管理Server

`BridgeServer.start()`はローカルの`bridge serve`を子プロセスとして起動します。

- 既定hostは`127.0.0.1`
- port未指定時は空きポートを選択
- `/readyz`が成功するまで待機
- Pythonではcontext managerを推奨
- TypeScriptでは`finally`で`stop()`を必ず実行

外部公開、常駐サービス、複数ホスト運用にはSDK管理Serverではなく、運用基盤から`bridge serve --config ...`を起動してください。
