from __future__ import annotations

import math
import os
import time
import tracemalloc
from concurrent.futures import ProcessPoolExecutor, as_completed
from typing import List, Tuple

from ..graph import Graph, euclidean
from ..types import Corridor, Node, PathResult
from .dijkstra import dijkstra


_WORKER_GRAPH: Graph | None = None
_WORKER_SOURCE: Node | None = None
_WORKER_TARGET: Node | None = None


def _init_corridor_worker(G: Graph, source: Node, target: Node) -> None:
    global _WORKER_GRAPH, _WORKER_SOURCE, _WORKER_TARGET
    _WORKER_GRAPH = G
    _WORKER_SOURCE = source
    _WORKER_TARGET = target


def _search_corridor_worker(c: Corridor) -> tuple[Corridor, PathResult]:
    if _WORKER_GRAPH is None:
        raise RuntimeError("MPRC worker graph is not initialized.")
    r = dijkstra(
        _WORKER_GRAPH,
        _WORKER_SOURCE,
        _WORKER_TARGET,
        allowed_nodes=set(c.nodes),
        solver_name=f"mprc:{c.corridor_id}",
    )
    return c, r


def _line_distance_and_t(p, a, b) -> Tuple[float, float]:
    ax, ay = a; bx, by = b; px, py = p
    vx, vy = bx - ax, by - ay
    denom = vx * vx + vy * vy
    if denom == 0:
        return euclidean(p, a), 0.0
    t = ((px - ax) * vx + (py - ay) * vy) / denom
    proj = (ax + t * vx, ay + t * vy)
    return euclidean(p, proj), t


def _offset_distance(p, a, b, offset: float) -> Tuple[float, float]:
    ax, ay = a; bx, by = b
    vx, vy = bx - ax, by - ay
    norm = math.hypot(vx, vy)
    if norm == 0:
        return euclidean(p, a), 0.0
    nx, ny = -vy / norm, vx / norm
    aa = (ax + nx * offset, ay + ny * offset)
    bb = (bx + nx * offset, by + ny * offset)
    return _line_distance_and_t(p, aa, bb)


