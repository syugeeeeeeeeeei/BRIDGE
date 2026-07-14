# ANCHOR改善実装・監査結果

## 実装内容

- TRUSSの固定3仮説（corridor/portal/hub）を削除した。
- ANCHORを単一の`adaptive_fast_path` Sessionで開始する構造へ変更した。
- Heuristic WeightはRoute ModeとGraph分析結果から選択する。
- Fast/Balanced/Qualityでは最初のCandidate発見後に返却可能とした。
- Exactのみ条件付きCertificationを許可した。
- 停滞時だけBOLTS Weighted A*を起動する条件付きHandoffを実装した。
- Heuristic換算係数の全Graph走査をNodeごとに繰り返す欠陥を修正した。
- minimum観測でAction Eventを生成しないfast pathを実装した。
- Region更新の全Sliceコピーを廃止した。

## 検証

`go test ./...`は全パッケージ成功。

1000ノード級6トポロジー、2計測Run、1 warm-upで再ベンチマークした。

| Algorithm | 平均Work | 平均Solver ms | 平均Gap | 最大Gap |
|---|---:|---:|---:|---:|
| Weighted A* | 4,552.3 | 0.5248 | 0.221% | 0.801% |
| ANCHOR | 2,470.8 | 0.2999 | 1.594% | 7.974% |
| BRIDGE | 2,301.3 | 0.5138 | 1.594% | 7.974% |

ANCHORは平均Workを45.7%、平均Solver時間を42.9%削減した。一方、random-geometricでRelative Gapが約7.97%となり、最大5%目標を満たしていない。CommunityではWeighted A*のWork 284に対してANCHOR 1,534であり、全トポロジー低Workも未達である。

## 完了判定

実装チェック項目は完了したが、性能受入基準は未完了である。この版を「新ANCHOR方針の性能完成版」とは扱わない。
