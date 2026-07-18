# BRIDGEが守るべき不変条件とルール

## 1. 予算所有権

- portfolio全体のWork BudgetはTRUSSだけが所有する。
- 各solverは、割り当てられたBudget Sliceを超えてはならない。
- task単位、component単位、portfolio全体のWorkを混同しない。
- deadline、cancellation、Work Budget到達、品質達成を別の終了理由として記録する。
- 強制できない予算を保証済みと表現しない。

## 2. WorkとStepの意味

- Workは意味的な探索Actionの総数である。
- CPU命令数、関数呼出し数、経過時間、I/O時間、GC、mutex待機はWorkではない。
- Stepは、依存関係を保ちながら同時実行できるWorkをまとめた論理実行段階である。
- 並列性を主張する場合、Work、Logical Step、Scheduled Step、Worker数を区別する。
- Work定義は全アルゴリズムで共通でなければならない。
- アルゴリズムごとに有利な独自カウントを導入してはならない。

## 3. 観測非干渉

Observation Modeやobserverの有効・無効によって、次を変化させてはならない。

- path
- distance
- found、exact、quality-certified等の意味的状態
- WorkとStep
- tie-breaking
- seed消費順
- solver選択
- budget配分
- candidate採用

観測失敗は記録するが、探索結果を破壊しない。観測I/Oやprofile overheadは探索Workへ含めない。

## 4. 決定論性

同一の実装版、Graph、Query、Mode、Budget、Worker数、Seedでは、timing以外の意味的結果を再現可能にする。

- adjacency順を正規化する。
- 同順位candidateのtie-breakingを固定する。
- map等の非決定的iteration順へ依存しない。
- candidate適用順を固定する。
- Stable Digestへ経過時間、address、非決定的IDを含めない。
- randomizeが必要な場合、seedと消費規則を明示する。

## 5. 正しさと品質保証

- Graph上に存在しないedgeを含むpathを返してはならない。
- 到達不能と未発見を区別する。
- exactは、exact solverまたは同等の証明がある場合だけ主張する。
- Distance Ratioは評価値であり、実行中の品質証明ではない。
- Certified RatioはUpper BoundとLower Boundに基づく証明値である。
- First Path、Best Path、Final Pathを区別する。

## 6. 責務分離

- TRUSSはfrontier操作を行わない。
- ANCHORはBOLTSを直接起動しない。
- BOLTSはportfolio制御を行わない。
- GATEはsolver選択を行わない。
- BEARINGは観測事実を制御指示へ変換しない。
- ULTRASOUNDは探索判断を変更しない。
- TRAFFICはprivate stateを利用して評価対象を有利にしない。
- HEALTHYはartifactを変更せず、評価結果を同一Runへ戻さない。
- COREは特定アルゴリズムや外部I/Oへ依存しない。

## 7. 本番経路とbenchmark経路

- benchmark専用の特別な探索ロジックを製品経路とは別に実装しない。
- 同じ公開契約と同じsolver実装を使用する。
- benchmarkでは低オーバーヘッドの一回実行経路を使用できるが、意味的な探索経路は通常利用と共通にする。
- 評価の都合で本番コードにshortcut、正解注入、特別caseを追加しない。

## 8. BRIDGE実行境界の原則

BRIDGEの処理およびbenchmark計測は、実際にBRIDGEを外部システムまたはアプリケーションへ組み込んだ場合の責務境界に基づき、`Preparing`、`Running`、`Finalizing`の三つへ分類する。

- `Preparing`は、BRIDGEへ渡す入力を外部で準備する区間である。Scenario解釈、Graph生成、Query生成、実行条件の確定を含む。外部システム、地図アプリケーション、生成script、TRAFFICによるGraph生成はBRIDGEの実行時間へ含めない。
- `Running`は、BRIDGEが入力を受理してから、経路結果および要求されたTrace結果を外部利用可能な形式で返却するまでの区間である。入力検証、内部Graph構築、Graph分析、特徴抽出、方針決定、経路探索、結果構築、Trace生成・直列化・返却を含む。
- `Finalizing`は、BRIDGEが返却した結果を外部で集計、分析、評価、保存または可視化する区間である。複数Run集計、HEALTHY評価、性能回帰判定、bottleneck分析、artifact生成、外部SimulatorによるTrace可視化を含む。

処理の所属は、実装コンポーネントではなく、実運用上の責務と公開interface境界によって決定する。benchmark実装上の都合を計測境界の正本としてはならない。

- Graph生成時間をBRIDGEの実行時間として扱わない。
- 外部GraphをBRIDGE内部表現へ変換する時間をBRIDGEの実行時間から除外しない。
- Graph分析および特徴抽出はRunningに含めるが、RouteまたはSolverの時間とは分離して記録する。
- Trace取得、構築、直列化および返却はRunningに含める。Traceを用いた描画、動画化、比較分析はFinalizingに含める。
- 経路探索性能、BRIDGE全体性能、benchmark全体性能を別の計測値として保持する。

## 9. baselineの隔離

- Exact Baselineは正解との比較に使用する。
- Performance Baselineは性能比較に使用する。
- Regression Baselineは変更前との比較に使用する。
- Reference Implementationは意味的整合性の確認に使用する。
- baselineをANCHORのcandidate生成、TRUSSの通常判断、BOLTSの探索順へ入力しない。

## 10. traceの完全性

- replay可能と主張するtraceは、状態遷移を再構成できる必要がある。
- sampling、drop、truncation、破損、digest不一致があるtraceを完全traceとして扱わない。
- trace schemaはversion管理する。
- 未知eventを安全に無視できる後方互換性を優先する。
- traceは診断logと区別する。

## 11. テストと検証

変更時には、変更対象に応じて次を確認する。

- 単体テスト
- path妥当性
- budget境界
- cancellationとdeadline
- 決定論性
- observer非干渉
- dependency rule
- exact baseline一致
- Work保存則
- trace再構成
- 既存実装またはReference Implementationとのpaired comparison
- topology、規模、seedを変えた回帰

## 12. 文書と用語

- BRIDGE固有の意味を持つ用語は正式用語集へ定義する。
- schema field、CLI option、trace vocabulary、公開resultへ未定義語を追加しない。
- 用語変更は文言変更ではなく契約変更として扱う。
- 文書とコードの不一致は、どちらかを暗黙に正しいものとして扱わず、欠陥として修正する。
- 報告書の時点依存事実を恒久仕様へ昇格させない。

## 13. 禁止される設計上の近道

- exact solverの結果をANCHORの性能として計上する。
- 観測eventからsolver切替えを行う。
- timeout内に収めるためWorkを未計上にする。
- benchmarkでのみWork定義を変更する。
- 失敗Runを黙って集計から除外する。
- warm-up Runを通常Runと混ぜる。
- private implementationへ依存した評価を行う。
- 実行時間だけで優劣を結論づける。
- 特定seedまたは特定topologyだけで一般性を主張する。
