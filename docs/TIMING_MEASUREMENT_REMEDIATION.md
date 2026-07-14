# BRIDGE 時間計測基盤 設計欠陥・改修完了チェックリスト

## 1. 厳格評価

従来実装は、研究ベンチマークとして時間値を比較できる状態ではなかった。

- GATEが公開API境界時間からTRUSS内部時間を減算し、GATE固有時間を推定していた。
- 独立した時計区間の差分を内訳として扱い、負値を0へ補正して計測不整合を隠蔽していた。
- solver、TRUSS、GATEの短い単発計測を、時計分解能や量子化を検証せず研究結果へ採用していた。
- TRAFFICが自ら所有すべきベンチマーク境界時間を測定せず、本番コンポーネントの診断値を研究上の一次測定値として流用していた。
- `zero_duration`がsolverまたはend-to-endの片方が0であるだけで真となり、測定失敗と実行時間0を区別できなかった。
- ミリ秒浮動小数のみを一次値として保持し、精度、由来、妥当性を監査できなかった。

これは単なる精度不足ではなく、測定責務、データ来歴、不変条件の設計違反である。

## 2. 責務に基づく修正方針

### CORE

- 時間の公開契約を定義する。
- ナノ秒整数を一次値、ミリ秒を互換表示値とする。
- 評価判断や計測実行は所有しない。

### ANCHOR / BOLTS

- 自身のsolver境界だけを診断計測する。
- GATE、TRUSS、ベンチマーク全体時間を推定しない。

### TRUSS

- RouteまたはExecuteOnceの実行境界を直接計測する。
- solver診断値との差分を正規のexclusive時間として断定しない。

### GATE

- 公開API境界を直接計測する。
- 内部時間との差分によるGATE時間推定と負値補正を行わない。
- アルゴリズム性能の判定を行わない。

### TRAFFIC

- 研究ベンチマークの一次end-to-end時間を公開API呼出しの周囲で直接計測する。
- 計測値の妥当性、0時間、時計分解能以下の可能性を記録する。
- 短時間solverの順位付けには反復ベンチマークが必要であることを明示する。

### HEALTHY

- 時間値の非負性、境界順序、0時間率、量子化、一次値の存在を検査する。
- 不正な時間内訳を警告または失敗として報告する。

## 3. 実装チェックリスト

- [x] `TimeBreakdown`へ`total_ns`、`solver_ns`、`truss_ns`、`gate_ns`を追加
- [x] ミリ秒値をナノ秒から導出する形式へ変更
- [x] ANCHOR／BOLTSの`Microseconds()`切捨てを廃止
- [x] TRUSSの実行境界をナノ秒で直接計測
- [x] GATEの「外側時間－内側時間」による推定を廃止
- [x] GATEの負値を0へ隠蔽する処理を削除
- [x] TRAFFICが公開API呼出し境界を直接計測
- [x] `solver_time_ns`と`end_to_end_time_ns`をraw runへ保存
- [x] `zero_duration`を公開API境界の実測値だけで判定
- [x] `timing_valid`と`timing_issue`を追加
- [x] 既存JSONのミリ秒フィールドを互換維持
- [x] COMPONENT_RULEへ時間計測責務を追記
- [x] 回帰テストを追加
- [x] `go test ./src/bridge/...`通過
- [x] `go test ./...`通過
- [x] `go vet ./...`通過

## 4. 残る制約

今回の修正は、0 msを「実行が無時間だった」と誤認する問題と、階層差分による不整合を解消する。

ただし、1回のsolver処理が時計分解能に近い場合、単発値だけでアルゴリズム順位を確定してはならない。研究用の厳密な速度比較では、TRAFFICにキャリブレーション付き反復計測を追加し、一定総計測時間以上のサンプルを生成する必要がある。この制約は`timing_valid`と`timing_issue`で明示される。

## 5. 再ベンチマークで判明した追加欠陥と修正

簡易格子シナリオによる回帰ベンチマークで、BRIDGE Routeだけ `solver_ms > 0` である一方、`solver_ns = 0` となる欠陥が判明した。

原因は、TRUSS RouteがANCHOR／BOLTSの実行時間をミリ秒値だけで加算し、新設したナノ秒一次値を集計・伝播していなかったことである。これは「ナノ秒を一次値とする」というCOREの時間契約に違反し、TRAFFICの妥当性判定をBRIDGEだけ失敗させていた。

### 追加修正チェックリスト

- [x] `TimeBreakdown`へ`anchor_ns`、`bolts_ns`、`fallback_ns`を追加
- [x] `TimeBreakdown`へ`supervisor_ns`、`arbiter_ns`、`orchestration_ns`を追加
- [x] TRUSS Routeで各solverの`solver_ns`を直接集計
- [x] `solver_ns = anchor_ns + bolts_ns`を不変条件として実装
- [x] `solver_ms`、`anchor_ms`、`bolts_ms`を各ナノ秒値から導出
- [x] `solver_time_ns`をtelemetryへ伝播
- [x] ExecuteOnceがsolverのナノ秒値を保持することを回帰テストで確認
- [x] BRIDGE Routeの`solver_ns > 0`を回帰テストで確認
- [x] `total_ns >= solver_ns`を回帰テストで確認
- [x] 全パッケージテスト通過
- [x] `go vet ./...`通過
- [x] 100ノード簡易ベンチマークでBRIDGEを含む全本計測の`timing_valid=true`を確認
- [x] 同ベンチマークで時間境界順序とナノ秒・ミリ秒整合を確認

### 再ベンチマーク結果

100ノード開放格子、観測OFF、warm-up 2回、本計測10回で確認した。

- BRIDGE本計測10件すべてで`solver_time_ns > 0`
- BRIDGE本計測10件すべてで`timing_valid = true`
- `solver_ns = anchor_ns + bolts_ns`が全件成立
- `total_ns >= solver_ns`が全件成立
- `gate_ns >= total_ns`が全件成立
- `end_to_end_time_ns >= gate_ns`が全件成立
- `solver_ms = solver_ns / 1,000,000`が全件成立

これにより、前回残っていたBRIDGE Routeのナノ秒伝播欠陥は解消した。
