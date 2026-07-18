# 観測処理・純粋アルゴリズム性能 分離修正監査

## 実施内容

- `ANCHOR Session.Step`で`state_delta`と`action`の有効性をStep開始時に判定するよう変更
- 無効な`state_delta`イベントについて、属性`map[string]any`を生成しないよう全高頻度呼出しをガード
- Observation無効時と有効時のAllocation差を検証する回帰テストを追加
- BEARINGおよびANCHORのコンポーネント規則へ、Observation無効時のAllocation非増幅規則を追加
- BOLTS、TRUSS、ULTRASOUND、GATE、TRAFFICのイベント生成・進捗収集・Artifact処理を監査

## 修正対象

- `src/bridge/anchor/session.go`
- `src/bridge/anchor/session_test.go`
- `src/bridge/anchor/COMPONENT_RULE.md`
- `src/bridge/bearing/COMPONENT_RULE.md`

## 不変条件確認

Maze 10,000 / seed 7 / Balanced:

- Path Found: 維持
- Path Cost: 2,466
- Work: 43,972
- Observationによる探索判断変更: なし

Grid 10,000 / seed 7 / Balanced:

- Path Found: 維持
- Work: 2,780

## 修正後ローカル測定

| Scenario | Solver median | End-to-End median | Allocation | malloc | Work |
|---|---:|---:|---:|---:|---:|
| Maze 10,000 | 2.077 ms | 2.084 ms | 1,216,768 B | 19,429 | 43,972 |
| Grid 10,000 | 0.733 ms | 0.738 ms | 498,496 B | 1,134 | 2,780 |

測定はGraphを事前生成し、`gate.New(nil).Route`を9回実行、初回をWarm-upとして除外した中央値です。既存の正式Benchmark Runnerとは計測境界が異なるため、正式Baseline値の置換には使用しません。

## 追加監査結果

### 高頻度ループ内の観測コスト

- ANCHOR: 今回修正。主要欠陥を確認した唯一の箇所
- BOLTS: 高頻度`state_delta`は既に`bearing.Wants`で属性生成前にガード済み
- ULTRASOUND: CollectorはObserverとして受信後に処理し、無効時の探索ループへ介入しない
- Progress Sample: `CollectProgressSamples`がfalseの場合はSample構造を生成しない

### 残存する純粋アルゴリズム外コスト

1. Graph変換・再構築
   - `traffic.graphToInput`
   - `gate.buildGraph`
   - `core.NewAdjacencyGraph` / edge canonicalization
   - Benchmarkや外部入力経路のEnd-to-End固定費であり、ANCHOR Solver内部の主要因ではない

2. TRUSSの低頻度イベント
   - search/component/fallback単位でMapを生成する
   - Workごとではなく定数回であるため、現段階では重大ボトルネックではない
   - Observation完全off時のゼロ固定費を目指す場合は、将来の追加改善候補

3. BOLTSの開始・終了イベント
   - 1探索当たり定数回のMap生成が残る
   - 高頻度イベントはガード済み
   - サブミリ秒Solverでの厳密な固定費監査候補

4. Region Nodes・Priority Queue
   - 修正後測定ではMazeの重大なAllocation異常を示していない
   - アルゴリズム状態であり、観測処理とは異なる
   - 次回profileで5%以上を占めた場合のみ最適化対象とする

## テスト結果

- `go test -count=1 ./...`: PASS
- Observation無効時Allocation回帰テスト: PASS
- ANCHOR Snapshot/Resume同値性: PASS

## 判定

観測仕様が探索WorkごとのCPU・Allocationを増幅する主要欠陥は修正済みです。同種の高頻度欠陥は他コンポーネントでは確認されませんでした。残存項目はGraph変換などのEnd-to-End固定費と、1探索当たり定数回の低頻度イベント生成です。
