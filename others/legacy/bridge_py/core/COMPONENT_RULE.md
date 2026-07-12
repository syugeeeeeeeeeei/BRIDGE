# CORE COMPONENT RULE

## 1. 定義

COREはBRIDGE全体で共有される値オブジェクト、Graph表現、request/result、budget、bounds、progress、task、cancellationの契約を提供する。COREは制御層でもsolverでもない。

## 2. 所有する責務

- `Graph`、node、edge、座標および正規化済みgraph表現
- `RouteRequest`と入力済み制約の内部表現
- `PathResult`とsolver非依存の結果表現
- `WorkBudget`、`Deadline`、`CancellationToken`
- `SolverTask`、`SolverProgress`、`SharedSearchState`
- upper/lower boundおよびquality certificateの共通値型
- コンポーネント間で交換する不変データ契約

## 3. 所有してはならない責務

- solver選択、strategy選択、fallback判断、終了判断
- frontier操作、edge relaxation、candidate生成
- telemetry保存、trace分析、benchmark実行
- API/CLI固有のserialization
- concrete componentの生成やdependency injection

## 4. 依存規則

### 許可

- Python標準ライブラリ
- CORE内部の相対import

### 禁止

- `gate`、`truss`、`anchor`、`bolts`、`bearing`、`ultrasound`、`traffic`への依存
- runtime callbackやobserverの保持
- solver固有enumや具体classの取り込み

## 5. 公開契約

- 公開型は原則としてdataclass、Protocolまたは純粋関数とする。
- fieldの単位、nullability、default、validation条件を明示する。
- mutable objectを共有する場合は所有権を明記する。
- `PathResult.distance`はpathのedge cost総和を表し、heuristic scoreを格納してはならない。
- `work_expanded_nodes`はportfolioで集計されたnode expansion数を表す。

## 6. 不変条件

- edge weightは有限かつ非負である。
- found resultはsourceからtargetまで連続したpathを持つ。
- not-found resultのdistanceは原則`inf`とする。
- exact resultは証明可能なsolverまたは証明済みboundに由来する。
- budget値は負数を受け入れない。
- COREの型は観測有無によって意味が変化しない。

## 7. 変更管理

- field追加は後方互換性を確認する。
- field削除、意味、単位、nullability変更は破壊的変更とする。
- solver固有情報は`telemetry`へ無制限に追加せず、共有意味が確立した場合のみ正式fieldへ昇格する。

## 8. 必須テスト

- graph validation
- request validation
- path/result整合性
- bounds計算
- budget境界値
- serialization可能性
- 他コンポーネントへのimportがないこと

## 0. 文書の効力

この文書は当該コンポーネントの実装・レビュー・テスト・将来変更に対する規範である。ルートのアーキテクチャ仕様と矛盾する場合は、より厳格な規則を採用し、矛盾を放置してはならない。

変更時には、コード、公開契約、テスト、観測意味論、本書を同一変更単位で更新する。文書と実装の不一致は不具合として扱う。
