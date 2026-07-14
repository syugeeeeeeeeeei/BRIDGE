# Reports Index

`reports/` は履歴性の強い文書を目的別に分類した領域です。

## Categories

- `audits/`: 完了チェック、gap audit、implementation audit
- `benchmarks/`: ベンチマーク結果、性能比較、runtime 評価
- `implementation/`: 実装内容の要約、改修記録、反映報告
- `migration/`: Go 移植、Python 比較、移行 readiness
- `validation/`: 契約検証、意味論統制、コンポーネント整合
- `data/`: レポートに紐づく JSON などの補助データ
- `timing-regression/`: timing regression 再現用の入力と結果

## Classification Rule

- 将来の新規文書は、恒久的な仕様なら `docs` 直下の各仕様系ディレクトリへ置く。
- 実装時点の状態を記録する文書は `reports/` 配下へ置く。
- チェックリストや完了判定は `audits/` を優先する。
- 実験結果や比較評価は `benchmarks/` を優先する。
