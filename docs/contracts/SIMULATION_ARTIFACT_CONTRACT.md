# Simulation Artifact Contract

> Status: Normative  
> Applies To: BRIDGE Simulator v0.15.x  
> Input: Benchmark Archive  
> Output: Simulation directory

## 入力契約

Simulatorの主入力はTRAFFICが生成したBenchmark Archiveです。個別のGraph、Trace、Algorithm名を通常利用者へ要求しません。

Simulatorは、Execution manifest、`runs.jsonl`、各Runの`references`を読み、Graph snapshotとTraceを対応付けます。パスを命名規則から推測する処理はFallbackにしません。

## Run選択

- TraceとGraph snapshotを持つ正式測定Runを自動処理します。
- Warm-up Runは除外します。
- TraceなしRunは警告として記録し、他のRun処理を継続します。
- 不完全Traceは出力Manifestへ状態を記録します。

## 時間軸

再生時間は`elapsed_ns`を正本とします。欠落時のみ`logical_step`を使用します。同時刻終了のRunは同一の正規化時刻で終了させます。

## 安全性

ZIP展開時に絶対パス、`..`、symlink、過大な展開サイズを拒否します。Schema非対応、参照切れ、GraphとQueryの不整合は明示的なエラーとします。

## 出力

各RunのGIFと、入力Archive、処理対象Run、警告、生成物を記録した`manifest.json`を生成します。
