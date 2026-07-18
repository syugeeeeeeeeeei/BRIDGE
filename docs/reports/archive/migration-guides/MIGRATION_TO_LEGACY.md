# Python版のlegacy移行規則

旧`bridge_py`は`others/legacy/bridge_py`へ移動済みである。

## 保持目的

- Go移植差分の確認
- Golden/paired fixture生成
- 過去benchmarkの再現
- algorithm履歴の保存

## 禁止事項

- Go本番packageからの依存
- 新機能の追加
- 通常CLI実行への組込み
- legacy testの既定収集

Go版で再実装した機能は、現行component ruleとGo data structureへ適合させる。
