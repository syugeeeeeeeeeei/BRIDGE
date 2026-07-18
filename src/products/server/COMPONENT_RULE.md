# products/server コンポーネント規則

## 責務

HTTPをBRIDGE公開契約へ変換し、GATEを経由して探索を実行する製品アダプターです。

## 許可される依存

- `src/bridge/gate`
- Go標準ライブラリ

## 禁止事項

- TRUSS、ANCHOR、BOLTS、TRAFFICを直接呼び出すこと
- クライアント指定のサーバーローカルファイルを読むこと
- リクエスト固有のObserverや探索状態を共有すること
- 無制限の本文、同時実行、worker、Work Budgetを許可すること