def generate_corridors(G: Graph, source: Node, target: Node, k_corridors: int = 7, width_scale: float = 0.12) -> List[Corridor]:
    if not G.pos:
        return [Corridor("corridor_all", math.inf, 0.0, G.nodes())]
    a, b = G.pos[source], G.pos[target]
    base = max(euclidean(a, b), 1e-12)
    offsets = [0.0]
    for i in range(1, k_corridors):
        mag = ((i + 1) // 2) * base * width_scale
        offsets.append(mag if i % 2 else -mag)
    widths = [base * width_scale * (1.0 + 0.35 * (i // 2)) for i in range(k_corridors)]
    corridors: List[Corridor] = []
    for i, (off, width) in enumerate(zip(offsets, widths)):
        nodes: List[Node] = []
        for n, p in G.pos.items():
            d, t = _offset_distance(p, a, b, off)
            # 少し外側も許容し、source/target近傍の候補を落としにくくする。
            if d <= width and -0.08 <= t <= 1.08:
                nodes.append(n)
        if source not in nodes:
            nodes.append(source)
        if target not in nodes:
            nodes.append(target)
        corridors.append(Corridor(f"corridor_{i}", width, off, nodes))
    return corridors


def _search_corridors_sequential(G: Graph, source: Node, target: Node, corridors: list[Corridor]) -> list[tuple[Corridor, PathResult]]:
    return [
        (
            c,
            dijkstra(G, source, target, allowed_nodes=set(c.nodes), solver_name=f"mprc:{c.corridor_id}"),
        )
        for c in corridors
    ]


def _search_corridors_processes(G: Graph, source: Node, target: Node, corridors: list[Corridor], workers: int) -> list[tuple[Corridor, PathResult]]:
    max_workers = max(1, min(int(workers), len(corridors), os.cpu_count() or 1))
    if max_workers <= 1:
        return _search_corridors_sequential(G, source, target, corridors)

    results: list[tuple[Corridor, PathResult]] = []
    with ProcessPoolExecutor(max_workers=max_workers, initializer=_init_corridor_worker, initargs=(G, source, target)) as ex:
        futures = [ex.submit(_search_corridor_worker, c) for c in corridors]
        for fut in as_completed(futures):
            results.append(fut.result())
    # as_completedは順序を保存しないため、telemetryの読みやすさのために戻す。
    order = {c.corridor_id: i for i, c in enumerate(corridors)}
    results.sort(key=lambda cr: order[cr[0].corridor_id])
    return results


def mprc(
    G: Graph,
    source: Node,
    target: Node,
    *,
    k_corridors: int = 7,
    width_scale: float = 0.12,
    rescue: bool = True,
    repair_ratio_threshold: float = 1.02,
    workers: int = 1,
    parallel_backend: str = "process",
    min_parallel_nodes: int = 3000,
    min_parallel_corridor_nodes: int = 1500,
) -> PathResult:
    """Reference MPRC implementation for algorithmic evaluation.

    `workers > 1` enables process-level parallel corridor search.  The metric
    `parallel_steps` still represents logical synchronization rounds, while
    `time_ms` is actual wall-clock time including process startup/IPC overhead.
    """
    start = time.perf_counter()
    tracemalloc.start()
    corridors = generate_corridors(G, source, target, k_corridors, width_scale)

    worker_count = max(1, int(workers or 1))
    backend_used = "sequential"
    mean_corridor_nodes = sum(len(c.nodes) for c in corridors) / max(1, len(corridors))
    process_profitable = (
        worker_count > 1
        and parallel_backend == "process"
        and len(corridors) > 1
        and len(G.adj) >= int(min_parallel_nodes)
        and mean_corridor_nodes >= int(min_parallel_corridor_nodes)
    )
    try:
        if process_profitable:
            corridor_results = _search_corridors_processes(G, source, target, corridors, worker_count)
            backend_used = "process"
        else:
            corridor_results = _search_corridors_sequential(G, source, target, corridors)
            if worker_count > 1 and parallel_backend == "process":
                backend_used = "sequential_auto_gate"
    except Exception as exc:
        # Multiprocessing can fail for unpickleable custom node objects.  In that
        # case the reference implementation remains usable and records fallback.
        corridor_results = _search_corridors_sequential(G, source, target, corridors)
        backend_used = f"sequential_after_process_error:{type(exc).__name__}"

    candidates: List[Tuple[Corridor, PathResult]] = []
    work_relax = expanded = pushes = pops = 0
    for c, r in corridor_results:
        work_relax += r.work_relaxations; expanded += r.work_expanded_nodes
        pushes += r.queue_pushes; pops += r.queue_pops
        if r.found:
            candidates.append((c, r))

    best_c = None
    best_r = None
    if candidates:
        best_c, best_r = min(candidates, key=lambda x: x[1].distance)

    rescue_triggered = False
    repair_triggered = False
    repair_success = None
    error_code = None
    if rescue and best_r is None:
        rescue_triggered = True
        repair_triggered = True
        exact = dijkstra(G, source, target, solver_name="mprc_rescue_dijkstra")
        work_relax += exact.work_relaxations; expanded += exact.work_expanded_nodes
        pushes += exact.queue_pushes; pops += exact.queue_pops
        best_r = exact; best_c = Corridor("rescue_exact", math.inf, 0.0, G.nodes())
        repair_success = exact.found
        if not exact.found:
            error_code = "NO_CANDIDATE"

    found = best_r is not None and best_r.found
    path = best_r.path if found else []
    distance = best_r.distance if found else math.inf
    _, peak = tracemalloc.get_traced_memory(); tracemalloc.stop()
    logical_steps = 5 + (2 if rescue_triggered else 0)
    telemetry = {
        "k_corridors": len(corridors),
        "corridor_widths": [c.width for c in corridors],
        "corridor_offsets": [c.offset for c in corridors],
        "corridor_node_counts": [len(c.nodes) for c in corridors],
        "candidate_count": len(candidates),
        "valid_candidate_count": len(candidates),
        "best_corridor_id": best_c.corridor_id if best_c else "",
        "best_corridor_width": best_c.width if best_c else math.inf,
        "candidate_distances": [r.distance for _, r in candidates],
        "candidate_scores": [r.distance for _, r in candidates],
        "candidate_risks": [],
        "risk_score": 0.0 if candidates else 1.0,
        "rescue_triggered": rescue_triggered,
        "rescue_reason": "NO_CANDIDATE" if rescue_triggered else None,
        "repair_triggered": repair_triggered,
        "repair_success": repair_success,
        "post_repair_distance": distance if repair_triggered and found else None,
        "post_repair_ratio": None,
        "telemetry_completeness": 1.0,
        "work_candidate_expansions": 0,
        "work_repair": 0,
        "workers_requested": worker_count,
        "workers_used": min(worker_count, len(corridors), os.cpu_count() or 1) if backend_used == "process" else 1,
        "parallel_backend": backend_used,
        "mean_corridor_nodes": mean_corridor_nodes,
        "process_profitable": process_profitable,
        "error_code": error_code,
    }
    return PathResult(path, distance, found, False, "mprc", work_relax, expanded, pushes, pops, logical_steps, (time.perf_counter()-start)*1000, peak/1024, telemetry)
