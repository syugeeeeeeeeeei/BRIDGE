# BOLTS COMPONENT RULE

## 1. 定義

BOLTSはdetour、repair補助、fallback、reachability、exact探索、lower bound、quality certificationを提供する交換可能なsolver群である。

## 2. 所有する責務

- 公開capabilityを持つ独立solver実装
- Dijkstra、双方向Dijkstra、A*、reachability、emergency approximate等
- task slice内の探索、work/time/memory報告
- path、bound、exact性、failure reasonの返却
- pause/resume/cancel対応可能性の宣言と履行
- BEARINGへのsolver event発行

## 3. 所有してはならない責務

- portfolio budget配分
- ANCHOR継続、fallback chain、最終終了の判断
- 他BOLTS solverの独断起動
- ANCHOR private stateの操作
- GATEへの直接公開
- ULTRASOUNDへの直接依存

## 4. 依存規則

### 許可

- CORE
- BOLTSの公開Port/capability
- BEARING event/observer契約

### 禁止

- TRUSS、ANCHOR concrete implementation
- GATE、ULTRASOUND、TRAFFIC
- 他solverのprivate queue/parent map

## 5. Capability規則

- `exact=True`は入力契約を含めて最適性が保証される場合のみ設定する。
- A*はheuristic admissibilityが保証できない場合exactを宣言しない。
- `resumable=True`は完全なstate保存・復元を実装した場合のみ設定する。
- solver名と実アルゴリズムを一致させる。単方向Dijkstraを双方向と称してはならない。

## 6. 予算・終了規則

- node expansion loop内でwork、deadline、cancellationを確認する。
- budget超過後に結果を完走させてはならない。
- partial resultとexact resultを明確に区別する。
- reachabilityのnot-found exact resultはcomponent非接続の証明を意味する。

## 7. 観測規則

- task ID、phase、lane、logical stepを保持する。
- candidate eventはfound、distance、path length、solverを完全に記録する。
- lower bound/upper bound/certificateの意味を混同しない。

## 8. 必須テスト

- solverごとのgolden correctness
- exact capability検証
- hard budget/deadline/cancel
- disconnected/reachability
- pause/resume宣言整合性
- solver名と実装の一致
- 禁止依存検査

## 0. 文書の効力

この文書は当該コンポーネントの実装・レビュー・テスト・将来変更に対する規範である。ルートのアーキテクチャ仕様と矛盾する場合は、より厳格な規則を採用し、矛盾を放置してはならない。

変更時には、コード、公開契約、テスト、観測意味論、本書を同一変更単位で更新する。文書と実装の不一致は不具合として扱う。
