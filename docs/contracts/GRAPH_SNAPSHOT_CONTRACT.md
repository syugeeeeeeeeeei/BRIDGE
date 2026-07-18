# Graph Snapshot Contract

> Status: Normative  
> Applies To: `bridge.benchmark.graph_snapshot.v1`  
> Producer: TRAFFIC  
> Consumers: Simulator, HEALTHY

## 目的

`graph.json`は、当該Runで実際に使用したGraphとQueryを再構成するための完全Snapshotです。Generator設定の再実行結果を代替として使用してはなりません。

## 必須情報

- `schema_version`
- `graph`: GATEのinline Graph入力（`directed`、`nodes`、`edges`）
- `source`
- `target`

`graph.edges`の各Edgeは`from`、`to`、`weight`を持ちます。Weightは有限の0以上の数値です。Node IDはJSON整数です。

無向Graphでは、同一の無向辺を重複して格納しないことを推奨します。Consumerは`directed=false`の場合に双方向移動可能として解釈します。

座標は任意です。座標がない場合、Simulatorは入力Graphから決定論的にLayoutを生成します。

## 機械契約

`src/contracts/json-schema/benchmark-graph-snapshot-v1.schema.json`
