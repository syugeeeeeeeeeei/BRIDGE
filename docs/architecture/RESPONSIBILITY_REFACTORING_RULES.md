# 責務分離リファクタリング規則

## 目的

既存のメインコンポーネントを維持したまま、各コンポーネント内の責務をサブコンポーネント単位へ分割する。新しい橋名メインコンポーネントは追加しない。

## 不変条件

- ANCHOR、TRUSS、BOLTSの実装ファイルを変更しない。
- BRIDGEの探索経路、探索結果、証明状態、Work計上、Budget Ledgerを変更しない。
- 同一Scenario、同一seed、同一worker数、同一Budgetで、変更前後の決定的ベンチマーク結果を完全一致させる。
- 実時間、割当量、実行日時、Execution IDなど非決定的な計測値は一致判定から除外する。
- `stable_digest`、返却経路、経路コスト、証明状態、終了理由、Work、Budget Ledger、グラフ・クエリ・アルゴリズム設定に差があれば未完了とする。

## 分割方針

- `products/cli`: app、route、benchmark、serve、scenario、artifact、metadataへ分割する。
- `TRAFFIC`: scenario model、generator、runner、metrics、artifactへ分割する。
- `GATE`: contracts、router、observation、graph mapperへ分割する。
- `HEALTHY`: profile、analysis service、validation、oracle、comparison、evaluationへ分割する。
- `products/server`: config、lifecycle、handlers、middlewareへ分割する。
- `CORE`: graph、route、errors、work、metrics、solverへ分割する。

この段階ではパッケージ名、公開API、処理順序、データ構造を変更しない。
