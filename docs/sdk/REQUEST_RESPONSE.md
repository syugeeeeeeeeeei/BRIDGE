# Route Request / Response

最小Request例:

````json
{
  "schema_version": "bridge.route.request.v1",
  "request_id": "example-001",
  "graph": {
    "type": "inline",
    "directed": false,
    "nodes": [{"id": 0}, {"id": 1}, {"id": 2}],
    "edges": [
      {"from": 0, "to": 1, "weight": 1},
      {"from": 1, "to": 2, "weight": 1},
      {"from": 0, "to": 2, "weight": 5}
    ]
  },
  "route": {
    "source": 0,
    "target": 2,
    "route_mode": "balanced",
    "logical_worker_count": 1,
    "seed": 42
  },
  "budget": {"total_work": 100000, "timeout_ms": 5000},
  "observation_config": {"level": "minimum", "sample_rate": 1}
}
````

正式Observation Levelは`minimum`、`debug`、`trace`です。

`path_found=false`、`unreachable`、Budget到達は、正常に計算された探索結果であり、通信例外ではありません。`result.status`、証明フィールド、Work情報を確認してください。
