# BRIDGEのアーキテクチャと責務境界

## 1. 基本構造

BRIDGEは、各コンポーネントが一つの主要責務を所有し、他コンポーネントの判断を代行しない構造を採用する。

```text
External Caller
      |
    GATE
      |
    TRUSS
   /     \
ANCHOR  BOLTS
   \     /
   BEARING
      |
 ULTRASOUND

TRAFFIC  -> 公開経路を通じた評価
HEALTHY  -> 生成済みartifactのread-only検証
CORE     -> 全体が共有する中立契約
```

この図は概念上の責務関係を示す。具体的なpackage依存や呼出し構造は現行規範を確認する。

## 2. CORE

### 役割

COREは、中立的な共有値型とデータ契約を提供する。

### 所有するもの

- Graph、Node、Edge等の基本契約
- RouteRequest、RouteResult
- Budget、Bounds、WorkMetrics
- cancellation、deadline、errorの中立表現
- コンポーネント間で共有するが、特定実装へ偏らない型

### 所有しないもの

- solver選択
- portfolio制御
- algorithm固有heuristic
- trace保存
- CLIや外部形式への変換

### 原則

COREは最下層の中立契約であり、上位コンポーネントの事情を取り込まない。

## 3. GATE

### 役割

GATEは、BRIDGEの外部公開境界である。

### 所有するもの

- 入力検証
- 既定値適用
- 外部IDと内部IDの変換
- 公開API用の要求・結果表現
- 外部向けerror mapping
- observer契約の受け渡し

### 所有しないもの

- ANCHORやBOLTSの直接起動
- solver選択
- portfolio予算配分
- 品質終了判断
- trace collectorや保存先の生成
- CLI、HTTP、ファイル等の具体的I/Oそのもの

### 原則

外部利用者はGATEを通じてBRIDGEを利用する。評価経路も、可能な限り同じ公開経路を使用する。

## 4. TRUSS

### 役割

TRUSSは、portfolio制御と予算の唯一の所有者である。

### 所有するもの

- query特性の評価
- task生成
- Budget Sliceの割当
- strategyとsolverの選択
- shared bounds
- fallbackとcertificationの開始判断
- 候補間の最終採用判断
- portfolio全体の終了判断

### 所有しないもの

- frontier操作
- edge relaxation
- solver private stateへの依存
- 観測結果を根拠とする制御変更

### 原則

ANCHORとBOLTSは互いを直接起動しない。TRUSSが両者を調停する。portfolio全体の予算、順序、継続、切替えはTRUSSだけが判断する。

## 5. ANCHOR

### 役割

ANCHORは、BRIDGE固有の主Anytime探索である。

### 所有するもの

- 複数の探索Hypothesis
- Corridor等を用いた主探索
- First Path生成
- Candidate生成と比較
- 局所Repair
- 自身が消費したWorkとStepの報告
- 進捗および緊急状態の報告

### 所有しないもの

- portfolio全体の予算変更
- fallback solverの選択
- 未証明のexact主張
- ULTRASOUNDへの直接依存
- BOLTSの直接起動

### 原則

ANCHORは主探索を担うが、全体最適な運用判断は行わない。必要な支援はTRUSSへ報告し、TRUSSがBOLTSを含む対応を決定する。

## 6. BOLTS

### 役割

BOLTSは、交換可能な補助solver群である。

### 所有するもの

- exactまたは準exact探索
- reachability
- fallback
- detourとrepair支援
- lower bound
- certification
- 自身が消費したWorkとStepの報告

### 所有しないもの

- portfolio scheduling
- ANCHORの継続判断
- 他solverの無制限な連鎖起動
- GATEへの直接公開

### 原則

BOLTSはアルゴリズムの格納場所ではなく、共通契約に従う補助能力の集合である。各solverは、予算境界、決定論性、計測契約を守る。

## 7. BEARING

### 役割

BEARINGは、探索層と観測層をつなぐ非干渉のevent契約である。

### 所有するもの

- typed event
- phase、step、lane等の共通語彙
- Observer interface
- Null Observer
- Safe Observer

### 所有しないもの

- event保存
- 集計
- replay
- bottleneck分析
- 予算変更
- cancellation指示

### 原則

BEARINGは制御チャネルではない。探索から観測へ事実を一方向に伝える。

## 8. ULTRASOUND

### 役割

ULTRASOUNDは、開発・検証・研究のための観測、保存、再生、分析を担う。

### 所有するもの

- event収集
- sequenceや経過時間等の付与
- Observation Modeによる選別
- Sinkへの配送
- trace保存
- metrics集計
- schema検証
- replayと分析

### 所有しないもの

- solver選択
- 予算再配分
- candidate採用
- graph変更
- 本番探索の必須依存

### 原則

ULTRASOUNDは探索結果を観測するが、探索を変えない。観測コストは探索Workへ含めず、別のoverheadとして扱う。

## 9. TRAFFIC

### 役割

TRAFFICは、テスト、benchmark、回帰、比較研究を担う評価基盤である。

### 所有するもの

- graphとqueryの生成・読込み
- ScenarioとRunの生成
- exact baselineとperformance baselineの設定
- benchmark、stress、回帰試験
- raw observationと統計集計
- 受入基準の判定

### 所有しないもの

- solver内部状態の変更
- 実行中の探索制御への介入
- private APIへの依存
- 本番route処理への組込み

### 原則

TRAFFICはBRIDGEを測定する側であり、BRIDGEの探索能力を補助してはならない。baselineは評価にのみ使用し、通常探索へ注入しない。

## 10. HEALTHY

### 役割

HEALTHYは、TRAFFICやULTRASOUNDが生成したartifactをread-onlyで検証する健全性評価層である。

### 所有するもの

- artifactの整合性確認
- Work保存則の検証
- Budget Ledgerとの照合
- traceからのWork再構成
- invalid Runの識別
- 比較・回帰結果の妥当性確認

### 所有しないもの

- solver制御
- artifact書換え
- exact referenceの探索注入
- 独自Work定義
- invalid Runを性能集計へ混入させる判断

### 原則

HEALTHYの評価結果は、同一Runの探索判断へ戻さない。事後検証に限定する。

## 11. 製品系と評価系の分離

### 製品系

- CORE
- GATE
- TRUSS
- ANCHOR
- BOLTS
- BEARING

### 任意観測・評価系

- ULTRASOUND
- TRAFFIC
- HEALTHY

製品系は、評価系が存在しなくても正しく動作できなければならない。評価系は製品系の公開契約を使用し、private implementationへ侵入しない。
