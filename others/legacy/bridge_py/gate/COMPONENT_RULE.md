# GATE COMPONENT RULE

## 1. 定義

GATEはBRIDGEの唯一の外部公開境界であり、外部入力をCORE契約へ正規化し、TRUSSの公開APIだけを呼び出す。

## 2. 所有する責務

- Python API、将来のCLI/REST/RPC/SDK entrypoint
- 外部graphの正規化
- request validationとmode/configurationの正規化
- error mappingおよび外部向けresult表現
- schema/version negotiationの入口
- composition rootから注入されたTRUSSの呼び出し

## 3. 所有してはならない責務

- solver、strategy、fallback、certificationの選択
- budget配分、deadline scheduling
- ANCHOR/BOLTS sessionの直接生成
- trace保存、benchmark判定
- graph探索、path修復

## 4. 依存規則

### 許可

- COREの公開契約
- TRUSSの公開API
- BEARINGのNull Observer等、公開境界の既定構成に必要なinterface

### 禁止

- ANCHOR、BOLTSの直接importまたは起動
- TRUSS private module/private fieldへの依存
- ULTRASOUND具体保存処理への依存
- TRAFFICへの依存

## 5. 公開API規則

- すべてのroute requestは`RouteRequest`へ正規化してからTRUSSへ渡す。
- 外部dictやSDK固有型をTRUSS以下へ流さない。
- GATEは結果を書き換えて品質を高く見せてはならない。
- exceptionは安定した外部errorへ変換し、内部stackやprivate typeを契約化しない。
- default設定は明示し、version間で無断変更しない。

## 6. 不変条件

- GATE経由と同一`RouteRequest`をTRUSSへ直接渡した場合、探索結果の意味は一致する。
- 入力正規化はgraph cost、node identity、constraintsを変質させない。
- 観測の有無はGATEの返却結果を変えない。

## 7. 必須テスト

- 公開modeごとのsmoke test
- invalid request/error mapping
- graph normalization
- GATEがTRUSS以外のsolverを呼ばないこと
- direct TRUSS resultとの意味的同値性
- API後方互換性

## 0. 文書の効力

この文書は当該コンポーネントの実装・レビュー・テスト・将来変更に対する規範である。ルートのアーキテクチャ仕様と矛盾する場合は、より厳格な規則を採用し、矛盾を放置してはならない。

変更時には、コード、公開契約、テスト、観測意味論、本書を同一変更単位で更新する。文書と実装の不一致は不具合として扱う。
