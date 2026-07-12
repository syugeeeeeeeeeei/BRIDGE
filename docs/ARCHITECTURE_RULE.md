# BRIDGE アーキテクチャ規則

**文書ID:** BRIDGE-ARCH-RULE-001  
**版:** v2.0  
**対象:** BRIDGE Go本番実装、互換性検証、開発・評価資産  
**状態:** 規範文書

## 1. 目的

本書は、BRIDGEのGo実装におけるリポジトリ構造、コンポーネント責務、依存方向、予算所有権、観測非干渉、試験、文書管理を定める最上位規則である。

BRIDGEは経路を返すだけでは正しい実装とはみなさない。以下が同時に成立する必要がある。

- コンポーネント責務が分離されている
- 依存方向が一方向である
- TRUSSがportfolio全体の予算を所有する
- Work、Step、品質指標の意味が一貫している
- 観測の有無が探索結果へ影響しない
- 同一入力で決定論的な結果を再現できる
- Python参照版との比較研究が可能である
- 文書とコードの不一致を欠陥として扱う

## 2. 規範文書の優先順位

内容が重複する場合、次の順に優先する。

1. `docs/ARCHITECTURE_RULE.md`
2. `docs/WORD_DEFINITION.md`
3. `src/bridge/<component>/COMPONENT_RULE.md`
4. `docs/architecture/BRIDGE_architecture_spec_v0.0.1.md`
5. `docs/algorithms/`配下のアルゴリズム仕様
6. `docs/migration/`配下の移行基準
7. `docs/reports/`配下の実装・評価記録
8. `README.md`

報告書は特定時点の記録であり、規範文書を上書きしない。コードと規範文書が異なる場合は、どちらかを無言で読み替えず、不整合として修正する。

### 2.1 用語集更新義務

BRIDGE独自の定義、意味、責務、計測範囲、判定条件、または使用制約を持つ用語を追加・変更する場合、同一変更で`docs/WORD_DEFINITION.md`を更新する。一般的な技術用語をBRIDGE固有の限定された意味で使用する場合も対象とする。

公開schema、CLI option、Scenario field、result field、trace vocabulary、component名、algorithm機構名、研究指標へ未定義語を追加してはならない。code reviewと互換性検証では、用語集更新の要否を確認する。

## 3. リポジトリ構成

```text
BRIDGE/
├─ src/
│  ├─ bridge/
│  │  ├─ core/
│  │  ├─ gate/
│  │  ├─ truss/
│  │  ├─ anchor/
│  │  ├─ bolts/
│  │  ├─ bearing/
│  │  ├─ ultrasound/
│  │  └─ traffic/
│  ├─ products/cli/cmd/bridge/
│  ├─ sdk/
│  │  ├─ python/
│  │  └─ typescript/
│  ├─ contracts/
│  └─ internal/
├─ docs/
├─ tests/
│  ├─ examples/
│  ├─ scenarios/
│  └─ compatibility/
├─ others/
│  └─ legacy/
├─ go.mod
├─ mise.toml
└─ README.md
```

### 3.1 ルートに置けるもの

- `README.md`
- `go.mod`および`go.sum`
- 開発環境の最小設定ファイル
- `src/`、`docs/`、`tests/`、`others/`の4領域

本体コード、SDK、契約、テスト資産、履歴資産をルートへ直接追加してはならない。ベンチマーク生成物、trace、profiling結果、展開済みアーカイブも恒久配置してはならない。

### 3.2 `src/`

BRIDGEそのものに関わる現行資産を置く。Go本体、正式製品、SDK、公開契約、現行製品だけが使用するinternal packageを含む。

Goの`internal`可視性規則により、`src/products`や`src/bridge`から利用する非公開packageは`src/internal/`に置く。実働packageを`others/internal/`へ移してはならない。

### 3.3 `tests/`

実行例、Scenario、互換性検証、fixtureなど、試験・評価に関わる資産を置く。本番コードから実行時依存してはならない。

### 3.4 `others/legacy/`

`others/legacy/bridge_py`はPython参照実装であり、Go本番実装からimportまたは実行時依存してはならない。利用目的は次に限定する。

- paired comparison
- Golden Case生成
- 移植差分調査
- 過去仕様の確認

