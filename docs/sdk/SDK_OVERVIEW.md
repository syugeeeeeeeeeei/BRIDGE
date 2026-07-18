# BRIDGE SDK 概要

BRIDGE SDKは、同一のRoute契約を次の3方式で利用します。

| 方式 | 用途 | 特徴 |
|---|---|---|
| Local Solver | 単発処理、ローカルスクリプト | 呼出しごとに`bridge route`を起動 |
| HTTP Client | 高頻度処理、複数プロセス共有 | 起動済み`bridge serve`へ接続 |
| Managed Server | テスト、一時的な統合処理 | SDKがローカルServerを起動・停止 |

高頻度または複数クライアントで利用する場合はHTTP Clientを推奨します。Managed Serverはローカル開発・テスト向けであり、外部公開運用にはServer設定、認証、TLS、ネットワーク制限を別途用意してください。

関連文書:

- `LOCAL_TRANSPORT.md`
- `HTTP_TRANSPORT.md`
- `MANAGED_SERVER.md`
- `COMPATIBILITY.md`
- `ERROR_MODEL.md`
