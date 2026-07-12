# ANCHOR COMPONENT RULE

## 1. 定義

ANCHORはBRIDGE固有の主Anytime探索エンジンである。TRUSSから指定された探索仮説を実行し、first-path candidateと残予算内のlocal refinementを生成する。

## 2. 所有する責務

- frontier展開、node expansion、edge relaxation
- 指定strategyの探索primitive実行
- geometric corridor、grid detour、portal、hub-aware、weighted-cost等の候補生成
- first pathの発見
- candidate diversityとlocal repair/refinement
- session内のfrontier/parent/candidate state
- task slice内のwork/deadline/cancellation確認
- BEARINGへの詳細探索event発行

## 3. 所有してはならない責務

- graph/query全体からportfolio strategyを決めること
- alternate hypothesis、fallback、reachability、exact solverを独断起動すること
- portfolio全体のbudget/deadline所有
- exact certificationまたは未証明品質の証明
- 最終portfolio result選択
- BOLTS concrete classのimport
- ULTRASOUNDへの直接依存・ファイル出力

## 4. 依存規則

### 許可

- CORE
- ANCHOR自身のPort/interface
- BEARING event/observer契約
- TRUSSから注入された抽象`DetourPort`（導入時のみ）

### 禁止

- TRUSS concrete implementation
- BOLTS concrete implementation
- ULTRASOUND、TRAFFIC、GATE
- observerの戻り値に基づく探索変更

## 5. アルゴリズム規則

- strategyはtask parameterとして受け取り、ANCHOR内部でportfolio再選択しない。
- heuristic score、tentative distance、route cost、boundを別概念として扱う。
- first pathとrefined candidateの発見時点を区別する。
- local repairは割当sliceを超えない。
- emergency unrestricted searchを実装しない。
- exact性は明示的な証明がない限り`False`とする。

## 6. Session規則

- `run_slice()`は割当workを超えない。
- pause/resume可能な実装ではfrontier、parents、distances、logical step、累積workを保存する。
- cancel後に追加node expansionを行わない。
- 同一state/seedからのresumeは決定的である。

## 7. 観測規則

- `logical_step`はtask/lane内のnode expansion ordinalとする。
- `work_used`はtask内累積node expansion数とする。
- `edge_relaxed.new_distance`へheuristic scoreを格納しない。
- event意味はULTRASOUNDの意味論仕様に従うが、ANCHORはULTRASOUNDをimportしない。

## 8. 必須テスト

- 旧PIER paired regression
- strategy別golden case
- hard budget、cancel、deadline
- candidate/path validity
- workと`node_expanded`件数の一致
- trace logical step連続性
- BOLTS/TRUSS/ULTRASOUNDへの禁止依存

## 0. 文書の効力

この文書は当該コンポーネントの実装・レビュー・テスト・将来変更に対する規範である。ルートのアーキテクチャ仕様と矛盾する場合は、より厳格な規則を採用し、矛盾を放置してはならない。

変更時には、コード、公開契約、テスト、観測意味論、本書を同一変更単位で更新する。文書と実装の不一致は不具合として扱う。