新機能をlegacyへ追加してはならない。

## 4. コンポーネント構成

```text
GATE → TRUSS → ANCHOR / BOLTS
  │       │          │
  └───────┴──────────┴→ BEARING契約
                           ↑
                    ULTRASOUND（任意）

TRAFFIC → GATE公開API（開発・評価専用）
```

### CORE

共有値型、Graph契約、RouteRequest、RouteResult、WorkMetrics、Budget、Boundsを所有する。制御判断、solver選択、永続化、外部API変換を持たない。

### GATE

公開API、入力検証、既定値適用、外部ID変換、結果表現、エラー変換を所有する。個別solverを直接呼ばず、TRUSSのみを呼ぶ。観測についてはBEARINGのObserver契約だけを受け取り、Collector、Sink、保存先を生成しない。stdin、ファイル読込み、HTTP等の外部I/Oは製品アダプターが担当する。

### TRUSS

portfolio計画、全体Work Budget、task生成、solver選択、fallback、certification、終了判定、最終結果選択を単独所有する。

### ANCHOR

BRIDGE固有の主Anytime探索を実行する。Corridor、Hypothesis、Candidate、Repairを扱うが、portfolio全体の予算やfallback選択を所有しない。

### BOLTS

Dijkstra、双方向Dijkstra、A*、reachability、detour、repair、certification等の補助solverを提供する。BOLTS自身が他solverを起動してはならない。

### BEARING

型付き観測契約とNull Observerを提供する。Event、Observer、DetailObserver、NullObserver、SafeObserver以外の収集・保存責務を持たない。Collector、Sink、Recorder、集計、分析、制御判断を行わない。

### ULTRASOUND

BEARING eventの収集、sequence・elapsed・delta付与、Sink配送、保存、集計、truncation管理、検証、replay、分析を行う開発・検証用コンポーネントである。本番経路探索の必須依存にしない。

### TRAFFIC

テスト、benchmark、stress、回帰判定、Python-Go比較を行う開発・検証用コンポーネントである。探索結果を変更しない。

## 5. 依存規則

### 許可

```text
CORE       → Go標準ライブラリ
BEARING    → COREの中立的な値型
ANCHOR     → CORE, BEARING
BOLTS      → CORE, BEARING
TRUSS      → CORE, ANCHOR公開契約, BOLTS公開契約, BEARING
GATE       → CORE, TRUSS, BEARINGのObserver契約のみ
ULTRASOUND → BEARING, CORE read-only schema
TRAFFIC    → GATE公開API, CORE公開schema, ULTRASOUND公開artifact API
src/products/* → 対応する公開package
SDK        → src/products/cliのstdin/stdout transportのみ
```

### 禁止

- SDKによるバイナリの自動ダウンロードまたは自動更新
- BRIDGE本体によるAPIサーバー提供（APIサーバーはSDK利用者が構築する）
- production packageから`legacy`への依存
- GATEからANCHORまたはBOLTS concrete実装への直接依存
- GATEからULTRASOUND、Collector、Sink、保存実装への依存
- ANCHORからTRUSS、GATE、ULTRASOUND、TRAFFICへの依存
- BOLTSからTRUSS、ANCHOR concrete、GATE、ULTRASOUNDへの依存
- BEARINGからULTRASOUNDへの依存
- ULTRASOUNDから探索制御APIへの依存
- TRAFFICからprivate packageまたはsolver内部状態への依存
- package間循環依存

依存規則は`go list -deps`、architecture test、code reviewで検査する。

## 6. Go実装規則

### 6.1 公開API

- public identifierにはGoDocを付ける
- errorはpanicではなく返り値で伝える
- `context.Context`は最上位引数として渡す
- `time.Duration`を内部deadline表現に使用する
- 外部文字列IDはGATEで連続`NodeID`へ変換する
- enum相当は独自型と定数で表現する

### 6.2 データ構造

- 本番探索では連続`NodeID`とslice中心の表現を優先する
- 大規模静的graphにはCSRを推奨する
- `map[string]any`を主要な内部契約に使用しない
- Graph内部sliceを外部から変更可能な形で公開しない
- queryごとの大規模配列はworkspace再利用とepoch方式を検討する
- priority queue要素の不要なpointer allocationを避ける

