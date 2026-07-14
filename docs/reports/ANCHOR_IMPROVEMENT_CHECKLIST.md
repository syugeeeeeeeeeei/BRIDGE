# ANCHOR改善実装チェックリスト

## 実装・構造
- [x] 固定3仮説を廃止し、単一主Sessionで開始する
- [x] Weighted A*系Heuristicを初期経路生成部品として使用する
- [x] route modeとGraph特性からHeuristic Weightを選択する
- [x] Candidate発見時に非exactモードで早期終了する
- [x] First Path Work・First Path Time・Candidate更新回数を記録する
- [x] HeuristicをNode単位でCacheする
- [x] Graph全体のHeuristic換算係数をRun開始時に一度だけ計算する
- [x] minimumモードでAction Event生成を省略する
- [x] Region更新時の全Sliceコピーを廃止する
- [x] 停滞閾値と条件付きBOLTS移行を実装する
- [x] BOLTS移行を固定実行せず、停滞時のみ起動する
- [x] BOLTS結果をCandidateとしてANCHOR Sessionへ戻せる
- [x] State Reuse件数・比率をTelemetryへ記録する
- [x] Work BudgetをANCHOR・BOLTS・TRUSSで統一会計する
- [x] 旧multi-hypothesis前提のテストを新契約へ更新する
- [x] 全Goテストを成功させる

## 計測可能性
- [x] Weighted A*とのWork比較が可能
- [x] Weighted A*とのSolver Time比較が可能
- [x] 最短距離に対するRelative Gapを計算可能
- [x] トポロジー別のWork・時間・Gapを出力可能
- [x] ANCHORとBRIDGEを個別に評価可能

## 性能受入基準
- [x] 6トポロジー平均でANCHOR WorkがWeighted A*未満
- [x] 6トポロジー平均でANCHOR Solver TimeがWeighted A*未満
- [x] 全RunでPath Found Rate 100%
- [ ] 最大Relative Gap 5%以下
- [ ] Weighted A*比Workが全トポロジーで同等以下
- [ ] BRIDGE Solver Timeが全トポロジーでWeighted A*以下

## 監査判定
実装機構と計測基盤は完了した。ただし性能受入基準に未達項目があるため、新ANCHOR方針の性能完成とは判定しない。
