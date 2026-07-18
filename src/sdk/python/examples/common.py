REQUEST = {
    "schema_version": "bridge.route.request.v1",
    "request_id": "sdk-example",
    "graph": {
        "type": "inline",
        "directed": False,
        "nodes": [{"id": 0}, {"id": 1}, {"id": 2}],
        "edges": [
            {"from": 0, "to": 1, "weight": 1},
            {"from": 1, "to": 2, "weight": 1},
            {"from": 0, "to": 2, "weight": 5},
        ],
    },
    "route": {"source": 0, "target": 2, "logical_worker_count": 1, "seed": 42},
    "budget": {"total_work": 100000, "timeout_ms": 5000},
    "observation_config": {"level": "minimum", "sample_rate": 1},
}
