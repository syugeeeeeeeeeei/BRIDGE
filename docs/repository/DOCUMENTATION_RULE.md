# Documentation Rule

> Status: Normative  
> Applies To: BRIDGE repository

## 必須メタデータ

主要文書は`Status`、`Applies To`、`Owner`を冒頭に記載します。

## 更新単位

公開契約を変更する場合、同一変更で次を更新します。

- Producer実装
- Consumer実装
- JSON Schema
- `docs/contracts/`の契約文書
- `docs/operations/`の利用手順
- テスト

## 長期原則文書

`docs/project-knowledge/`には、BRIDGEの目的、責務境界、不変条件、正式用語、評価原則および変更判断基準を配置します。

- 頻繁に変わるコマンド、性能値、実装進捗は記載しません。
- 上位規範、正式用語集、公開契約と競合してはなりません。
- 実行境界または原理原則を変更した場合は、該当する上位規範とプロジェクト知識文書を同一変更で更新します。
- `BRIDGE_PROJECT_KNOWLEDGE.md`は分割文書から生成・同期される統合参照版として扱います。

## 禁止事項

- Historical文書を現行仕様として引用しません。
- 存在しないCLI、Path、Fieldを利用手順へ記載しません。
- 互換目的の廃止Fieldを現行契約へ残しません。
- Schema version、ファイル名、`$id`を異なる世代にしません。

## レビュー

文書内Pathの存在、Markdown link、Schema JSON構文、コード上のSchema定数との一致をCIで検査します。
