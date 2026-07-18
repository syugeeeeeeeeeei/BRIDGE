# TRAFFIC コンポーネント規則

**対象package:** `src/bridge/traffic`  
**対象版:** v0.15.0以降  
**状態:** 規範文書

## 1. 責務

- Scenario、Graph、Query、Dataset生成
- benchmark、stress、ablation、回帰
- raw Run、統計、環境metadata
- 公開API境界のend-to-end timing
- benchmark不変条件のfail-closed検査

## 2. 禁止事項

- solver private stateの参照または変更
- 同一Run中の探索制御介入
- 証明状態の再推定
- 本番Route処理への組込み

## 3. 証明・結果契約

GATEが返した`search_completed`、`reachability_proven`、`optimality_proven`、`TerminationStatus`をそのまま保存する。`found`や`error_code`から再計算してはならない。

以下を検出したRunは集計せず失敗させる。

- pathなしでoptimality proven
- search未完了でoptimality proven
- pathありでreachability未証明
- budget不足なのに証明済み
- `NO_PATH`とpathありの併存
- timing validかつsolver timeが0
- path距離と再計算値の不一致

## 4. TimingとWarm-up

- end-to-end時間は公開API呼出し境界で直接測定する
- nsを一次値とし、msは派生値とする
- 無効Timingを性能比較へ使用しない
- Warm-upは性能集計から除外する
- Warm-upではObservation、Collector、Trace保存を無効化する
- Warm-upをraw測定Runとして保存してはならない

## 5. 必須テスト

- Scenario validation
- benchmark invariant negative test
- timing validity test
- warm-up非観測テスト
- deterministic repetition test
- structure別ablation test

## Benchmark lifecycle observation

TRAFFIC MUST publish low-frequency lifecycle boundaries through BEARING for benchmark run, graph generation, query generation, and graph conversion. HEALTHY remains independent; its caller publishes the evaluation boundary. Artifact encoding, integrity calculation, file I/O, and compression are benchmark-process responsibilities and MUST remain outside solver/request timing.
