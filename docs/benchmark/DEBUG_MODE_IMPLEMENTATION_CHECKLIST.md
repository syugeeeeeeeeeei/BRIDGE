# BRIDGE debugモード 実装完全性チェックリスト

## 1. 完了条件

本チェックリストの全項目が実装済みかつ自動テストまたは1000ノード実測で確認されるまで、debugモード改修を完了としない。

## 2. Collector計算量・メモリ

- [x] Event追加ごとの全履歴再集計を廃止した。
- [x] `Summarize(c.events)`を実行経路から除去した。
- [x] debugモードでEvent全件を保持しない。
- [x] Kind、Phase、Sequence、最大Logical StepをO(1)で逐次集約する。
- [x] debugモードでAction Eventを要求しない。
- [x] debugモードでAction Eventを受信しても保持・集計しない。
- [x] debugモードでTrace Sinkへ書き込まない。
- [x] debugモードでTraceファイルおよび`traces/`を生成しない。
- [x] `sink_write_ns`が0であることを回帰テストで確認した。
- [x] Collector Metrics返却時に内部Mapおよび履歴Sliceを複製する。

## 3. debug診断データ

- [x] Work Action別件数を`debug_summary.action_counts`へ保存する。
- [x] Component別Workを`debug_summary.work_by_component`へ保存する。
- [x] Purpose別Budget Grantedを保存する。
- [x] Purpose別Budget Usedを保存する。
- [x] Candidate更新回数を保存する。
- [x] Fallback回数を保存する。
- [x] Certification回数を保存する。
- [x] State Reuse適用回数を保存する。
- [x] 最大Frontier Sizeを保存する。
- [x] Component別診断Event件数を保存する。
- [x] 観測処理時間を保存する。
- [x] Trace Sink書込時間を保存する。
- [x] Dropped Event数とTruncation状態を保存する。
- [x] Raw Runごとに構造化された`debug_summary`を保存する。
- [x] Canonical WorkとBudget Ledgerを診断集約の情報源として使用する。

## 4. 実行安全性

- [x] Context終了時にExecution Engineが未実行Taskへ値を設定しないことを許容する。
- [x] Task投入中もContext終了を検出する。
- [x] TRUSSがTask結果を型Assertionする前にContext終了を検査する。
- [x] nilまたは不正なTask結果をPanicではなくErrorとして処理する。
- [x] 1000ノードdebug実行がPanicせず完走することを確認した。

## 5. 結果不変性

- [x] minimumとdebugでPath Foundが一致する。
- [x] minimumとdebugでOptimality判定が一致する。
- [x] minimumとdebugでPath Costが一致する。
- [x] minimumとdebugでWorkが一致する。
- [x] Warm-up Runではminimum相当を使用する。
- [x] 観測処理をWorkへ算入しない。

## 6. 性能・Trace分離

- [x] 1000ノード、BRIDGE、3計測Runでminimumとdebugを比較した。
- [x] debug成果物にTraceディレクトリが存在しないことを確認した。
- [x] debugの`trace_sink_write_ns`が0であることを確認した。
- [x] debugの平均End-to-End時間がminimumと同等範囲であることを確認した。
- [x] 旧実装の約63倍のdebugオーバーヘッドが解消された。
- [x] Event数がWork数に比例するAction Streamではなく、診断対象Eventへ限定された。

## 7. HEALTHY・成果物

- [x] minimum成果物に`healthy.json`が生成される。
- [x] debug成果物に`healthy.json`が生成される。
- [x] 全計測RunのHEALTHY Run Validationが`pass`である。
- [x] `runs.jsonl`から診断値を取得できる。
- [x] `summary.csv`、`result.json`、`manifest.json`が通常どおり生成される。

## 8. 自動テスト

- [x] debug Collectorが10,000件のAction Eventを保持しないテストを追加した。
- [x] debug CollectorがTrace Sinkへ書き込まないテストを追加した。
- [x] Frontier最大値の逐次集約テストを追加した。
- [x] Context取消時のExecution Engine回帰テストを追加した。
- [x] `go test ./...`が全件成功した。

## 9. 実測結果

条件:

- Graph: `grid-open-1000`
- Algorithm: `bridge`
- Seed: `101`
- Warm-up: 1
- 計測Run: 3
- Observation: `minimum`対`debug`

結果:

| 指標 | minimum | debug |
|---|---:|---:|
| 平均End-to-End | 185.336 ms | 183.608 ms |
| 平均Solver | 184.764 ms | 183.006 ms |
| Work | 42,861 | 42,861 |
| Traceディレクトリ | なし | なし |
| Trace Sink書込 | 0 | 0 |
| HEALTHY Run Validation | 全件pass | 全件pass |

単回の短時間測定ではdebugがminimumより速く見えるが、これは測定揺らぎであり、debug高速化を意味しない。重要な確認事項は、旧実装の二次的再集計とAction Stream保持が除去され、Trace相当の負荷が再現しなかったことである。

## 10. 完了判定

- [x] 全項目完了
- [x] 未完了項目なし
- [x] debugモード改修を完了と判定する
