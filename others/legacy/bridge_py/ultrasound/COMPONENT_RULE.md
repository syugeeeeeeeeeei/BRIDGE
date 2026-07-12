# ULTRASOUND COMPONENT RULE

## 1. 定義

ULTRASOUNDはBEARING eventを収集、検証、保存、集計、再生する開発・検証専用観測基盤である。本番routeの必須依存ではない。

## 2. 所有する責務

- observer adapter
- sequence、relative time等のartifact metadata付与
- trace metrics集計
- JSONL/binary等への保存と読込
- schema/semantic validation
- replay、snapshot、seek、分析の将来拡張
- `TRACE_SEMANTICS.md`と実行可能意味論レジストリの管理

## 3. 所有してはならない責務

- solver、strategy、budget、candidate、終了判断
- graphまたはsolver stateの変更
- TRUSS control APIの呼び出し
- event producerのprivate state参照
- 本番routeに必須の処理

## 4. 依存規則

### 許可

- BEARING公開event/observer契約
- CORE read-only schema
- ULTRASOUND内部validator/serializer

### 禁止

- TRUSS、ANCHOR、BOLTS、GATE、TRAFFICの制御/具体実装
- private queue、frontier、parent mapへの依存

## 5. 意味論ガバナンス

- すべてのevent/fieldに定義、単位、producer、生成時点、null意味、不変条件、禁止解釈を持たせる。
- 未定義fieldを永続化しない。
- schema version不一致を黙って受け入れない。
- trace保存前と読込後にvalidationを行う。
- heuristic scoreをdistance、logical stepをwall time、laneを物理threadとして解釈しない。
- derived metricは計算式と入力fieldを明示する。

## 6. Artifact規則

- `sequence`はartifact内で1から連続する。
- `relative_ns`はobserver起点からの単調時間であり性能保証値ではない。
- event orderingはappend順を表し、物理的同時性を証明しない。
- artifactにはschema version、producer version、環境情報を関連付ける。
- validation errorのあるtraceを正式成果物として扱わない。

## 7. 非干渉・性能規則

- observerの有無で探索結果を変えない。
- OFF時は本番依存を最小化する。
- trace levelごとのevent filtering規則を明示する。
- overheadはモード別にTRAFFICで定量測定する。

## 8. 必須テスト

- 全event kindのsemantic registry登録
- sequence/logical step/work/bounds/relaxation整合性
- invalid artifact拒否
- JSONL round trip
- observer ON/OFF非干渉
- workとnode-expanded件数の対応
- schema migration test

## 0. 文書の効力

この文書は当該コンポーネントの実装・レビュー・テスト・将来変更に対する規範である。ルートのアーキテクチャ仕様と矛盾する場合は、より厳格な規則を採用し、矛盾を放置してはならない。

変更時には、コード、公開契約、テスト、観測意味論、本書を同一変更単位で更新する。文書と実装の不一致は不具合として扱う。
