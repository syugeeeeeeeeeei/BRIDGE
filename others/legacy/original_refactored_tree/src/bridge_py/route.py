from __future__ import annotations

from typing import Any, Dict

from .gate import Gate
from .core import RouteRequest
from .graph import Graph, normalize_graph
from .solvers.dijkstra import bidirectional_dijkstra
from .solvers.mprc import mprc
from .solvers.mrpc_cg import mrpc_cg, mrpc_dg4_handoff, mrpc_dg5_switchback, build_component_index
from .solvers.mrpc_dg6 import mrpc_dg6
from .solvers.pier import pier
from .types import Adjacency, Node, PathResult


def route(
    G: Graph | Adjacency,
    source: Node,
    target: Node,
    mode: str = "auto",
    constraints: Dict[str, Any] | None = None,
    *,
    max_suboptimality: float | None = None,
    deadline_ms: float | None = None,
    work_budget: int | None = None,
    memory_budget_kib: float | None = None,
    workers: int | None = None,
    seed: int = 0,
) -> PathResult:
    """Route a query through BRIDGE or an explicitly selected legacy solver.

    Public modes are fast, balanced, quality and exact. ``auto`` maps to
    balanced. Legacy modes remain available for reproducible research.
    """
    graph = normalize_graph(G)
    constraints = dict(constraints or {})
    workers = int(workers if workers is not None else constraints.pop("workers", 1))

    if mode == "auto":
        mode = "balanced"
    if mode in {"fast", "balanced", "quality", "exact"}:
        req = RouteRequest(
            source=source,
            target=target,
            mode=mode,
            max_suboptimality=max_suboptimality if max_suboptimality is not None else constraints.pop("max_suboptimality", constraints.pop("max_distance_ratio", None)),
            deadline_ms=deadline_ms if deadline_ms is not None else constraints.pop("deadline_ms", None),
            work_budget=work_budget if work_budget is not None else constraints.pop("work_budget", None),
            memory_budget_kib=memory_budget_kib if memory_budget_kib is not None else constraints.pop("memory_budget_kib", None),
            workers=workers,
            seed=seed,
            constraints=constraints,
        )
        return Gate().route_request(graph, req)

    if mode in {"pier", "pier_v0.1"}:
        return pier(graph, source, target, workers=workers, **constraints)
    if mode == "mrpc_cg":
        return mrpc_cg(graph, source, target, workers=workers, max_distance_ratio=constraints.get("max_distance_ratio"))
    if mode in {"mrpc_dg6", "dg6"}:
        return mrpc_dg6(graph, source, target, **constraints)
    if mode in {"mrpc_dg5", "mrpc_switchback"}:
        return mrpc_dg5_switchback(
            graph, source, target,
            workers=workers,
            budget_ratio=float(constraints.get("budget_ratio", 0.10)),
            local_exact_ratio=float(constraints.get("local_exact_ratio", 0.20)),
            global_fallback_ratio=float(constraints.get("global_fallback_ratio", 1.00)),
            max_switches=int(constraints.get("max_switches", 3)),
            component_index=constraints.get("component_index"),
            enable_component_precheck=bool(constraints.get("component_precheck", True)),
            safe_exact_completion=bool(constraints.get("safe_exact_completion", True)),
            enable_topology_gate=bool(constraints.get("enable_topology_gate", True)),
            dg5_mode=str(constraints.get("dg5_mode", "balanced")),
        )
    if mode in {"mrpc_dg4", "mrpc"}:
        component_index = constraints.get("component_index")
        if component_index is None and constraints.get("component_precheck", True):
            component_index = build_component_index(graph)
        return mrpc_dg4_handoff(graph, source, target, workers=workers, component_index=component_index)
    if mode == "mprc":
        return mprc(graph, source, target, workers=workers)
    if mode == "bidirectional_dijkstra":
        return bidirectional_dijkstra(graph, source, target)
    raise ValueError(f"unsupported route mode: {mode}")


def path(G: Graph | Adjacency, source: Node, target: Node, mode: str = "auto"):
    return route(G, source, target, mode).path
