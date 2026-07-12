from __future__ import annotations

import heapq
import math
import time
import tracemalloc
from typing import Dict, List, Optional, Set, Tuple

from ..graph import Graph
from ..types import Node, PathResult


def _memory_begin() -> bool:
    """Start tracemalloc only if this solver owns the measurement."""
    already = tracemalloc.is_tracing()
    if not already:
        tracemalloc.start()
    return already


def _memory_end(already_tracing: bool) -> float:
    if not tracemalloc.is_tracing():
        return 0.0
    _, peak = tracemalloc.get_traced_memory()
    if not already_tracing:
        tracemalloc.stop()
    return peak / 1024


def _reconstruct(prev: Dict[Node, Node], source: Node, target: Node) -> List[Node]:
    if source == target:
        return [source]
    if target not in prev:
        return []
    out = [target]
    cur = target
    while cur != source:
        cur = prev[cur]
        out.append(cur)
    out.reverse()
    return out


def dijkstra(G: Graph, source: Node, target: Node, allowed_nodes: Optional[Set[Node]] = None, solver_name: str = "dijkstra") -> PathResult:
    start = time.perf_counter()
    outer_memory = _memory_begin()
    if source not in G.adj or target not in G.adj:
        peak_kib = _memory_end(outer_memory)
        return PathResult([], math.inf, False, True, solver_name, time_ms=(time.perf_counter()-start)*1000, peak_memory_kib=peak_kib)
    if allowed_nodes is not None and (source not in allowed_nodes or target not in allowed_nodes):
        peak_kib = _memory_end(outer_memory)
        return PathResult([], math.inf, False, True, solver_name, time_ms=(time.perf_counter()-start)*1000, peak_memory_kib=peak_kib)

    dist: Dict[Node, float] = {source: 0.0}
    prev: Dict[Node, Node] = {}
    pq: List[Tuple[float, int, Node]] = [(0.0, 0, source)]
    counter = 1
    expanded = relax = pops = pushes = 0
    settled: Set[Node] = set()
    while pq:
        du, _, u = heapq.heappop(pq); pops += 1
        if u in settled:
            continue
        settled.add(u); expanded += 1
        if u == target:
            break
        for v, w in G.adj.get(u, []):
            if allowed_nodes is not None and v not in allowed_nodes:
                continue
            relax += 1
            nd = du + w
            if nd < dist.get(v, math.inf):
                dist[v] = nd; prev[v] = u
                heapq.heappush(pq, (nd, counter, v)); counter += 1; pushes += 1
    path = _reconstruct(prev, source, target)
    found = bool(path)
    peak_kib = _memory_end(outer_memory)
    return PathResult(path, dist.get(target, math.inf), found, True, solver_name, relax, expanded, pushes, pops, expanded, (time.perf_counter()-start)*1000, peak_kib)


def bidirectional_dijkstra(G: Graph, source: Node, target: Node) -> PathResult:
    start = time.perf_counter()
    outer_memory = _memory_begin()
    if source == target:
        peak_kib = _memory_end(outer_memory)
        return PathResult([source], 0.0, True, True, "bidirectional_dijkstra", time_ms=(time.perf_counter()-start)*1000, peak_memory_kib=peak_kib)
    if source not in G.adj or target not in G.adj:
        peak_kib = _memory_end(outer_memory)
        return PathResult([], math.inf, False, True, "bidirectional_dijkstra", time_ms=(time.perf_counter()-start)*1000, peak_memory_kib=peak_kib)

    R = G.reversed()
    dist_f: Dict[Node, float] = {source: 0.0}; dist_b: Dict[Node, float] = {target: 0.0}
    prev_f: Dict[Node, Node] = {}; prev_b: Dict[Node, Node] = {}
    pq_f: List[Tuple[float, int, Node]] = [(0.0, 0, source)]
    pq_b: List[Tuple[float, int, Node]] = [(0.0, 0, target)]
    seen_f: Set[Node] = set(); seen_b: Set[Node] = set()
    best = math.inf; meet: Node | None = None
    c = 1; expanded = relax = pushes = pops = steps = 0

    while pq_f and pq_b:
        if pq_f[0][0] + pq_b[0][0] >= best:
            break
        forward = pq_f[0][0] <= pq_b[0][0]
        pq = pq_f if forward else pq_b
        adj = G.adj if forward else R.adj
        dist = dist_f if forward else dist_b
        other = dist_b if forward else dist_f
        seen = seen_f if forward else seen_b
        prev = prev_f if forward else prev_b
        du, _, u = heapq.heappop(pq); pops += 1; steps += 1
        if u in seen:
            continue
        seen.add(u); expanded += 1
        if u in other and du + other[u] < best:
            best = du + other[u]; meet = u
        for v, w in adj.get(u, []):
            relax += 1
            nd = du + w
            if nd < dist.get(v, math.inf):
                dist[v] = nd; prev[v] = u
                heapq.heappush(pq, (nd, c, v)); c += 1; pushes += 1
            if v in other and nd + other[v] < best:
                best = nd + other[v]; meet = v

    path: List[Node] = []
    if meet is not None and math.isfinite(best):
        left = _reconstruct(prev_f, source, meet)
        right = []
        cur = meet
        while cur != target:
            cur = prev_b[cur]
            right.append(cur)
        path = left + right
    peak_kib = _memory_end(outer_memory)
    return PathResult(path, best, bool(path), True, "bidirectional_dijkstra", relax, expanded, pushes, pops, steps, (time.perf_counter()-start)*1000, peak_kib)
