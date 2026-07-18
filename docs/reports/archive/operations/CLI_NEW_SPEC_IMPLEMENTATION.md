# BRIDGE CLI 新仕様 実装概要

## 正式コマンド

- `bridge route <request.json>`
- `bridge serve`
- `bridge serve init|validate|show`
- `bridge scenario init|validate|inspect|list-presets`
- `bridge benchmark run|list`
- `bridge artifact inspect|validate|evaluate`
- `bridge schema list|show`
- `bridge capabilities`
- `bridge completion`
- `bridge version`

## ひな形

`bridge serve init`と`bridge scenario init`は日本語コメント付きYAMLを生成します。`--comments full|summary|none`は説明量だけを変更し、コメントを除いた設定値は同一です。

## Server

`bridge serve`は既定で`127.0.0.1:8080`へbindし、`POST /v1/routes`、`GET /v1/capabilities`、`GET /healthz`、`GET /readyz`を提供します。HTTP入力はinline graphのみ受理します。

## 廃止した入口

- `bridge benchmark <scenario>`
- `bridge benchmark validate`
- `bridge health check`
- `bridge route --request`
