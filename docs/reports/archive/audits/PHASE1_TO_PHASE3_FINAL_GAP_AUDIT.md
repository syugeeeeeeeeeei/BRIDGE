# Phase 1〜3 最終抜け漏れ監査

## 判定

Phase 1〜3の公開契約、実装、schema、テスト、用語集を再監査した結果、Phase 4へ進むことを妨げる未実装は確認されなかった。

## 再監査した観点

1. Scenario fieldがvalidationだけでなく実行経路で消費されているか。
2. raw resultからquery別summaryを再生成できるか。
3. 観測modeがTRAFFICからGATE・BEARING・ULTRASOUNDまで接続されるか。
4. summaryがtrace保存を要求せず、trace I/Oを行わないか。
5. sample rateが決定論的にevent採否へ反映されるか。
6. Stable Digestが観測modeで変化しないか。
7. Workと観測・runtime計測が混同されていないか。
8. 時間・Work・system metricsがrawとsummaryの双方へ保存されるか。
9. trace manifestにsampling、欠落、overhead、checksumが存在するか。
10. 現行packageに旧Recorderや旧modeが残っていないか。
11. JSON SchemaとGo構造体のfield名が一致するか。
12. BRIDGE固有用語が用語集に定義されているか。

## 破壊的整理

- 旧ULTRASOUND Recorderを`src/bridge/ultrasound`から削除した。
- 旧実装は`others/legacy/ultrasound-recorder-v0.14.1/`へ退避した。
- legacy CLIは`legacy` build tagなしではbuild対象にしない。
- `heap_alloc_peak`を廃止し、境界観測値とprofile sampled peakを別fieldへ分離した。
- Case summaryをScenario×Algorithm×Query単位へ破壊的変更した。

## 検証結果

- `go test ./...`: 成功
- `go test -race ./...`: 成功
- `go vet ./...`: 成功
- `python tests/compatibility/verify.py`: 成功
- Python-Go semantic parity: 10件一致

## Phase 1〜3の外にある既知事項

`evaluate_research_readiness.py`の`trend_correlation`は0.407665で、既存閾値0.7を満たしていない。これはPhase 1〜3の観測・artifact基盤の欠落ではなく、Go版とPython版のWork trend相関という移植・アルゴリズム評価上の課題である。Phase 4以降の研究評価で継続して扱う。

## 完了判定

`docs/reports/audits/PHASE1_TO_PHASE3_COMPLETION_CHECKLIST.md`の項目はすべて実装または明示的なlegacy移行で解決済みである。Phase 1〜3を完了とみなす。

