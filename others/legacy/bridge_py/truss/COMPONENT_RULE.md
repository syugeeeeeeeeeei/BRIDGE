# TRUSS COMPONENT RULE

## 1. 定義

TRUSSはBRIDGE全体の唯一の制御層であり、query profiling、実行計画、portfolio予算、solver orchestration、fallback、certification、終了判断、最終候補選択を所有する。

## 2. 所有する責務

- query profileと`AnchorPlan`の生成
- ANCHOR/BOLTS taskの生成、順序、purpose、budget slice決定
- portfolio全体のwork、deadline、worker、memory制約の単一所有
- shared upper/lower bound管理
- stagnation、継続価値、fallback、certificationの判断
- candidate集合からの最終result選択
- task traceおよびportfolio-level telemetry

## 3. 所有してはならない責務

- frontier/priority queue操作
- node expansion、edge relaxation、parent map操作
- corridor、portal、hub等の候補探索実装
- Dijkstra/A*/reachabilityの探索実装
- JSONL保存やtrace分析
- 外部request形式の解釈

## 4. 依存規則

### 許可

- CORE
- ANCHORの公開Port
- BOLTSの公開Port/registry
- BEARING observer契約

### 禁止

- ANCHOR/BOLTS private stateへのアクセス
- ULTRASOUNDへの直接依存
- GATE、TRAFFICへの依存
- observer出力を制御入力として使用すること

## 5. 予算規則

- portfolio budgetの唯一の真実はTRUSS stateとする。
- task開始前にsliceを確定し、solverへ明示する。
- 残予算0でtaskを起動しない。
- solver報告workを無条件に信用せず、契約違反を検出する。
- task/portfolio budget超過を正常動作として扱わない。
- deadline超過後に新規taskを開始しない。

## 6. 判断規則

- strategy選択理由を`AnchorPlan.reason`等で記録する。
- alternate ANCHOR hypothesis、reachability、fallback、certificationは明示的taskとする。
- ANCHOR failureとdisconnectedを混同しない。
- exact/quality certificationはBOLTS結果とboundsに基づく。
- final resultは候補の有効性、距離、exact性、制約適合を比較して選ぶ。

## 7. 観測規則

- budget、phase、bound、task selectionをBEARING経由で通知する。
- eventは事後説明用であり、observerから戻り値を受け取らない。
- portfolio workとtask workの意味を混在させない。

## 8. 必須テスト

- portfolio hard budget
- deadlineでの新規task停止
- solver差し替えcontract
- strategy/fallback/certification ownership
- disconnected判定
- candidate selection
- observer ON/OFF非干渉
- ANCHOR/BOLTS private moduleへ依存しないこと

## 0. 文書の効力

この文書は当該コンポーネントの実装・レビュー・テスト・将来変更に対する規範である。ルートのアーキテクチャ仕様と矛盾する場合は、より厳格な規則を採用し、矛盾を放置してはならない。

変更時には、コード、公開契約、テスト、観測意味論、本書を同一変更単位で更新する。文書と実装の不一致は不具合として扱う。
