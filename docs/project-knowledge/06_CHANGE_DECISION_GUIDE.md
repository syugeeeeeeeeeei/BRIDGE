# BRIDGE変更時の判断ガイド

## 1. 変更前に確認する問い

新機能、最適化、リファクタリング、用語変更を行う前に、次を確認する。

1. この変更の目的は何か。
2. どのコンポーネントが責務を所有するべきか。
3. 既存の責務境界を越えていないか。
4. portfolio予算の所有権を侵害しないか。
5. WorkとStepの意味を変えないか。
6. 観測有無で結果が変わらないか。
7. 決定論性を維持できるか。
8. 公開契約、schema、用語集へ影響するか。
9. 製品経路とbenchmark経路を分岐させていないか。
10. 正しさと性能をどのテストで証明するか。

## 2. 責務配置の判断

### GATEへ置く

外部入力検証、外部ID変換、公開API表現、error mappingである場合。

### TRUSSへ置く

solver選択、task生成、予算配分、継続、fallback、certification、最終候補選択である場合。

### ANCHORへ置く

BRIDGE固有の主探索Hypothesis、Candidate生成、局所Repairである場合。

### BOLTSへ置く

交換可能な補助solver、exact探索、reachability、lower bound、certification能力である場合。

### BEARINGへ置く

探索から観測へ渡す中立的なtyped event契約である場合。

### ULTRASOUNDへ置く

event収集、保存、replay、metrics、profile等の観測処理である場合。

### TRAFFICへ置く

Scenario、Run、graph/query生成、benchmark、比較、統計、acceptanceである場合。

### HEALTHYへ置く

生成済みartifactのread-onlyな整合性、Work、Ledger、trace再構成検証である場合。

### COREへ置く

複数コンポーネントが共有し、特定アルゴリズムや外部I/Oへ依存しない中立契約である場合。

## 3. 変更が拒否される条件

次のいずれかに該当する変更は、そのまま採用しない。

- ANCHORがBOLTSを直接起動する。
- BOLTSがportfolio全体の切替えを決める。
- GATEがsolver内部を選択または操作する。
- ULTRASOUNDのeventが探索判断へ帰還する。
- TRAFFICがprivate APIを通じて有利な状態を注入する。
- benchmark専用の別アルゴリズム経路を作る。
- Work未計上の探索Actionを増やす。
- observer有効時だけ結果が改善または悪化する。
- exactでない結果をexactと表記する。
- baselineを通常探索へ入力する。
- 失敗Runを理由なく除外する。
- 未定義用語を公開契約へ追加する。

## 4. アルゴリズム追加

新しいsolverをBOLTSへ追加する場合、少なくとも次を満たす。

- 共通入力・出力契約を使用する。
- Budget Sliceを超えない。
- 共通Work Actionで計測する。
- Step計測規則を明示する。
- cancellationとdeadlineを扱う。
- tie-breakingを決定論的にする。
- path妥当性を検証する。
- observer非干渉を確認する。
- solver固有のprivate metricを共通Workへ混ぜない。
- TRAFFICから同一条件で比較できる。

ANCHORへ新しいHypothesisを追加する場合、さらに次を確認する。

- 主探索の目的に適合する。
- 別solverの隠れた呼出しになっていない。
- Candidate生成とportfolio判断を混同しない。
- 既存Hypothesisとの重複Workを測定できる。
- アブレーション可能である。

## 5. 観測項目追加

新しいeventやmetricを追加する場合、次を確認する。

- 何を表すかが明確である。
- 所有componentが明確である。
- 単位と集計範囲が明確である。
- Workに含むか含まないかが明確である。
- modeごとの取得可否が明確である。
- replayに必要か、profile専用かを区別する。
- schema versionと互換性を検討する。
- event欠損時の扱いを定義する。
- 取得の有無で探索結果が変わらない。

## 6. 用語変更

用語変更時は、次を同時に確認する。

- 正式定義
- 含むものと含まないもの
- 所有component
- 対応するfield名
- 類似語との差異
- schema
- code identifier
- Scenario
- trace
- test
- 既存artifact互換性

既存語の意味を静かに変更しない。破壊的変更であれば、versionまたはmigration方針を明示する。

## 7. 性能最適化

最適化は、意味を変えずに測定可能な改善として行う。

- 最適化前後でpath、distance、状態、Work定義を比較する。
- wall-clock timeだけでなくallocation、memory、Work、Stepを測る。
- mapからslice、object allocation削減、workspace再利用等は、決定論性と安全性を確認する。
- 並列化はWork削減と混同せず、Step削減として評価する。
- instrumentation削減は観測modeの契約内で行う。
- 最適化のためにprivate stateを他componentへ漏らさない。

## 8. 文書更新

変更内容に応じて更新対象を選ぶ。

### 恒久原則が変わる

- 最上位アーキテクチャ規則
- 本知識ベース
- 関連コンポーネント規則
- 用語集

### 公開契約が変わる

- schema
- API仕様
- 用語集
- 互換性試験
- migration情報

### 実装だけが変わる

- code
- test
- CHANGELOG
- 必要に応じて実装報告

### 性能値が変わる

- benchmark artifact
- 評価報告

性能値を本知識ベースへ固定値として書かない。

## 9. 完了判定

変更は、実装が存在するだけでは完了しない。

- 責務境界が正しい
- 依存方向が正しい
- public contractが整合する
- 用語が定義されている
- testが追加されている
- budget違反がない
- Work保存則が成立する
- observer非干渉が成立する
- 決定論性が成立する
- 既存機能に回帰がない
- 文書とコードが一致する

これらを確認して初めて、BRIDGEとして安全に変更されたとみなす。