### 6.3 並行処理

- goroutineを起動する主体はTRUSSまたは明示的なschedulerとする
- solverは割当Work Budgetを超えない
- shared state更新順を決定論的に定義する
- race detectorを必須検査とする
- observer I/Oで探索goroutineを長時間blockさせない

## 7. Work・Step・品質

用語と計測単位は`docs/WORD_DEFINITION.md`を正本とする。

- Workは意味的探索Actionの総数
- Stepは依存関係を考慮した実行深度
- 逐次実装では`Work = Scheduled Steps`
- 複数taskのWorkは合算する
- 並列taskのStepは実際のscheduleに基づいて合成する
- elapsed time、allocation、GC、I/OはWorkに含めない
- fallbackとalternate hypothesisを区別する
- Distance RatioとCertified Ratioを区別する

## 8. 予算と終了条件

- TRUSSだけがportfolio Work Budgetを所有する
- solverは事前割当sliceを超えない
- `MaxSuboptimality`はquality終了条件へ接続する
- memory budgetを強制できない版は「保証済み」と表記しない
- deadline、cancellation、Work Budgetの終了理由を区別する
- taskごとのWorkとportfolio全体Workを混在させない

## 9. 観測非干渉

ULTRASOUNDの有効・無効で次が変化してはならない。

- path
- distance
- found/exact状態
- WorkとStep
- tie-breaking
- seed消費順
- solver選択
- budget配分

observer例外は探索結果を破壊してはならないが、観測失敗は明示的に記録する。

## 10. 決定論性

同一version、graph、query、mode、budget、workers、seedでは、timing以外のbenchmark結果を再現可能にする。

- adjacency順を正規化する
- priority queueの同順位規則を固定する
- candidate適用順を固定する
- map iteration順へ依存しない
- Stable Digestへ非決定的値を含めない

## 11. テスト要件

変更時に最低限、次を実行する。

```bash
go test ./...
go test -race ./...
go vet ./...
python tests/compatibility/verify.py
python tests/compatibility/evaluate_research_readiness.py
```

必須検査:

- path妥当性
- exact baseline一致
- Work Budget超過0件
- observer非干渉
- repeatability 100%
- architecture dependency test
- Python-Go paired trend評価

## 12. 文書管理

- 現行規則は現在形で記述する
- 過去の名称やPython設計は「履歴」として明示する
- active docsは日本語を標準言語とする
- code identifier、コマンド、schema fieldは原表記を維持する
- 実測値を記載する報告書にはversion、条件、case数を記載する

## 13. 変更手順

アーキテクチャ変更は次の順で行う。

1. 規範文書の変更案を作成
2. 依存・責務への影響を確認
3. codeとtestを変更
4. race、互換性、研究準備性を検証
5. CHANGELOGと報告書を更新

## 14. 最重要原則

```text
TRUSSが判断する。
ANCHORが主探索する。
BOLTSが補助する。
BEARINGが観測契約をつなぐ。
ULTRASOUNDが観測する。
TRAFFICが試験する。
GATEが公開する。
COREは中立である。
```


## 12. SDK配布規則

- Python SDKとTypeScript SDKはBRIDGEの探索を再実装しない。
- SDKは`bridge route`をshellを介さず子プロセスとして起動する。
- SDKにはLinux amd64/arm64、Windows amd64、macOS amd64/arm64の正式バイナリを静的ファイルとして含める。
- バイナリ解決順は、利用者の明示パス、`BRIDGE_BINARY`、同梱バイナリ、PATHとする。
- SDKはバイナリをネットワークから自動ダウンロードせず、自動更新もしない。
- SDKは暗黙にTrace、結果ファイル、Scenarioを生成しない。
- APIサーバーはBRIDGEの製品ではなく、利用者がSDKを組み込んで構築する。


## HEALTHY境界

HEALTHYはTRAFFIC artifactのread-only検証・比較・評価のみを担う。ANCHOR、BOLTS、TRUSSのprivate実装へ依存してはならず、評価結果を同一Run中の探索制御へ戻してはならない。Exact ReferenceはGATEの公開ExecuteOnce経由で取得する。
