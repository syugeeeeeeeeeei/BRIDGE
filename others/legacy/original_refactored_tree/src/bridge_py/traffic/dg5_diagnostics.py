from __future__ import annotations

import argparse
import json
import math
from dataclasses import asdict
from pathlib import Path
from typing import Any

from ..graph import Graph
from ..solvers.dijkstra import bidirectional_dijkstra, dijkstra
from ..solvers.mrpc_cg import mrpc_dg5_switchback


def _safe(value: Any) -> Any:
    if isinstance(value, float) and not math.isfinite(value):
        return str(value)
    if isinstance(value, dict):
        return {str(k): _safe(v) for k, v in value.items()}
    if isinstance(value, (list, tuple)):
        return [_safe(v) for v in value]
    return value


def diagnose_graph(
    graph: Graph,
    source: Any,
    target: Any,
    *,
    trace_level: int = 2,
    trace_sample_every: int = 1,
    workers: int = 4,
) -> dict[str, Any]:
    """Run cold-start DG5 and exact baselines with preprocessing included."""
    dg5 = mrpc_dg5_switchback(
        graph,
        source,
        target,
        workers=workers,
        component_index=None,
        enable_component_precheck=True,
        trace_level=trace_level,
        trace_sample_every=trace_sample_every,
    )
    dij = dijkstra(graph, source, target)
    bidir = bidirectional_dijkstra(graph, source, target)
    exact_distance = bidir.distance if bidir.found else math.inf

    def result_row(result, *, preprocessing_time_ms: float = 0.0, preprocessing_work: int = 0) -> dict[str, Any]:
        tel = dict(result.telemetry)
        total_time = float(tel.get("total_time_ms", result.time_ms))
        total_work = int(tel.get("total_work_including_preprocessing", result.total_work + preprocessing_work))
        ratio = result.distance / exact_distance if result.found and math.isfinite(exact_distance) and exact_distance > 0 else (1.0 if result.found and exact_distance == 0 else math.inf)
        return {
            "solver": result.solver_name,
            "found": result.found,
            "distance": result.distance,
            "distance_ratio": ratio,
            "preprocessing_time_ms": float(tel.get("preprocessing_time_ms", preprocessing_time_ms)),
            "query_time_ms": float(tel.get("query_time_ms", total_time - preprocessing_time_ms)),
            "total_time_ms": total_time,
            "preprocessing_work": int(tel.get("preprocessing_work", preprocessing_work)),
            "query_work": result.total_work,
            "total_work": total_work,
            "parallel_steps": result.parallel_steps,
            "peak_memory_kib": result.peak_memory_kib,
            "error_code": tel.get("error_code"),
        }

    return _safe({
        "schema_version": "bridge-dg5-diagnostic-v1",
        "source": source,
        "target": target,
        "comparison": [result_row(dg5), result_row(dij), result_row(bidir)],
        "dg5_telemetry": dg5.telemetry,
    })


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description="Run DG5 diagnostic logging on a JSON graph.")
    p.add_argument("graph", help="JSON file with edges [[u,v,w], ...] and optional pos mapping")
    p.add_argument("source")
    p.add_argument("target")
    p.add_argument("-o", "--output", default="dg5_diagnostic.json")
    p.add_argument("--trace-level", type=int, choices=[0, 1, 2], default=2)
    p.add_argument("--sample-every", type=int, default=1)
    p.add_argument("--workers", type=int, default=4)
    args = p.parse_args(argv)
    raw = json.loads(Path(args.graph).read_text(encoding="utf-8"))
    graph = Graph.from_edges([tuple(e) for e in raw["edges"]], pos=raw.get("pos"), directed=bool(raw.get("directed", False)))
    report = diagnose_graph(graph, args.source, args.target, trace_level=args.trace_level, trace_sample_every=args.sample_every, workers=args.workers)
    Path(args.output).write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
