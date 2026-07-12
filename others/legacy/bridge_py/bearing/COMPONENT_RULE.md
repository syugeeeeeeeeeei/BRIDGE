# BEARING COMPONENT RULE

## 1. 定義

BEARINGは探索層と観測層を隔離する、非干渉なevent・observer契約である。BEARING自身は保存も分析も制御もしない。

## 2. 所有する責務

- event vocabularyと型付きpayload
- observer Protocol
- Null Observer、Safe Observer
- phase、lane、logical step、metric名の共通契約
- event schema version連携点
- producerが守るべき最小field型

## 3. 所有してはならない責務

- JSONL/binary/CSV保存
- metrics集計、trace replay、可視化
- solver selection、budget変更、cancel指示
- graph/solver stateへの参照や変更
- ULTRASOUND concrete adapterのimport

## 4. 依存規則

### 許可

- Python標準ライブラリ
- 必要最小限のCORE read-only型

### 禁止

- TRUSS、ANCHOR、BOLTS、GATE、ULTRASOUND、TRAFFICへの依存
- callbackの戻り値を制御へ返す設計

## 5. Event設計規則

- event名は観測事実を表し、推測・評価・命令を混在させない。
- fieldは単位、生成時点、null意味、producerを定義できるものだけ追加する。
- route cost、heuristic score、bound、work、timeを別fieldとして扱う。
- event dataclassは原則immutableまたは生成後変更禁止とする。
- observer methodの戻り値は常に`None`とする。

## 6. 非干渉規則

- observerは乱数を消費しない。
- observer例外はSafe Observerで隔離し、探索結果を変えない。
- observerはbudget/workを消費した扱いにしない。
- ON/OFFでpath、distance、work、tie-break、seed消費順を変えない。

## 7. 変更管理

- event/field追加時はULTRASOUND意味論レジストリと文書を同時更新する。
- fieldの意味・単位・生成時点変更はschema breaking changeとする。
- producer未実装のeventを契約へ追加したまま放置しない。

## 8. 必須テスト

- Null/Safe Observer contract
- 全eventの型/必須field
- observer exception isolation
- 非干渉
- 禁止依存
- ULTRASOUND semantic registryとの完全対応

## 0. 文書の効力

この文書は当該コンポーネントの実装・レビュー・テスト・将来変更に対する規範である。ルートのアーキテクチャ仕様と矛盾する場合は、より厳格な規則を採用し、矛盾を放置してはならない。

変更時には、コード、公開契約、テスト、観測意味論、本書を同一変更単位で更新する。文書と実装の不一致は不具合として扱う。
