# BRIDGE debugモード改修 実装結果

## 概要

旧debug Collectorは、全Eventのメモリ保持とEvent追加ごとの全履歴再集計により、Event数に対して二次的に処理時間が増加していた。また、Work ActionをほぼTrace同等の粒度で取得していたため、Traceファイルを保存しなくても大きな観測負荷が発生していた。

今回の改修では、debugを「Traceを保存しないモード」ではなく「限定された診断値をO(1)で逐次集約するモード」として実装し直した。

## 主な実装

- Collectorの全Event保持を廃止
- Eventごとの`Summarize`再走査を廃止
- Action Eventをdebug対象から除外
- Kind、Phase、Sequence、Frontier最大値を逐次集約
- 構造化`debug_summary`をRaw Runへ追加
- Work Action別、Component別Work、Purpose別Budgetを記録
- Candidate、Fallback、Certification、State Reuseを記録
- Context終了時のTRUSS PanicをError処理へ変更
- Context取消をExecution EngineのTask投入時にも処理
- 回帰テスト追加

## 検証

`go test ./...`は全パッケージで成功した。

1000ノード条件では、旧実装で発生したPanicは再現せず、minimumとdebugのWork、経路発見率、最適性判定が一致した。debug成果物には`traces/`が作成されず、`trace_sink_write_ns`は0であった。

詳細な項目と証跡は`docs/benchmark/DEBUG_MODE_IMPLEMENTATION_CHECKLIST.md`を参照する。
