# ANCHOR アルゴリズム仕様

**文書ID:** BRIDGE-ANCHOR-SPEC-GO-001  
**版:** v1.0  
**対象:** `src/bridge/anchor`

## 1. 目的

ANCHORは、複数の探索仮説を利用してfirst pathを早期に発見し、残Work Budgetで候補経路を改善するBRIDGE固有の主Anytime探索である。

## 2. 入出力

入力:

- read-only Graph
- source/target
- TRUSSが選択したstrategy
- Work Budget slice
- worker slice
- shared upper bound
- observer

出力:

- Candidateまたはnot found
- distance
- WorkMetrics
- first path work/step
- lower/upper bound更新候補
- stagnation情報
- failure reason

## 3. 探索仮説

### geometric corridor

source-target直線や複数offsetを基に部分領域を作り、その内部を探索する。

### weighted cost

幾何距離とedge weightのばらつきを考慮し、重みノイズが大きいgraphで探索する。

### hub-aware

degreeの高いnodeを接続候補として用いる。

### portal

長距離edgeや領域境界のendpointを経由候補として使う。

### bidirectional hypothesis

source側とtarget側のlaneを持つ。exact保証はBOLTSの双方向Dijkstraが担当し、ANCHOR版はcandidate生成を目的とする。

### repair

既存pathの高cost区間を局所再探索して置換する。

## 4. Work計測

各意味的Actionを`docs/WORD_DEFINITION.md`に従って計上する。単なるloop、配列参照、telemetryはWorkに含めない。

## 5. 決定論性

- corridor順を固定する
- portal/hub順位を安定sortする
- queue同順位ではscore、sequence、NodeIDの順で比較する
- 同距離Candidateでは先に確定したものを保持する

## 6. 禁止事項

- portfolio全体Budgetの変更
- fallback solverの独断起動
- exact性の未証明主張
- ULTRASOUNDへの直接依存
- private BOLTS実装への直接依存

## 7. 必須テスト

- open grid
- wall
- U-shape
- cul-de-sac
- weighted noise
- disconnected
- Work Budget境界
- observer非干渉
- 連続再実行一致
- path妥当性
