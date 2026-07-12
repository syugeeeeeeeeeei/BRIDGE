# BRIDGE Go 統合アーキテクチャ仕様

**文書ID:** BRIDGE-ARCH-SPEC-GO-001  
**版:** v1.0  
**状態:** 現行仕様  
**対象:** Go本番実装および比較研究基盤

## 1. 定義

BRIDGEは単一の最短路アルゴリズムではない。品質目標、Work Budget、deadline、worker数、memory制約、graph特性に応じて、主探索と補助solverを統合運用するAnytime経路探索スタックである。

## 2. 目的

- first pathを早く得る
- 残予算で経路品質を改善する
- 苦手条件を検知する
- BOLTSへ適切に切り替える
- exactまたは品質比を証明する
- Work、Step、deadline、memoryを管理する
- 実行を再現・比較・分析可能にする

## 3. コンポーネント

```text
BRIDGE
├─ CORE
├─ GATE
├─ TRUSS
├─ ANCHOR
├─ BOLTS
├─ BEARING
├─ ULTRASOUND
└─ TRAFFIC
```

本番最小構成は`CORE + GATE + TRUSS + ANCHOR + BOLTS + BEARING Null Observer`である。ULTRASOUNDとTRAFFICは開発・検証専用である。

## 4. 制御フロー

```text
利用者
  │
  ▼
GATE
  │ RouteRequest
  ▼
TRUSS
  ├─ ANCHOR task
  ├─ BOLTS reachability/fallback/certification task
  └─ portfolio metrics / shared bounds
        │
        ▼
     RouteResult
```

観測eventは探索層からBEARINGを経てULTRASOUNDへ流れる。観測側から探索制御へ値を返さない。

## 5. Goデータモデル

- 外部IDはGATEで連続`NodeID`へ変換する
- Graphはread-only探索契約を公開する
- 構築用graphと本番用CSR graphを分離可能にする
- requestは値型で保持し、既定値適用とvalidationを分ける
- resultはalgorithm結果とportfolio metricsを区別する
- error codeは安定した独自型で公開する

## 6. WorkとStep

`docs/WORD_DEFINITION.md`を正本とする。Workは意味的Actionの総数、Stepは並列化後の依存深度である。各taskはAction別counterを返し、TRUSSがportfolio全体を合算する。

## 7. TRUSS

TRUSSは以下を所有する。

- query profile
- 初期計画
- Work Budget
- budget slice
- solver capability参照
- task scheduling
- shared upper/lower bounds
- stagnation検出
- fallback
- certification
- 終了条件
- 最終結果選択

TRUSSはfrontier、edge relaxation、parent mapを直接操作しない。

## 8. ANCHOR

ANCHORは複数Hypothesisを利用する。

- geometric corridor
- weighted cost
- bidirectional hypothesis
- hub-aware
- portal
- diverse corridor
- repair

戦略選択はTRUSSが行い、ANCHORは指定された戦略を実行する。ANCHOR内のalternate hypothesisはfallbackではない。

## 9. BOLTS

BOLTSは交換可能な補助solverである。

- Dijkstra
- bidirectional Dijkstra
- A*
- reachability
- local detour
- repair
- lower bound
- exact certification

各solverはcapability、Work、Step、found、distance、bound、exact性を報告する。

## 10. BEARINGとULTRASOUND

BEARINGはtyped eventとNull Observerを定義する。ULTRASOUNDは保存・集計・replay・分析を行う。

本番既定はNull Observerとし、観測有無でpath、distance、Work、Step、solver選択が変化してはならない。

## 11. TRAFFIC

TRAFFICは次を担当する。

- topology generator
- deterministic fixture
- benchmark manifest
- paired Python-Go comparison
- p50/p95/p99/worst集計
- repeatability判定
- migration completion gate
- failure bundle

本番route実行には含めない。

## 12. 決定論性

- adjacencyを正規化する
- queue tie-breakingを固定する
- floating point正規化規則を固定する
- candidate採用順を固定する
- map iteration順へ依存しない
- Stable Digestからtimingを除外する

## 13. 旧アーキテクチャからの継承

旧Python版で確立した次の方針は維持する。

- TRUSS相当の予算単一所有
- ANCHOR相当の主探索
- BOLTS相当のdetour/fallback/exact分離
- BEARINGによる観測契約
- ULTRASOUNDとTRAFFICの本番分離
- GATE以外から外部形式を受け取らない

名称変更前のPIER、CABLE、MPRC/MRPCに関する資料は`others/legacy/`の履歴として扱う。

## 14. 完了条件

- architecture testが通る
- budget violationが0件
- observer非干渉が成立する
- repeatabilityが100%
- exact modeがbaselineと一致する
- Python-Go傾向相関が移行基準を満たす
- Go本番実装がlegacyなしで動作する
