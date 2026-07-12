# BRIDGE Python SDK

BRIDGE 0.14.0の実行バイナリを静的に同梱し、子プロセス経由で経路探索を行います。外部ダウンロードや暗黙のファイル生成は行いません。

```python
from bridge_sdk import BridgeClient
client = BridgeClient.local()
response = client.route(request)
print(response.result)
```

バイナリ解決順は、`binary_path`、`BRIDGE_BINARY`、同梱バイナリ、PATHです。
