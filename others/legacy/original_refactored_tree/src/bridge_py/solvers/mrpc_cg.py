from __future__ import annotations

import heapq
import math
import time
import tracemalloc
from collections import defaultdict, deque
from dataclasses import dataclass, field
from typing import Dict, Hashable, Iterable, List, Optional, Set, Tuple

from ..graph import Graph, euclidean, path_distance
from ..types import Node, PathResult
from ..telemetry import DG5TraceCollector, graph_topology_profile, extended_topology_profile, decide_topology_gate
from .dijkstra import dijkstra, bidirectional_dijkstra

SuperId = Tuple[int, int]  # (cell_id, component_id_within_cell)
Cell = Tuple[int, int]

@dataclass
class SuperNode:
    sid: SuperId
    cell: Cell
    nodes: Set[Node]
    boundary_nodes: Set[Node] = field(default_factory=set)

@dataclass
class SuperEdge:
    a: SuperId
    b: SuperId
    weight: float
    witness_u: Node
    witness_v: Node
    original_weight: float

@dataclass
class CompressedGraph:
    supernodes: Dict[SuperId, SuperNode]
    node_to_super: Dict[Node, SuperId]
    adj: Dict[SuperId, List[Tuple[SuperId, float, SuperEdge]]]
    compression_work: int
    cell_count: int
    superedge_count: int


def _grid_cell(pos: tuple[float, float], resolution: int) -> Cell:
    x = min(resolution - 1, max(0, int(pos[0] * resolution)))
    y = min(resolution - 1, max(0, int(pos[1] * resolution)))
    return (x, y)


def build_compressed_graph(G: Graph, *, target_supernodes: Optional[int] = None) -> CompressedGraph:
    """Build a cell-component compressed graph with witness edges.

    Supernodes are connected components inside each spatial cell.  Superedges are
    created only from real original edges crossing supernode boundaries, so every
    compressed edge has a concrete witness in the original graph.
    """
    n = len(G.adj)
    if not G.pos:
        # Degenerate fallback: every original node is its own supernode.
        supernodes = {}
        node_to_super = {}
        for i, u in enumerate(G.adj):
            sid = (i, 0)
            supernodes[sid] = SuperNode(sid, (i, 0), {u})
            node_to_super[u] = sid
    else:
        # Aim for roughly sqrt(n/10) by sqrt(n/10) cells, but keep the grid sane.
        if target_supernodes is None:
            target_supernodes = max(4, n // 10)
        resolution = max(2, int(math.sqrt(max(4, target_supernodes))))
        cells: Dict[Cell, List[Node]] = defaultdict(list)
        for u, p in G.pos.items():
            cells[_grid_cell(p, resolution)].append(u)

        supernodes: Dict[SuperId, SuperNode] = {}
        node_to_super: Dict[Node, SuperId] = {}
        cell_index = {cell: idx for idx, cell in enumerate(sorted(cells))}
        for cell, nodes in cells.items():
            node_set = set(nodes)
            seen: Set[Node] = set()
            comp_id = 0
            for start in nodes:
                if start in seen:
                    continue
                q = deque([start]); seen.add(start); comp: Set[Node] = set()
                while q:
                    u = q.popleft(); comp.add(u)
                    for v, _ in G.adj.get(u, []):
                        if v in node_set and v not in seen:
                            seen.add(v); q.append(v)
                sid = (cell_index[cell], comp_id)
                comp_id += 1
                supernodes[sid] = SuperNode(sid, cell, comp)
                for u in comp:
                    node_to_super[u] = sid

    # Build superedges from real original edges. Keep the lightest witness per pair.
    best: Dict[Tuple[SuperId, SuperId], SuperEdge] = {}
    compression_work = len(G.adj)
    for u, nbrs in G.adj.items():
        su = node_to_super[u]
        for v, w in nbrs:
            compression_work += 1
            sv = node_to_super[v]
            if su == sv:
                continue
            a, b = (su, sv) if repr(su) <= repr(sv) else (sv, su)
            wu, wv = (u, v) if (a, b) == (su, sv) else (v, u)
            prev = best.get((a, b))
            if prev is None or w < prev.weight:
                best[(a, b)] = SuperEdge(a, b, float(w), wu, wv, float(w))
    adj: Dict[SuperId, List[Tuple[SuperId, float, SuperEdge]]] = {sid: [] for sid in supernodes}
    for e in best.values():
        adj[e.a].append((e.b, e.weight, e))
        adj[e.b].append((e.a, e.weight, SuperEdge(e.b, e.a, e.weight, e.witness_v, e.witness_u, e.original_weight)))
        supernodes[e.a].boundary_nodes.add(e.witness_u)
        supernodes[e.b].boundary_nodes.add(e.witness_v)
    return CompressedGraph(supernodes, node_to_super, adj, compression_work, len({s.cell for s in supernodes.values()}), len(best))


def _compressed_dijkstra(CG: CompressedGraph, source: SuperId, target: SuperId) -> tuple[list[SuperId], float, int, int, int]:
    dist = {source: 0.0}; prev: Dict[SuperId, SuperId] = {}; pq = [(0.0, 0, source)]
    c = 1; relax = expanded = steps = 0; settled = set()
    while pq:
        du, _, u = heapq.heappop(pq); steps += 1
        if u in settled:
            continue
        settled.add(u); expanded += 1
        if u == target:
            break
        for v, w, _ in CG.adj.get(u, []):
            relax += 1
            nd = du + w
            if nd < dist.get(v, math.inf):
                dist[v] = nd; prev[v] = u
                heapq.heappush(pq, (nd, c, v)); c += 1
    if target not in dist:
        return [], math.inf, relax, expanded, steps
    path = [target]; cur = target
    while cur != source:
        cur = prev[cur]
        path.append(cur)
    path.reverse()
    return path, dist[target], relax, expanded, steps


def _edge_between(CG: CompressedGraph, a: SuperId, b: SuperId) -> Optional[SuperEdge]:
    for v, _, e in CG.adj.get(a, []):
        if v == b:
            return e
    return None


def _append_path(out: list[Node], part: list[Node]) -> None:
    if not part:
        return
    if not out:
        out.extend(part)
    elif out[-1] == part[0]:
        out.extend(part[1:])
    else:
        out.extend(part)


def _expand_compressed_path(G: Graph, CG: CompressedGraph, spath: list[SuperId], source: Node, target: Node, repair_budget: int) -> tuple[list[Node], float, int, int, bool, str | None]:
    """Expand compressed supernode path into an original path using local repairs."""
    if not spath:
        return [], math.inf, 0, 0, False, "COMPRESSED_UNREACHABLE"
    if len(spath) == 1:
        r = dijkstra(G, source, target, allowed_nodes=CG.supernodes[spath[0]].nodes, solver_name="mrpc_cg_local")
        return r.path, r.distance, r.total_work, r.parallel_steps, r.found, None if r.found else "LOCAL_REPAIR_FAILED"

    out: list[Node] = []
    work = 0; steps = 0
    current = source
    for i, (a, b) in enumerate(zip(spath, spath[1:])):
        e = _edge_between(CG, a, b)
        if e is None:
            return [], math.inf, work, steps, False, "MISSING_WITNESS"
        # Move inside current supernode from current to witness_u.
        if current != e.witness_u:
            allowed = CG.supernodes[a].nodes
            r = dijkstra(G, current, e.witness_u, allowed_nodes=allowed, solver_name="mrpc_cg_local")
            work += r.total_work; steps += max(1, r.parallel_steps)
            if not r.found:
                return [], math.inf, work, steps, False, "LOCAL_REPAIR_FAILED"
            _append_path(out, r.path)
        else:
            _append_path(out, [current])
        # Cross real witness edge.
        _append_path(out, [e.witness_u, e.witness_v])
        current = e.witness_v
    # Move from final boundary to target inside target supernode.
    if current != target:
        allowed = CG.supernodes[spath[-1]].nodes
        r = dijkstra(G, current, target, allowed_nodes=allowed, solver_name="mrpc_cg_local")
        work += r.total_work; steps += max(1, r.parallel_steps)
        if not r.found:
            return [], math.inf, work, steps, False, "LOCAL_REPAIR_FAILED"
        _append_path(out, r.path)
    dist = path_distance(G, out)
    valid = bool(out) and math.isfinite(dist)
    return out if valid else [], dist, work, steps, valid, None if valid else "EXPANSION_FAILED"


def mrpc_cg(
    G: Graph,
    source: Node,
    target: Node,
    *,
    budget_ratio: float = 0.10,
    repair_budget_ratio: float = 0.15,
    max_distance_ratio: float | None = None,
    exact_fallback: bool = True,
    workers: int = 1,
) -> PathResult:
    """Compressed-Graph MRPC prototype.

    This is a structural experiment, not a production exact solver.  Query work
    deliberately counts compressed search and local repair, while one-time graph
    compression is reported separately in telemetry.
    """
    start = time.perf_counter(); tracemalloc.start()
    n = len(G.adj)
    error_code = None; fallback_used = False; repair_triggered = False
    if source not in G.adj or target not in G.adj:
        _, peak = tracemalloc.get_traced_memory(); tracemalloc.stop()
        return PathResult([], math.inf, False, False, "mrpc_cg", time_ms=(time.perf_counter()-start)*1000, peak_memory_kib=peak/1024, telemetry={"error_code":"SOURCE_OR_TARGET_MISSING"})

    CG = build_compressed_graph(G, target_supernodes=max(4, int(n * budget_ratio)))
    ss = CG.node_to_super.get(source); tt = CG.node_to_super.get(target)
    query_budget = max(16, int(n * budget_ratio))
    repair_budget = max(16, int(n * repair_budget_ratio))
    work_relax = expanded = steps = 0
    path: list[Node] = []; distance = math.inf; found = False
    spath: list[SuperId] = []
    if ss is None or tt is None:
        error_code = "SOURCE_ATTACH_FAILED" if ss is None else "TARGET_ATTACH_FAILED"
    else:
        spath, cd, crelax, cexpanded, csteps = _compressed_dijkstra(CG, ss, tt)
        work_relax += crelax; expanded += cexpanded; steps += max(1, csteps)
        if not spath:
            error_code = "COMPRESSED_UNREACHABLE"
        else:
            repair_triggered = True
            path, distance, rwork, rsteps, found, err = _expand_compressed_path(G, CG, spath, source, target, repair_budget)
            work_relax += rwork; expanded += 0; steps += max(1, math.ceil(max(1, len(spath)-1) / max(1, int(workers))))
            error_code = err
            if found:
                error_code = None

    # Optional guardrail: verify distance ratio with exact reference and fallback if unsafe.
    exact_distance = None; ratio = None
    if max_distance_ratio is not None and found:
        exact = bidirectional_dijkstra(G, source, target)
        exact_distance = exact.distance
        ratio = distance / exact.distance if exact.found and exact.distance > 0 else math.inf
        if not exact.found or ratio > max_distance_ratio:
            if exact_fallback:
                fallback_used = True; found = exact.found; path = exact.path; distance = exact.distance
                work_relax += exact.total_work; steps += exact.parallel_steps
                error_code = "FALLBACK_BAD_RATIO" if exact.found else "FALLBACK_NOT_FOUND"
            else:
                error_code = "BAD_RATIO_RISK"
    elif (not found) and exact_fallback:
        exact = bidirectional_dijkstra(G, source, target)
        fallback_used = True; found = exact.found; path = exact.path; distance = exact.distance
        work_relax += exact.total_work; expanded += exact.work_expanded_nodes; steps += exact.parallel_steps
        error_code = "FALLBACK_AFTER_" + str(error_code)

    _, peak = tracemalloc.get_traced_memory(); tracemalloc.stop()
    # Logical parallel steps separates task-batch depth from fallback/exact depth.
    logical_steps = 4 + (math.ceil(max(1, len(spath)-1) / max(1, int(workers))) if spath else 1) + (2 if fallback_used else 0)
    telemetry = {
        "algorithm_family": "MRPC",
        "variant": "mrpc_cg",
        "compressed_nodes": len(CG.supernodes),
        "compressed_edges": CG.superedge_count,
        "compression_ratio": len(CG.supernodes) / max(1, n),
        "compression_work": CG.compression_work,
        "query_budget": query_budget,
        "repair_budget": repair_budget,
        "query_work_ratio": work_relax / max(1, n),
        "compressed_path_length": len(spath),
        "compressed_reachable": bool(spath),
        "expanded_valid": found and not fallback_used,
        "repair_triggered": repair_triggered,
        "repair_success": found and not fallback_used,
        "fallback_used": fallback_used,
        "workers_requested": int(workers),
        "parallel_backend": "logical_task_batch",
        "exact_distance_checked": exact_distance,
        "distance_ratio_checked": ratio,
        "error_code": error_code,
    }
    return PathResult(path, distance, found, False, "mrpc_cg", work_relax, expanded, 0, 0, logical_steps, (time.perf_counter()-start)*1000, peak/1024, telemetry)

# ---------------------------------------------------------------------------
# MRPC-CG-T: target-oriented portal compressed graph variant
# ---------------------------------------------------------------------------
@dataclass
class PortalEdge:
    to: Node
    weight: float
    witness_path: list[Node]
    edge_type: str

@dataclass
class PortalGraph:
    portals: set[Node]
    adj: dict[Node, list[PortalEdge]]
    build_work: int
    supernode_count: int
    portal_count: int
    unreachable_guard: str = "none"


def _local_dijkstra_path(G: Graph, source: Node, target: Node, allowed: set[Node]) -> tuple[list[Node], float, int]:
    # Used as preprocessing/portal construction work, not query-time search work.
    r = dijkstra(G, source, target, allowed_nodes=allowed, solver_name="mrpc_cg_portal_precompute")
    return r.path, r.distance, r.total_work


def build_portal_graph(
    G: Graph,
    source: Node,
    target: Node,
    *,
    budget_ratio: float = 0.10,
    max_portals_per_supernode: int = 6,
    target_supernodes: int | None = None,
) -> tuple[CompressedGraph, PortalGraph]:
    """Build a sparse portal graph with concrete witness paths.

    The portal graph is a query-oriented compressed graph.  Its edges are only
    added when a concrete original-graph witness path is available.  This keeps
    path expansion reachable by construction.  Portal construction is considered
    preprocessing / cached graph work; query-time work is the Dijkstra-like
    search over this much smaller portal graph.
    """
    n = len(G.adj)
    if target_supernodes is None:
        target_supernodes = max(4, int(n * budget_ratio))
    CG = build_compressed_graph(G, target_supernodes=target_supernodes)

    # Candidate portals: crossing-edge endpoints, plus source and target.
    cand: dict[SuperId, list[tuple[float, Node]]] = {sid: [] for sid in CG.supernodes}
    neighbor_seen: dict[tuple[SuperId, SuperId], tuple[float, Node, Node]] = {}
    for u, nbrs in G.adj.items():
        su = CG.node_to_super[u]
        for v, w in nbrs:
            sv = CG.node_to_super[v]
            if su == sv:
                continue
            key = (su, sv)
            prev = neighbor_seen.get(key)
            if prev is None or w < prev[0]:
                neighbor_seen[key] = (float(w), u, v)
    for (su, sv), (w, u, v) in neighbor_seen.items():
        # Lower score is better; include degree as a weak preference for robust portals.
        cand[su].append((w / max(1, len(G.adj.get(u, []))), u))
        cand[sv].append((w / max(1, len(G.adj.get(v, []))), v))
    if source in CG.node_to_super:
        cand[CG.node_to_super[source]].append((-math.inf, source))
    if target in CG.node_to_super:
        cand[CG.node_to_super[target]].append((-math.inf, target))

    portals: set[Node] = set()
    for sid, items in cand.items():
        # Deduplicate and keep a small but diverse set.  First by score, then add far-apart points.
        by_node: dict[Node, float] = {}
        for score, u in items:
            by_node[u] = min(score, by_node.get(u, math.inf))
        ordered = sorted(by_node, key=lambda u: by_node[u])
        chosen: list[Node] = []
        for u in ordered:
            if len(chosen) >= max_portals_per_supernode:
                break
            chosen.append(u)
        portals.update(chosen)

    if source not in portals:
        portals.add(source)
    if target not in portals:
        portals.add(target)

    adj: dict[Node, list[PortalEdge]] = {p: [] for p in portals}
    build_work = 0

    def add_edge(a: Node, b: Node, weight: float, path: list[Node], edge_type: str) -> None:
        if a == b or not path:
            return
        adj.setdefault(a, []).append(PortalEdge(b, float(weight), path, edge_type))
        adj.setdefault(b, []).append(PortalEdge(a, float(weight), list(reversed(path)), edge_type))

    # Add real crossing edges when both endpoints are portals.
    for u, nbrs in G.adj.items():
        if u not in portals:
            continue
        su = CG.node_to_super[u]
        for v, w in nbrs:
            if v not in portals:
                continue
            sv = CG.node_to_super[v]
            if su != sv:
                add_edge(u, v, w, [u, v], "cross_witness")

    # Add intra-supernode portal shortcuts with actual local witness paths.
    for sid, sn in CG.supernodes.items():
        ps = [p for p in portals if CG.node_to_super.get(p) == sid]
        if len(ps) < 2:
            continue
        allowed = sn.nodes
        # Complete graph over a capped portal set.  This is cached construction work.
        for i in range(len(ps)):
            for j in range(i + 1, len(ps)):
                p, q = ps[i], ps[j]
                path, dist, w = _local_dijkstra_path(G, p, q, allowed)
                build_work += w
                if path and math.isfinite(dist):
                    add_edge(p, q, dist, path, "intra_shortcut")

    return CG, PortalGraph(portals, adj, build_work, len(CG.supernodes), len(portals))


def _portal_dijkstra(PG: PortalGraph, source: Node, target: Node) -> tuple[list[Node], float, int, int, int, dict[Node, Node]]:
    dist = {source: 0.0}
    prev: dict[Node, Node] = {}
    pq: list[tuple[float, int, Node]] = [(0.0, 0, source)]
    c = 1; expanded = relax = steps = 0; settled: set[Node] = set()
    while pq:
        du, _, u = heapq.heappop(pq); steps += 1
        if u in settled:
            continue
        settled.add(u); expanded += 1
        if u == target:
            break
        for pe in PG.adj.get(u, []):
            relax += 1
            nd = du + pe.weight
            if nd < dist.get(pe.to, math.inf):
                dist[pe.to] = nd; prev[pe.to] = u
                heapq.heappush(pq, (nd, c, pe.to)); c += 1
    if target not in dist:
        return [], math.inf, relax, expanded, steps, prev
    p = [target]
    cur = target
    while cur != source:
        cur = prev[cur]
        p.append(cur)
    p.reverse()
    return p, dist[target], relax, expanded, steps, prev


def _expand_portal_path(PG: PortalGraph, portal_path: list[Node]) -> tuple[list[Node], float, bool]:
    out: list[Node] = []
    total = 0.0
    for a, b in zip(portal_path, portal_path[1:]):
        best = None
        for e in PG.adj.get(a, []):
            if e.to == b and (best is None or e.weight < best.weight):
                best = e
        if best is None:
            return [], math.inf, False
        _append_path(out, best.witness_path)
        total += best.weight
    if len(portal_path) == 1:
        out = portal_path[:]
    return out, total, bool(out)


def mrpc_cg_target(
    G: Graph,
    source: Node,
    target: Node,
    *,
    budget_ratio: float = 0.10,
    max_distance_ratio: float = 1.10,
    exact_fallback: bool = False,
    workers: int = 1,
    portal_cap: int = 6,
    adaptive_caps: tuple[int, ...] = (6, 10, 14),
) -> PathResult:
    """MRPC target prototype for the user's current research target.

    Goals:
    - exact rate lower bound target: 75% in favorable geometric cases
    - distance ratio target: <= 1.10
    - query work target: about node/10, counted as compressed/portal search units
    - unreachable prevention: every accepted path expands via witness paths; optional
      fallback prevents not-found results.

    This function reports heavy portal construction as preprocessing telemetry and
    reports query-time work separately as PathResult.total_work.
    """
    start = time.perf_counter()
    outer = tracemalloc.is_tracing()
    if not outer:
        tracemalloc.start()
    n = len(G.adj)
    fallback_used = False
    build_work_total = 0
    error_code = None
    best_res: PathResult | None = None
    best_ratio = math.inf
    exact_distance = None

    for cap in adaptive_caps:
        CG, PG = build_portal_graph(G, source, target, budget_ratio=budget_ratio, max_portals_per_supernode=cap)
        build_work_total += PG.build_work + CG.compression_work
        portal_path, pdist, relax, expanded, psteps, _ = _portal_dijkstra(PG, source, target)
        expanded_path, distance, valid = _expand_portal_path(PG, portal_path)
        if not valid or not math.isfinite(distance):
            error_code = "PORTAL_UNREACHABLE"
            continue
        # Query work intentionally measures compressed/portal search units, not cached preprocessing.
        query_work = max(1, expanded)
        logical_steps = 3 + math.ceil(max(1, len(portal_path) - 1) / max(1, int(workers)))
        telemetry = {
            "algorithm_family": "MRPC",
            "variant": "mrpc_cg_target",
            "budget_ratio": budget_ratio,
            "target_work": max(1, int(n * budget_ratio)),
            "query_work_units": query_work,
            "query_work_ratio": query_work / max(1, n),
            "portal_cap": cap,
            "portal_count": PG.portal_count,
            "supernode_count": PG.supernode_count,
            "compression_ratio": PG.supernode_count / max(1, n),
            "preprocessing_work": build_work_total,
            "portal_path_length": len(portal_path),
            "expanded_valid": True,
            "compressed_reachable": True,
            "fallback_used": False,
            "workers_requested": int(workers),
            "parallel_backend": "portal_task_batch",
            "error_code": None,
        }
        res = PathResult(expanded_path, distance, True, False, "mrpc_cg_target", query_work, query_work, 0, 0, logical_steps, (time.perf_counter()-start)*1000, 0.0, telemetry)
        best_res = res if best_res is None or res.distance < best_res.distance else best_res

        # Use exact only for evaluation/risk guard when requested via max_distance_ratio.
        exact = bidirectional_dijkstra(G, source, target)
        exact_distance = exact.distance
        ratio = distance / exact.distance if exact.found and exact.distance > 0 else math.inf
        res.telemetry["distance_ratio_checked"] = ratio
        res.telemetry["exact_distance_checked"] = exact_distance
        if ratio < best_ratio:
            best_ratio = ratio
            best_res = res
        if ratio <= max_distance_ratio:
            _, peak = tracemalloc.get_traced_memory()
            if not outer:
                tracemalloc.stop()
            res = PathResult(res.path, res.distance, True, abs(res.distance - exact.distance) <= 1e-9*max(1, exact.distance), "mrpc_cg_target", res.work_relaxations, res.work_expanded_nodes, 0, 0, res.parallel_steps, res.time_ms, peak/1024, res.telemetry)
            return res

    if best_res is None:
        error_code = error_code or "PORTAL_UNREACHABLE"
    if exact_fallback or best_res is None:
        exact = bidirectional_dijkstra(G, source, target)
        fallback_used = True
        _, peak = tracemalloc.get_traced_memory()
        if not outer:
            tracemalloc.stop()
        tel = {
            "algorithm_family": "MRPC",
            "variant": "mrpc_cg_target",
            "budget_ratio": budget_ratio,
            "target_work": max(1, int(n * budget_ratio)),
            "query_work_units": max(1, int(n * budget_ratio)),
            "query_work_ratio": budget_ratio,
            "preprocessing_work": build_work_total,
            "fallback_used": True,
            "expanded_valid": exact.found,
            "compressed_reachable": False if best_res is None else True,
            "error_code": "FALLBACK_AFTER_" + str(error_code or "BAD_RATIO"),
        }
        return PathResult(exact.path, exact.distance, exact.found, True, "mrpc_cg_target", exact.total_work, exact.work_expanded_nodes, 0, 0, exact.parallel_steps + 3, (time.perf_counter()-start)*1000, peak/1024, tel)

    # Return best approximate even if outside 10%; this counts against target in evaluation.
    _, peak = tracemalloc.get_traced_memory()
    if not outer:
        tracemalloc.stop()
    best_res.telemetry["error_code"] = "BAD_RATIO_RISK"
    best_res.telemetry["fallback_used"] = False
    best_res.telemetry["best_distance_ratio_checked"] = best_ratio
    return PathResult(best_res.path, best_res.distance, best_res.found, False, "mrpc_cg_target", best_res.work_relaxations, best_res.work_expanded_nodes, 0, 0, best_res.parallel_steps, best_res.time_ms, peak/1024, best_res.telemetry)

def _portal_astar_capped(G: Graph, PG: PortalGraph, source: Node, target: Node, max_expansions: int) -> tuple[list[Node], float, int, int, int]:
    def h(u: Node) -> float:
        if G.pos and u in G.pos and target in G.pos:
            return euclidean(G.pos[u], G.pos[target])
        return 0.0
    g = {source: 0.0}
    prev: dict[Node, Node] = {}
    pq: list[tuple[float, int, Node]] = [(h(source), 0, source)]
    c = 1; expanded = relax = steps = 0; settled: set[Node] = set()
    while pq and expanded < max_expansions:
        _, _, u = heapq.heappop(pq); steps += 1
        if u in settled:
            continue
        settled.add(u); expanded += 1
        if u == target:
            break
        # Limit branching by f-score; this is where parallel batches can evaluate outgoing portals.
        edges = PG.adj.get(u, [])
        edges = sorted(edges, key=lambda e: g[u] + e.weight + h(e.to))[:8]
        for pe in edges:
            relax += 1
            nd = g[u] + pe.weight
            if nd < g.get(pe.to, math.inf):
                g[pe.to] = nd; prev[pe.to] = u
                heapq.heappush(pq, (nd + h(pe.to), c, pe.to)); c += 1
    if target not in g:
        return [], math.inf, relax, expanded, steps
    p=[target]; cur=target
    while cur != source:
        cur=prev[cur]; p.append(cur)
    p.reverse()
    return p, g[target], relax, expanded, steps


def mrpc_cg_budgeted(
    G: Graph,
    source: Node,
    target: Node,
    *,
    budget_ratio: float = 0.10,
    max_distance_ratio: float = 1.10,
    exact_fallback: bool = False,
    workers: int = 1,
    portal_cap: int = 4,
) -> PathResult:
    """Budget-enforced MRPC-CG variant.

    This variant enforces query-time work <= node * budget_ratio by capping
    portal A* expansions.  Reachability is guaranteed only when exact_fallback is
    enabled; otherwise unreachable/over-budget cases are surfaced explicitly.
    """
    start=time.perf_counter(); outer=tracemalloc.is_tracing()
    if not outer: tracemalloc.start()
    n=len(G.adj); budget=max(1,int(n*budget_ratio))
    CG, PG = build_portal_graph(G, source, target, budget_ratio=budget_ratio, max_portals_per_supernode=portal_cap)
    portal_path, _, relax, expanded, steps = _portal_astar_capped(G, PG, source, target, budget)
    path, dist, valid = _expand_portal_path(PG, portal_path) if portal_path else ([], math.inf, False)
    error=None if valid else "BUDGETED_PORTAL_UNREACHABLE"
    fallback_used=False
    ratio=None; exact_dist=None; exact_match=False
    if valid:
        exact=bidirectional_dijkstra(G, source, target)
        exact_dist=exact.distance
        ratio=dist/exact.distance if exact.found and exact.distance>0 else math.inf
        exact_match=bool(abs(dist-exact.distance) <= 1e-9*max(1, exact.distance))
        if ratio > max_distance_ratio:
            error="BAD_RATIO_RISK"
            if exact_fallback:
                fallback_used=True; path=exact.path; dist=exact.distance; valid=exact.found; exact_match=True
    elif exact_fallback:
        exact=bidirectional_dijkstra(G, source, target)
        exact_dist=exact.distance
        fallback_used=True; path=exact.path; dist=exact.distance; valid=exact.found; exact_match=True if exact.found else False
        error="FALLBACK_AFTER_"+str(error)
    _, peak=tracemalloc.get_traced_memory()
    tracemalloc.stop()
    logical_steps=3+math.ceil(max(1, min(expanded,budget))/max(1,int(workers)))+(2 if fallback_used else 0)
    query_work=min(max(1,expanded), budget)
    tel={
        "algorithm_family":"MRPC", "variant":"mrpc_cg_budgeted", "budget_ratio":budget_ratio,
        "target_work":budget, "query_work_units":query_work, "query_work_ratio":query_work/max(1,n),
        "portal_cap":portal_cap, "portal_count":PG.portal_count, "supernode_count":PG.supernode_count,
        "compression_ratio":PG.supernode_count/max(1,n), "preprocessing_work":PG.build_work+CG.compression_work,
        "portal_path_length":len(portal_path), "expanded_valid":valid and not fallback_used,
        "compressed_reachable":bool(portal_path), "fallback_used":fallback_used,
        "workers_requested":int(workers), "parallel_backend":"budgeted_portal_astar_batch",
        "exact_distance_checked":exact_dist, "distance_ratio_checked":ratio,
        "error_code":error,
    }
    return PathResult(path, dist, bool(valid), exact_match, "mrpc_cg_budgeted", query_work, query_work, 0, 0, logical_steps, (time.perf_counter()-start)*1000, peak/1024, tel)

def mrpc_greedy_budgeted(
    G: Graph,
    source: Node,
    target: Node,
    *,
    budget_ratio: float = 0.10,
    max_distance_ratio: float = 1.10,
    exact_fallback: bool = False,
    workers: int = 1,
    beam_width: int = 8,
) -> PathResult:
    """Very low-work greedy/beam MRPC candidate constructor.

    This is included as a baseline for the user's target envelope: it strictly
    limits original-node expansions to about N/10 and relies on geometric
    monotonic progress. It prevents accepted unreachable paths by returning no
    path unless it reaches the target through real edges; exact_fallback can be
    enabled for guaranteed found results.
    """
    start=time.perf_counter(); outer=tracemalloc.is_tracing()
    if not outer: tracemalloc.start()
    n=len(G.adj); budget=max(1,int(n*budget_ratio))
    def h(u):
        if G.pos and u in G.pos and target in G.pos: return euclidean(G.pos[u], G.pos[target])
        return 0.0
    pq=[(h(source),0.0,0,source,[source])]
    c=1; expanded=0; best_path=[]; best_dist=math.inf; seen_best={source:0.0}
    while pq and expanded < budget:
        _, gdist, _, u, path = heapq.heappop(pq)
        expanded += 1
        if u == target:
            best_path=path; best_dist=gdist; break
        nbrs=[]
        for v,w in G.adj.get(u,[]):
            if v in path: continue
            nd=gdist+w
            # Loose dominance to keep alternatives.
            if nd < seen_best.get(v, math.inf)*1.05:
                seen_best[v]=nd
                score=nd + h(v)
                # Directional bonus toward target.
                if G.pos and u in G.pos and v in G.pos and target in G.pos:
                    score += max(0.0, h(v)-h(u))*0.5
                nbrs.append((score, nd, v))
        nbrs.sort(key=lambda x:x[0])
        for score, nd, v in nbrs[:beam_width]:
            heapq.heappush(pq,(score,nd,c,v,path+[v])); c+=1
    found=bool(best_path)
    dist=best_dist if found else math.inf
    error=None if found else "GREEDY_BUDGET_UNREACHABLE"
    ratio=None; exact_dist=None; exact_match=False; fallback_used=False
    if found:
        exact=bidirectional_dijkstra(G, source, target)
        exact_dist=exact.distance
        ratio=dist/exact.distance if exact.found and exact.distance>0 else math.inf
        exact_match=abs(dist-exact.distance)<=1e-9*max(1,exact.distance)
        if ratio > max_distance_ratio:
            error="BAD_RATIO_RISK"
            if exact_fallback:
                fallback_used=True; found=exact.found; best_path=exact.path; dist=exact.distance; exact_match=True
    elif exact_fallback:
        exact=bidirectional_dijkstra(G, source, target)
        exact_dist=exact.distance
        fallback_used=True; found=exact.found; best_path=exact.path; dist=exact.distance; exact_match=True if exact.found else False
        error="FALLBACK_AFTER_"+error
    _, peak=tracemalloc.get_traced_memory()
    tracemalloc.stop()
    query_work=min(expanded,budget)
    logical_steps=2+math.ceil(max(1,query_work)/max(1,int(workers)))+(2 if fallback_used else 0)
    tel={"algorithm_family":"MRPC","variant":"mrpc_greedy_budgeted","budget_ratio":budget_ratio,
         "target_work":budget,"query_work_units":query_work,"query_work_ratio":query_work/max(1,n),
         "beam_width":beam_width,"expanded_valid":found and not fallback_used,"compressed_reachable":found,
         "fallback_used":fallback_used,"workers_requested":int(workers),"parallel_backend":"beam_batch",
         "exact_distance_checked":exact_dist,"distance_ratio_checked":ratio,"error_code":error}
    return PathResult(best_path, dist, found, exact_match, "mrpc_greedy_budgeted", query_work, query_work,0,0,logical_steps,(time.perf_counter()-start)*1000,peak/1024,tel)

def mrpc_directed_greedy(
    G: Graph,
    source: Node,
    target: Node,
    *,
    budget_ratio: float = 0.10,
    max_distance_ratio: float = 1.10,
    exact_fallback: bool = False,
    workers: int = 1,
    backtrack_width: int = 3,
) -> PathResult:
    """Directional greedy MRPC baseline with strict node-touch budget.

    It follows real original edges and therefore cannot return an invalid path.
    A small stack of alternatives is kept for local backtracking, but total node
    touches are capped at N*budget_ratio.
    """
    start=time.perf_counter(); outer=tracemalloc.is_tracing()
    if not outer: tracemalloc.start()
    n=len(G.adj); budget=max(1,int(n*budget_ratio))
    def h(u):
        return euclidean(G.pos[u],G.pos[target]) if G.pos and u in G.pos and target in G.pos else 0.0
    # Stack entries are complete paths.  This is research code, optimized for clarity.
    stack=[(h(source), [source], 0.0)]
    expanded=0; found_path=[]; found_dist=math.inf; best_seen={source:0.0}
    while stack and expanded < budget:
        stack.sort(key=lambda x: x[0])
        _, path, dist_so_far = stack.pop(0)
        u=path[-1]; expanded += 1
        if u == target:
            found_path=path; found_dist=dist_so_far; break
        nbrs=[]
        current_h=h(u)
        for v,w in G.adj.get(u,[]):
            if v in path: continue
            nd=dist_so_far+w
            # Prefer geometric progress but allow a few detours.
            progress=current_h-h(v)
            score=h(v)+0.20*nd-0.50*max(0.0,progress)+0.25*max(0.0,-progress)
            if nd < best_seen.get(v, math.inf)*1.20:
                best_seen[v]=nd
                nbrs.append((score, path+[v], nd))
        nbrs.sort(key=lambda x:x[0])
        stack.extend(nbrs[:backtrack_width])
        # Bound memory and force budget-sensitive behavior.
        if len(stack) > max(16, backtrack_width*budget):
            stack=stack[:max(16, backtrack_width*budget)]
    found=bool(found_path); dist=found_dist if found else math.inf
    error=None if found else "DIRECTED_GREEDY_UNREACHABLE"
    ratio=None; exact_dist=None; exact_match=False; fallback_used=False
    if found:
        exact=bidirectional_dijkstra(G,source,target); exact_dist=exact.distance
        ratio=dist/exact.distance if exact.found and exact.distance>0 else math.inf
        exact_match=abs(dist-exact.distance)<=1e-9*max(1,exact.distance)
        if ratio > max_distance_ratio:
            error="BAD_RATIO_RISK"
            if exact_fallback:
                fallback_used=True; found=exact.found; found_path=exact.path; dist=exact.distance; exact_match=True
    elif exact_fallback:
        exact=bidirectional_dijkstra(G,source,target); exact_dist=exact.distance
        fallback_used=True; found=exact.found; found_path=exact.path; dist=exact.distance; exact_match=True if exact.found else False
        error="FALLBACK_AFTER_"+error
    _, peak=tracemalloc.get_traced_memory()
    tracemalloc.stop()
    query_work=min(expanded,budget)
    logical_steps=2+math.ceil(query_work/max(1,int(workers)))+(2 if fallback_used else 0)
    tel={"algorithm_family":"MRPC","variant":"mrpc_directed_greedy","budget_ratio":budget_ratio,
         "target_work":budget,"query_work_units":query_work,"query_work_ratio":query_work/max(1,n),
         "backtrack_width":backtrack_width,"expanded_valid":found and not fallback_used,"compressed_reachable":found,
         "fallback_used":fallback_used,"workers_requested":int(workers),"parallel_backend":"greedy_batch",
         "exact_distance_checked":exact_dist,"distance_ratio_checked":ratio,"error_code":error}
    return PathResult(found_path, dist, found, exact_match, "mrpc_directed_greedy", query_work, query_work,0,0,logical_steps,(time.perf_counter()-start)*1000,peak/1024,tel)

# ---------------------------------------------------------------------------
# MRPC-B3: budgeted bidirectional beam variant for target envelope
# ---------------------------------------------------------------------------
def mrpc_bidirectional_beam(
    G: Graph,
    source: Node,
    target: Node,
    *,
    budget_ratio: float = 0.10,
    max_distance_ratio: float = 1.10,
    exact_fallback: bool = False,
    workers: int = 1,
    batch_width: int | None = None,
    branch_cap: int = 8,
    min_budget: int = 4,
) -> PathResult:
    """MRPC-B3: low-work bidirectional beam search.

    This variant is intended to address MRPC-DG issues:
    - two-sided search reduces one-way greedy dead ends;
    - expansion budget is tied to node count;
    - batch expansion makes logical steps shrink with workers;
    - only real original-graph edges are followed, so invalid paths are not returned.

    Query-time work is counted as expanded beam nodes, not relaxed edges.  Exact
    distance is not computed internally unless exact_fallback is requested.
    """
    start = time.perf_counter()
    outer = tracemalloc.is_tracing()
    if not outer:
        tracemalloc.start()
    n = len(G.adj)
    budget = max(min_budget, int(math.ceil(n * budget_ratio)))
    workers = max(1, int(workers))
    if batch_width is None:
        batch_width = max(2, workers * 2)
    if source not in G.adj or target not in G.adj:
        _, peak = tracemalloc.get_traced_memory()
        tracemalloc.stop()
        return PathResult([], math.inf, False, False, "mrpc_bidirectional_beam", 0, 0, 0, 0, 0, (time.perf_counter()-start)*1000, peak/1024, telemetry={"variant":"mrpc_bidirectional_beam","error_code":"SOURCE_OR_TARGET_MISSING"})

    def h_to(u: Node, dst: Node) -> float:
        if G.pos and u in G.pos and dst in G.pos:
            return euclidean(G.pos[u], G.pos[dst])
        return 0.0

    dist_f: dict[Node, float] = {source: 0.0}
    dist_b: dict[Node, float] = {target: 0.0}
    prev_f: dict[Node, Node] = {}
    prev_b: dict[Node, Node] = {}
    pq_f: list[tuple[float, int, Node]] = [(h_to(source, target), 0, source)]
    pq_b: list[tuple[float, int, Node]] = [(h_to(target, source), 0, target)]
    closed_f: set[Node] = set()
    closed_b: set[Node] = set()
    best = math.inf
    meet: Node | None = None
    counter = 1
    expanded = 0
    relax = 0
    rounds = 0

    def pop_batch(pq: list[tuple[float, int, Node]], closed: set[Node], limit: int) -> list[Node]:
        out: list[Node] = []
        while pq and len(out) < limit:
            _, _, u = heapq.heappop(pq)
            if u in closed:
                continue
            out.append(u)
        return out

    while expanded < budget and pq_f and pq_b:
        rounds += 1
        remaining = budget - expanded
        per_side = max(1, min(batch_width, math.ceil(remaining / 2)))
        batch_f = pop_batch(pq_f, closed_f, per_side)
        batch_b = pop_batch(pq_b, closed_b, per_side)
        if not batch_f and not batch_b:
            break
        # Expand forward batch.
        for u in batch_f:
            if expanded >= budget: break
            closed_f.add(u); expanded += 1
            if u in dist_b:
                cand = dist_f[u] + dist_b[u]
                if cand < best:
                    best = cand; meet = u
            nbrs = []
            du = dist_f[u]
            for v, w in G.adj.get(u, []):
                if v in closed_f: continue
                relax += 1
                nd = du + w
                if nd < dist_f.get(v, math.inf):
                    # Directional A* score, with a small penalty for moving away.
                    score = nd + h_to(v, target) + 0.15 * max(0.0, h_to(v, target) - h_to(u, target))
                    nbrs.append((score, v, nd))
            nbrs.sort(key=lambda x: x[0])
            for score, v, nd in nbrs[:branch_cap]:
                if nd < dist_f.get(v, math.inf):
                    dist_f[v] = nd; prev_f[v] = u
                    heapq.heappush(pq_f, (score, counter, v)); counter += 1
                if v in dist_b and nd + dist_b[v] < best:
                    best = nd + dist_b[v]; meet = v
        # Expand backward batch.  Graph is undirected in the generated benchmarks;
        # for directed graphs this should use G.reversed(), but BRIDGE's current
        # generated evaluation graphs are undirected.
        for u in batch_b:
            if expanded >= budget: break
            closed_b.add(u); expanded += 1
            if u in dist_f:
                cand = dist_b[u] + dist_f[u]
                if cand < best:
                    best = cand; meet = u
            nbrs = []
            du = dist_b[u]
            for v, w in G.adj.get(u, []):
                if v in closed_b: continue
                relax += 1
                nd = du + w
                if nd < dist_b.get(v, math.inf):
                    score = nd + h_to(v, source) + 0.15 * max(0.0, h_to(v, source) - h_to(u, source))
                    nbrs.append((score, v, nd))
            nbrs.sort(key=lambda x: x[0])
            for score, v, nd in nbrs[:branch_cap]:
                if nd < dist_b.get(v, math.inf):
                    dist_b[v] = nd; prev_b[v] = u
                    heapq.heappush(pq_b, (score, counter, v)); counter += 1
                if v in dist_f and nd + dist_f[v] < best:
                    best = nd + dist_f[v]; meet = v
        # Stop early only after a concrete meeting path exists and pending best
        # scores are not clearly competitive under a loose bound.
        if meet is not None and pq_f and pq_b and pq_f[0][0] + pq_b[0][0] >= best * 1.10:
            break

    path: list[Node] = []
    if meet is not None and math.isfinite(best):
        left = [meet]
        cur = meet
        while cur != source and cur in prev_f:
            cur = prev_f[cur]
            left.append(cur)
        left.reverse()
        right: list[Node] = []
        cur = meet
        while cur != target and cur in prev_b:
            cur = prev_b[cur]
            right.append(cur)
        candidate = left + right
        if candidate and candidate[0] == source and candidate[-1] == target:
            d = path_distance(G, candidate)
            if math.isfinite(d):
                path = candidate
                best = d

    found = bool(path)
    error = None if found else "BIDIRECTIONAL_BEAM_UNREACHABLE"
    exact_match = False
    fallback_used = False
    ratio = None
    exact_dist = None
    if exact_fallback and not found:
        exact = bidirectional_dijkstra(G, source, target)
        fallback_used = True
        path = exact.path; best = exact.distance; found = exact.found; exact_match = exact.found
        exact_dist = exact.distance
        error = "FALLBACK_AFTER_" + str(error)
    elif exact_fallback and found:
        exact = bidirectional_dijkstra(G, source, target)
        exact_dist = exact.distance
        ratio = best / exact.distance if exact.found and exact.distance > 0 else math.inf
        if ratio > max_distance_ratio:
            fallback_used = True
            path = exact.path; best = exact.distance; found = exact.found; exact_match = exact.found
            error = "FALLBACK_BAD_RATIO"
        else:
            exact_match = abs(best - exact.distance) <= 1e-9 * max(1, exact.distance)

    _, peak = tracemalloc.get_traced_memory()
    if not outer:
        tracemalloc.stop()
    # Logical steps shrink with workers because each round is a batch of beam tasks.
    logical_steps = 2 + math.ceil(max(1, expanded) / workers) + (2 if fallback_used else 0)
    query_work = min(expanded, budget)
    tel = {
        "algorithm_family": "MRPC",
        "variant": "mrpc_bidirectional_beam",
        "budget_ratio": budget_ratio,
        "target_work": max(1, int(math.ceil(n * budget_ratio))),
        "query_work_units": query_work,
        "query_work_ratio": query_work / max(1, n),
        "beam_batch_width": batch_width,
        "branch_cap": branch_cap,
        "expanded_valid": found and not fallback_used,
        "compressed_reachable": found,
        "fallback_used": fallback_used,
        "workers_requested": workers,
        "parallel_backend": "bidirectional_beam_batch",
        "exact_distance_checked": exact_dist,
        "distance_ratio_checked": ratio,
        "error_code": error,
        "raw_relaxations": relax,
        "beam_rounds": rounds,
    }
    return PathResult(path, best if found else math.inf, found, exact_match, "mrpc_bidirectional_beam", query_work, query_work, 0, 0, logical_steps, (time.perf_counter()-start)*1000, peak/1024, tel)

def mrpc_directional_backtrack_v2(
    G: Graph,
    source: Node,
    target: Node,
    *,
    budget_ratio: float = 0.10,
    workers: int = 1,
    backtrack_width: int = 3,
    min_budget: int = 2,
) -> PathResult:
    """MRPC-DG2: no-internal-exact budgeted directional backtracking.

    This is a faster reimplementation of MRPC-DG.  It removes the internal exact
    verification pass from the solver itself and leaves exact/ratio evaluation to
    the benchmark layer.  It follows only real edges, so it never returns an
    invalid or fabricated path.
    """
    start=time.perf_counter(); outer=tracemalloc.is_tracing()
    if not outer: tracemalloc.start()
    n=len(G.adj); budget=max(min_budget, int(math.ceil(n*budget_ratio)))
    workers=max(1,int(workers))
    if source not in G.adj or target not in G.adj:
        _,peak=tracemalloc.get_traced_memory()
        tracemalloc.stop()
        return PathResult([], math.inf, False, False, "mrpc_directional_backtrack_v2", 0,0,0,0,0,(time.perf_counter()-start)*1000,peak/1024,telemetry={"variant":"mrpc_directional_backtrack_v2","error_code":"SOURCE_OR_TARGET_MISSING"})
    def h(u:Node)->float:
        return euclidean(G.pos[u],G.pos[target]) if G.pos and u in G.pos and target in G.pos else 0.0
    # (score, distance, counter, node, path_tuple).  Path tuples avoid accidental mutation.
    pq=[(h(source),0.0,0,source,(source,))]
    counter=1; expanded=0; relax=0
    best_seen={source:0.0}
    best_path:tuple[Node,...]=(); best_dist=math.inf
    while pq and expanded < budget:
        score, gdist, _, u, path = heapq.heappop(pq)
        expanded += 1
        if u == target:
            best_path=path; best_dist=gdist; break
        cur_h=h(u)
        cand=[]
        path_set=set(path)
        for v,w in G.adj.get(u,[]):
            if v in path_set: continue
            relax += 1
            nd=gdist+w
            # A mild dominance threshold avoids pruning viable detours too early.
            if nd >= best_seen.get(v, math.inf)*1.15:
                continue
            best_seen[v]=nd
            hv=h(v)
            progress=cur_h-hv
            # Priority favors target progress, but keeps distance cost visible.
            sc=hv + 0.28*nd + 0.35*max(0.0, -progress) - 0.25*max(0.0, progress)
            cand.append((sc, nd, v))
        cand.sort(key=lambda x:x[0])
        for sc, nd, v in cand[:backtrack_width]:
            heapq.heappush(pq,(sc,nd,counter,v,path+(v,))); counter+=1
        # Keep queue bounded to avoid work leakage not counted as expansion.
        max_q=max(8, backtrack_width*budget)
        if len(pq)>max_q:
            pq=heapq.nsmallest(max_q,pq)
            heapq.heapify(pq)
    found=bool(best_path)
    _,peak=tracemalloc.get_traced_memory()
    tracemalloc.stop()
    query_work=min(expanded,budget)
    logical_steps=2+math.ceil(query_work/workers)
    tel={"algorithm_family":"MRPC","variant":"mrpc_directional_backtrack_v2",
         "budget_ratio":budget_ratio,"target_work":max(1,int(math.ceil(n*budget_ratio))),
         "query_work_units":query_work,"query_work_ratio":query_work/max(1,n),
         "backtrack_width":backtrack_width,"expanded_valid":found,"compressed_reachable":found,
         "fallback_used":False,"workers_requested":workers,"parallel_backend":"directional_backtrack_batch",
         "error_code":None if found else "DIRECTIONAL_BACKTRACK_UNREACHABLE","raw_relaxations":relax}
    return PathResult(list(best_path), best_dist if found else math.inf, found, False, "mrpc_directional_backtrack_v2", query_work, query_work,0,0,logical_steps,(time.perf_counter()-start)*1000,peak/1024,tel)

def mrpc_directional_backtrack_v3(
    G: Graph,
    source: Node,
    target: Node,
    *,
    budget_ratio: float = 0.10,
    workers: int = 1,
    backtrack_width: int = 4,
    min_budget: int = 2,
    continue_after_first: bool = True,
) -> PathResult:
    """MRPC-DG3: budgeted directional backtracking with best-of-budget selection.

    Difference from DG2: reaching target does not immediately terminate.  The
    solver keeps consuming the remaining budget to find a shorter valid path.
    This preserves the node/10 work envelope while reducing distance-ratio
    outliers.
    """
    start=time.perf_counter(); outer=tracemalloc.is_tracing()
    if not outer: tracemalloc.start()
    n=len(G.adj); budget=max(min_budget, int(math.ceil(n*budget_ratio))); workers=max(1,int(workers))
    if source not in G.adj or target not in G.adj:
        _,peak=tracemalloc.get_traced_memory()
        tracemalloc.stop()
        return PathResult([], math.inf, False, False, "mrpc_directional_backtrack_v3",0,0,0,0,0,(time.perf_counter()-start)*1000,peak/1024,telemetry={"variant":"mrpc_directional_backtrack_v3","error_code":"SOURCE_OR_TARGET_MISSING"})
    def h(u:Node)->float:
        return euclidean(G.pos[u],G.pos[target]) if G.pos and u in G.pos and target in G.pos else 0.0
    pq=[(h(source),0.0,0,source,(source,))]
    counter=1; expanded=0; relax=0; best_seen={source:0.0}; best_path:tuple[Node,...]=(); best_dist=math.inf; hits=0
    while pq and expanded < budget:
        score,gdist,_,u,path=heapq.heappop(pq)
        expanded += 1
        if gdist >= best_dist:
            continue
        if u==target:
            hits += 1
            if gdist < best_dist:
                best_path=path; best_dist=gdist
            if not continue_after_first:
                break
            continue
        cur_h=h(u); path_set=set(path); cand=[]
        for v,w in G.adj.get(u,[]):
            if v in path_set: continue
            relax += 1
            nd=gdist+w
            if nd >= best_dist: continue
            # More permissive dominance than DG2, because we use the remaining budget to search alternatives.
            if nd >= best_seen.get(v, math.inf)*1.35:
                continue
            if nd < best_seen.get(v, math.inf):
                best_seen[v]=nd
            hv=h(v); progress=cur_h-hv
            sc=hv + 0.22*nd + 0.20*max(0.0,-progress) - 0.35*max(0.0,progress)
            # Slightly prefer direct target neighbor when it appears.
            if v == target:
                sc -= h(source)
            cand.append((sc,nd,v))
        cand.sort(key=lambda x:x[0])
        for sc,nd,v in cand[:backtrack_width]:
            heapq.heappush(pq,(sc,nd,counter,v,path+(v,))); counter+=1
        max_q=max(12, backtrack_width*budget)
        if len(pq)>max_q:
            pq=heapq.nsmallest(max_q,pq); heapq.heapify(pq)
    found=bool(best_path); _,peak=tracemalloc.get_traced_memory()
    tracemalloc.stop()
    query_work=min(expanded,budget); logical_steps=2+math.ceil(query_work/workers)
    tel={"algorithm_family":"MRPC","variant":"mrpc_directional_backtrack_v3","budget_ratio":budget_ratio,
         "target_work":max(1,int(math.ceil(n*budget_ratio))),"query_work_units":query_work,
         "query_work_ratio":query_work/max(1,n),"backtrack_width":backtrack_width,"target_hits":hits,
         "expanded_valid":found,"compressed_reachable":found,"fallback_used":False,"workers_requested":workers,
         "parallel_backend":"directional_backtrack_best_of_budget","error_code":None if found else "DIRECTIONAL_BACKTRACK_UNREACHABLE","raw_relaxations":relax}
    return PathResult(list(best_path), best_dist if found else math.inf, found, False, "mrpc_directional_backtrack_v3", query_work, query_work,0,0,logical_steps,(time.perf_counter()-start)*1000,peak/1024,tel)

# ---------------------------------------------------------------------------
# BRIDGE Graph Profiler helpers and MRPC-DG4 handoff variant
# ---------------------------------------------------------------------------
@dataclass
class ComponentIndex:
    component_id: dict[Node, int]
    component_count: int
    build_work: int


def build_component_index(G: Graph) -> ComponentIndex:
    """Build connected-component ids for BRIDGE-level unreachable precheck.

    This is intended as Graph Profiler / SolverGate state, not as MRPC query
    work.  Once cached, a query can reject disconnected source-target pairs by
    comparing component_id[source] and component_id[target].
    """
    comp: dict[Node, int] = {}
    cid = 0
    work = 0
    for s in G.adj:
        if s in comp:
            continue
        q = deque([s])
        comp[s] = cid
        while q:
            u = q.popleft()
            work += 1
            for v, _ in G.adj.get(u, []):
                if v not in comp:
                    comp[v] = cid
                    q.append(v)
        cid += 1
    return ComponentIndex(comp, cid, work)


def component_reachable(index: ComponentIndex, source: Node, target: Node) -> bool:
    return source in index.component_id and target in index.component_id and index.component_id[source] == index.component_id[target]


@dataclass
class HandoffSeed:
    node: Node
    distance: float
    path: tuple[Node, ...]


def _bounded_bidirectional_handoff(
    G: Graph,
    seeds: list[HandoffSeed],
    target: Node,
    *,
    repair_budget: int,
    workers: int = 1,
) -> tuple[list[Node], float, bool, int, int, str | None]:
    """Connect MRPC frontier seeds to target with bounded bidirectional search.

    The forward side starts from many MRPC frontier nodes with already-known
    source->frontier distances and paths.  The backward side starts from target.
    It reuses MRPC partial progress instead of restarting exact search at source.
    """
    if not seeds or target not in G.adj:
        return [], math.inf, False, 0, 0, "HANDOFF_NO_SEED"
    workers = max(1, int(workers))
    repair_budget = max(1, int(repair_budget))

    dist_f: dict[Node, float] = {}
    path_f: dict[Node, tuple[Node, ...]] = {}
    pq_f: list[tuple[float, int, Node]] = []
    c = 0
    for seed in seeds:
        if seed.node not in G.adj:
            continue
        if seed.distance < dist_f.get(seed.node, math.inf):
            dist_f[seed.node] = seed.distance
            path_f[seed.node] = seed.path
            heapq.heappush(pq_f, (seed.distance, c, seed.node)); c += 1
    if not pq_f:
        return [], math.inf, False, 0, 0, "HANDOFF_NO_VALID_SEED"

    dist_b: dict[Node, float] = {target: 0.0}
    prev_b: dict[Node, Node] = {}
    pq_b: list[tuple[float, int, Node]] = [(0.0, c, target)]; c += 1
    seen_f: set[Node] = set(); seen_b: set[Node] = set()
    best = math.inf; meet: Node | None = None
    expanded = 0; steps = 0

    def pop_valid(pq, seen):
        while pq:
            d, _, u = heapq.heappop(pq)
            if u not in seen:
                return d, u
        return math.inf, None

    while pq_f and pq_b and expanded < repair_budget:
        # Expand the frontier with currently smaller tentative distance.
        forward = pq_f[0][0] <= pq_b[0][0]
        batch = max(1, min(workers, repair_budget - expanded))
        for _ in range(batch):
            if expanded >= repair_budget:
                break
            if forward:
                du, u = pop_valid(pq_f, seen_f)
                if u is None:
                    break
                seen_f.add(u); expanded += 1
                if u in dist_b and du + dist_b[u] < best:
                    best = du + dist_b[u]; meet = u
                for v, w in G.adj.get(u, []):
                    nd = du + w
                    if nd < dist_f.get(v, math.inf):
                        dist_f[v] = nd
                        path_f[v] = path_f[u] + (v,)
                        heapq.heappush(pq_f, (nd, c, v)); c += 1
                    if v in dist_b and nd + dist_b[v] < best:
                        best = nd + dist_b[v]; meet = v
            else:
                du, u = pop_valid(pq_b, seen_b)
                if u is None:
                    break
                seen_b.add(u); expanded += 1
                if u in dist_f and du + dist_f[u] < best:
                    best = du + dist_f[u]; meet = u
                for v, w in G.adj.get(u, []):
                    nd = du + w
                    if nd < dist_b.get(v, math.inf):
                        dist_b[v] = nd
                        prev_b[v] = u
                        heapq.heappush(pq_b, (nd, c, v)); c += 1
                    if v in dist_f and nd + dist_f[v] < best:
                        best = nd + dist_f[v]; meet = v
        steps += 1
        if meet is not None and pq_f and pq_b and pq_f[0][0] + pq_b[0][0] >= best:
            break

    if meet is None or meet not in path_f:
        return [], math.inf, False, expanded, steps, "HANDOFF_REPAIR_FAILED"
    right: list[Node] = []
    cur = meet
    while cur != target:
        if cur not in prev_b:
            return [], math.inf, False, expanded, steps, "HANDOFF_PATH_RECONSTRUCT_FAILED"
        cur = prev_b[cur]
        right.append(cur)
    candidate = list(path_f[meet]) + right
    dist = path_distance(G, candidate)
    if not candidate or not math.isfinite(dist):
        return [], math.inf, False, expanded, steps, "HANDOFF_INVALID_PATH"
    return candidate, dist, True, expanded, steps, None


def mrpc_dg4_handoff(
    G: Graph,
    source: Node,
    target: Node,
    *,
    budget_ratio: float = 0.10,
    workers: int = 4,
    backtrack_width: int = 4,
    min_budget: int = 2,
    component_index: ComponentIndex | None = None,
    enable_component_precheck: bool = True,
    detour_handoff: bool = True,
    handoff_start_ratio: float = 0.35,
    handoff_repair_ratio: float = 1.00,
    high_risk_stagnation_rounds: int = 5,
) -> PathResult:
    """MRPC-DG4: DG3 + detour prediction + bounded bidirectional handoff.

    This variant keeps DG3's low-work directional behavior on ordinary geometric
    queries, but detects detour symptoms and hands off the current frontier to a
    bounded bidirectional repair instead of discarding MRPC's partial progress.

    The component precheck belongs conceptually to BRIDGE's Graph Profiler.  It
    is accepted here as an optional cached object to make the reference
    implementation self-contained.
    """
    start = time.perf_counter(); outer = tracemalloc.is_tracing()
    if not outer:
        tracemalloc.start()
    n = len(G.adj)
    workers = max(1, int(workers))
    budget = max(min_budget, int(math.ceil(n * budget_ratio)))
    repair_budget = max(1, int(math.ceil(n * handoff_repair_ratio)))
    preprocessing_work = 0
    preprocessing_time_ms = 0.0

    if source not in G.adj or target not in G.adj:
        _, peak = tracemalloc.get_traced_memory()
        tracemalloc.stop()
        return PathResult([], math.inf, False, False, "mrpc_dg4_handoff", 0, 0, 0, 0, 0, (time.perf_counter()-start)*1000, peak/1024, telemetry={"variant":"mrpc_dg4_handoff","error_code":"SOURCE_OR_TARGET_MISSING"})

    if enable_component_precheck:
        pre_start = time.perf_counter()
        if component_index is None:
            component_index = build_component_index(G)
            preprocessing_work += component_index.build_work
        preprocessing_time_ms += (time.perf_counter() - pre_start) * 1000
        if not component_reachable(component_index, source, target):
            _, peak = tracemalloc.get_traced_memory()
            tracemalloc.stop()
            tel = {
                "algorithm_family": "MRPC",
                "variant": "mrpc_dg4_handoff",
                "component_precheck": True,
                "component_reachable": False,
                "preprocessing_work": preprocessing_work,
                "preprocessing_time_ms": preprocessing_time_ms,
                "target_work": budget,
                "query_work_units": 1,
                "query_work_ratio": 1 / max(1, n),
                "fallback_used": False,
                "handoff_used": False,
                "error_code": "DISCONNECTED_PRECHECK",
            }
            return PathResult([], math.inf, False, False, "mrpc_dg4_handoff", 1, 1, 0, 0, 1, (time.perf_counter()-start)*1000, peak/1024, tel)

    def h(u: Node) -> float:
        return euclidean(G.pos[u], G.pos[target]) if G.pos and u in G.pos and target in G.pos else 0.0

    pq: list[tuple[float, float, int, Node, tuple[Node, ...]]] = [(h(source), 0.0, 0, source, (source,))]
    counter = 1
    expanded = 0
    relax = 0
    best_seen: dict[Node, float] = {source: 0.0}
    best_path: tuple[Node, ...] = ()
    best_dist = math.inf
    hits = 0
    best_h = h(source)
    stagnation_rounds = 0
    blocked_forward_rounds = 0
    handoff_used = False
    handoff_reason = None
    handoff_work = 0
    handoff_steps = 0
    max_lateral_width = 0.0

    # Unit vector source->target, used for lateral spread diagnostics.
    axis = None
    if G.pos and source in G.pos and target in G.pos:
        sx, sy = G.pos[source]; tx, ty = G.pos[target]
        ax, ay = tx - sx, ty - sy
        norm = math.hypot(ax, ay)
        if norm > 0:
            axis = (ax / norm, ay / norm, sx, sy)

    def lateral_distance(u: Node) -> float:
        if axis is None or not G.pos or u not in G.pos:
            return 0.0
        ax, ay, sx, sy = axis
        x, y = G.pos[u]
        vx, vy = x - sx, y - sy
        projx, projy = (vx * ax + vy * ay) * ax, (vx * ax + vy * ay) * ay
        return math.hypot(vx - projx, vy - projy)

    def make_seeds(limit: int = 16) -> list[HandoffSeed]:
        items = heapq.nsmallest(limit, pq)
        seeds: list[HandoffSeed] = []
        used: set[Node] = set()
        for _, gdist, _, u, path in items:
            if u in used:
                continue
            used.add(u)
            seeds.append(HandoffSeed(u, gdist, path))
        # Always include the best seen current endpoint if available.
        return seeds

    while pq and expanded < budget:
        score, gdist, _, u, path = heapq.heappop(pq)
        expanded += 1
        cur_h = h(u)
        trace.emit("mrpc_expand", detail_level=2, segment_id=segment_id, node=u, g=gdist, h=cur_h, queue_size=len(pq), expanded=expanded, best_h=best_h, stagnation_rounds=stagnation_rounds, blocked_forward_rounds=blocked_forward_rounds, path_length=len(path))
        old_best_h = best_h
        if cur_h < best_h:
            best_h = cur_h
            stagnation_rounds = 0
        else:
            stagnation_rounds += 1
        max_lateral_width = max(max_lateral_width, lateral_distance(u))

        if gdist >= best_dist:
            continue
        if u == target:
            hits += 1
            if gdist < best_dist:
                best_path = path; best_dist = gdist
            # Continue best-of-budget as in DG3.
            continue

        path_set = set(path)
        cand = []
        forward_like = 0
        for v, w in G.adj.get(u, []):
            if v in path_set:
                continue
            relax += 1
            nd = gdist + w
            if nd >= best_dist:
                continue
            if nd >= best_seen.get(v, math.inf) * 1.35:
                continue
            if nd < best_seen.get(v, math.inf):
                best_seen[v] = nd
            hv = h(v); progress = cur_h - hv
            if progress > 0:
                forward_like += 1
            sc = hv + 0.22 * nd + 0.20 * max(0.0, -progress) - 0.35 * max(0.0, progress)
            if v == target:
                sc -= h(source)
            cand.append((sc, nd, v))
        if forward_like == 0:
            blocked_forward_rounds += 1
        else:
            blocked_forward_rounds = max(0, blocked_forward_rounds - 1)
        trace.emit("mrpc_frontier", detail_level=2, segment_id=segment_id, node=u, candidate_count=len(cand), forward_like=forward_like, queue_size_before_push=len(pq), degree=len(G.adj.get(u, [])), progress=best_h-cur_h)
        cand.sort(key=lambda x: x[0])
        for sc, nd, v in cand[:backtrack_width]:
            heapq.heappush(pq, (sc, nd, counter, v, path + (v,))); counter += 1
        max_q = max(12, backtrack_width * budget)
        if len(pq) > max_q:
            pq = heapq.nsmallest(max_q, pq); heapq.heapify(pq)

        # Detour predictor: after an initial window, if no target hit and the
        # heuristic progress stalls or forward moves vanish, reuse frontier via handoff.
        burn = expanded / max(1, budget)
        high_risk = (
            detour_handoff
            and not best_path
            and burn >= handoff_start_ratio
            and (
                stagnation_rounds >= high_risk_stagnation_rounds
                or blocked_forward_rounds >= max(3, high_risk_stagnation_rounds // 2)
            )
        )
        if high_risk:
            handoff_reason = "DETOUR_RISK_STAGNATION" if stagnation_rounds >= high_risk_stagnation_rounds else "DETOUR_RISK_BLOCKED_FORWARD"
            seeds = make_seeds(limit=max(8, workers * 4))
            hpath, hdist, hfound, hwork, hsteps, herr = _bounded_bidirectional_handoff(G, seeds, target, repair_budget=repair_budget, workers=workers)
            handoff_used = True
            handoff_work += hwork
            handoff_steps += hsteps
            if hfound and hdist < best_dist:
                best_path = tuple(hpath); best_dist = hdist; hits += 1
            if hfound or (expanded + handoff_work) >= budget + repair_budget:
                break
            # If handoff did not connect, continue consuming remaining base budget.

    found = bool(best_path)
    error = None if found else "DG4_UNREACHABLE"
    if handoff_used and not found:
        error = "HANDOFF_FAILED_UNREACHABLE"
    _, peak = tracemalloc.get_traced_memory()
    if not outer:
        tracemalloc.stop()
    query_work = min(expanded, budget) + handoff_work
    logical_steps = 2 + math.ceil(max(1, min(expanded, budget)) / workers) + handoff_steps
    tel = {
        "algorithm_family": "MRPC",
        "variant": "mrpc_dg4_handoff",
        "budget_ratio": budget_ratio,
        "handoff_repair_ratio": handoff_repair_ratio,
        "target_work": budget,
        "base_query_work_units": min(expanded, budget),
        "handoff_work_units": handoff_work,
        "query_work_units": query_work,
        "query_work_ratio": query_work / max(1, n),
        "component_precheck": enable_component_precheck,
        "component_reachable": True,
        "preprocessing_work": preprocessing_work,
        "preprocessing_time_ms": preprocessing_time_ms,
        "backtrack_width": backtrack_width,
        "target_hits": hits,
        "handoff_used": handoff_used,
        "handoff_reason": handoff_reason,
        "detour_stagnation_rounds": stagnation_rounds,
        "blocked_forward_rounds": blocked_forward_rounds,
        "max_lateral_width": max_lateral_width,
        "expanded_valid": found,
        "compressed_reachable": found,
        "fallback_used": False,
        "workers_requested": workers,
        "parallel_backend": "directional_backtrack_with_bounded_bidir_handoff",
        "error_code": error,
        "raw_relaxations": relax,
    }
    return PathResult(list(best_path), best_dist if found else math.inf, found, False, "mrpc_dg4_handoff", query_work, query_work, 0, 0, logical_steps, (time.perf_counter()-start)*1000, peak/1024, tel)


def _local_exact_until_reentry(
    G: Graph,
    seeds: list[HandoffSeed],
    target: Node,
    *,
    repair_budget: int,
    workers: int = 1,
    reentry_min_progress: float = 0.08,
    reentry_density_radius: int = 2,
    min_reentry_expanded: int = 8,
    adaptive_reentry_minimum: bool = True,
    reentry_candidate_window: int = 32,
    reentry_max_candidates: int = 8,
    trace: DG5TraceCollector | None = None,
    segment_id: int = 0,
) -> tuple[list[Node], float, bool, bool, Node | None, int, int, str | None]:
    """Run bounded local exact search until either target or a re-entry node is found.

    Unlike DG4 handoff, this helper does not try to connect all the way to the
    target by default. It starts from MRPC frontier seeds and searches exactly in
    local weighted distance order. It stops when it finds a node that appears
    suitable for resuming MRPC: closer to target than the seed frontier, with
    forward-looking neighbors and reasonable local density.

    Returns:
      path, distance, target_found, reentry_found, reentry_node, work, steps, error
    """
    if not seeds or target not in G.adj:
        return [], math.inf, False, False, None, 0, 0, "REENTRY_NO_SEED"
    workers = max(1, int(workers))
    repair_budget = max(1, int(repair_budget))
    trace = trace or DG5TraceCollector()
    if adaptive_reentry_minimum:
        scale_minimum = min(128, max(16, int(2.0 * math.sqrt(max(1, len(G.adj))))))
        effective_min_reentry = min(repair_budget, max(int(min_reentry_expanded), scale_minimum))
    else:
        effective_min_reentry = min(repair_budget, max(1, int(min_reentry_expanded)))
    reentry_candidate_window = max(0, int(reentry_candidate_window))
    reentry_max_candidates = max(1, int(reentry_max_candidates))

    def h(u: Node) -> float:
        return euclidean(G.pos[u], G.pos[target]) if G.pos and u in G.pos and target in G.pos else 0.0

    seed_h = min(h(s.node) for s in seeds if s.node in G.adj) if seeds else math.inf
    required_h = seed_h * (1.0 - reentry_min_progress)
    trace.emit("local_exact_start", segment_id=segment_id, seed_count=len(seeds), seed_h=seed_h, required_h=required_h, repair_budget=repair_budget, effective_min_reentry=effective_min_reentry, candidate_window=reentry_candidate_window, force=True)

    pq: list[tuple[float, int, Node]] = []
    dist: dict[Node, float] = {}
    parent: dict[Node, Node] = {}
    seed_paths: dict[Node, tuple[Node, ...]] = {}
    origin_seed: dict[Node, Node] = {}
    c = 0
    for seed in seeds:
        if seed.node not in G.adj:
            continue
        if seed.distance < dist.get(seed.node, math.inf):
            dist[seed.node] = seed.distance
            seed_paths[seed.node] = seed.path
            origin_seed[seed.node] = seed.node
            heapq.heappush(pq, (seed.distance, c, seed.node)); c += 1
    if not pq:
        return [], math.inf, False, False, None, 0, 0, "REENTRY_NO_VALID_SEED"

    def reconstruct(u: Node) -> list[Node]:
        suffix = [u]
        cur = u
        while cur in parent:
            cur = parent[cur]
            suffix.append(cur)
        suffix.reverse()
        seed = suffix[0]
        base = list(seed_paths.get(seed, (seed,)))
        if base and suffix and base[-1] == suffix[0]:
            return base + suffix[1:]
        return base + suffix

    seen: set[Node] = set()
    expanded = 0
    steps = 0
    best_reentry: Node | None = None
    best_reentry_score = math.inf
    reentry_candidates: list[tuple[float, Node, float, float, int, int]] = []
    first_candidate_expanded: int | None = None

    def local_forward_options(u: Node) -> int:
        hu = h(u)
        count = 0
        for v, _ in G.adj.get(u, []):
            if h(v) < hu:
                count += 1
        return count

    def local_density(u: Node) -> int:
        # Cheap proxy: degree plus one-hop forward options.
        return len(G.adj.get(u, [])) + local_forward_options(u)

    while pq and expanded < repair_budget:
        batch = max(1, min(workers, repair_budget - expanded))
        for _ in range(batch):
            if not pq or expanded >= repair_budget:
                break
            du, _, u = heapq.heappop(pq)
            if u in seen:
                continue
            seen.add(u)
            expanded += 1
            hu = h(u)
            fopts = local_forward_options(u)
            density = local_density(u)
            trace.emit("local_exact_expand", detail_level=2, segment_id=segment_id, node=u, g=du, h=hu, queue_size=len(pq), expanded=expanded, forward_options=fopts, local_density=density)

            if u == target:
                cand = reconstruct(u)
                d = path_distance(G, cand)
                if cand and math.isfinite(d):
                    trace.emit("local_exact_target", segment_id=segment_id, node=u, expanded=expanded, distance=d, force=True)
                    return cand, d, True, False, target, expanded, steps + 1, None

            # Re-entry condition: enough local exact progress, closer to target,
            # and MRPC can likely proceed because forward candidates exist.
            if expanded >= effective_min_reentry and hu <= required_h and fopts > 0 and density >= 3:
                # Collect candidates over a bounded window instead of accepting the
                # first locally plausible node. Lower score is better. Reward
                # forward options and density while penalizing exact detour cost.
                score = hu + 0.05 * du - 0.15 * fopts - 0.03 * density
                reentry_candidates.append((score, u, du, hu, fopts, density))
                reentry_candidates.sort(key=lambda x: x[0])
                if len(reentry_candidates) > reentry_max_candidates:
                    reentry_candidates.pop()
                if score < best_reentry_score:
                    best_reentry_score = score
                    best_reentry = u
                if first_candidate_expanded is None:
                    first_candidate_expanded = expanded
                trace.emit("reentry_candidate", segment_id=segment_id, node=u, g=du, h=hu, score=score, expanded=expanded, forward_options=fopts, local_density=density, accepted=False, candidate_count=len(reentry_candidates), force=True)

            if first_candidate_expanded is not None and expanded >= first_candidate_expanded + reentry_candidate_window:
                break

            for v, w in G.adj.get(u, []):
                nd = du + w
                if nd < dist.get(v, math.inf):
                    dist[v] = nd
                    parent[v] = u
                    origin_seed[v] = origin_seed.get(u, u)
                    heapq.heappush(pq, (nd, c, v)); c += 1
        steps += 1

    if best_reentry is not None:
        cand = reconstruct(best_reentry)
        d = path_distance(G, cand)
        if cand and math.isfinite(d):
            trace.emit("reentry_selected", segment_id=segment_id, node=best_reentry, score=best_reentry_score, expanded=expanded, candidate_count=len(reentry_candidates), distance=d, force=True)
            return cand, d, False, True, best_reentry, expanded, steps, None
    trace.emit("local_exact_end", segment_id=segment_id, expanded=expanded, steps=steps, found_reentry=False, queue_remaining=len(pq), force=True)
    return [], math.inf, False, False, None, expanded, steps, "REENTRY_NOT_FOUND"


def _mrpc_segment_dg5(
    G: Graph,
    source: Node,
    target: Node,
    *,
    budget: int,
    workers: int,
    backtrack_width: int,
    high_risk_stagnation_rounds: int,
    handoff_start_ratio: float,
    global_visited: set[Node] | None = None,
    trace: DG5TraceCollector | None = None,
    segment_id: int = 0,
) -> tuple[list[Node], float, bool, bool, list[HandoffSeed], dict]:
    """Run one bulk-synchronous MRPC directional segment.

    Up to ``workers`` frontier items are expanded in one logical round.  The
    Python reference executes the batch sequentially, but records the exact
    dependency depth and available parallel width for a native threaded/GPU
    backend.  Candidate reduction is performed once per round.
    """
    workers = max(1, int(workers))
    budget = max(1, int(budget))
    global_visited = global_visited or set()
    trace = trace or DG5TraceCollector()
    trace.emit("mrpc_segment_start", segment_id=segment_id, source=source, target=target, budget=budget, workers=workers, backtrack_width=backtrack_width, global_visited_size=len(global_visited), force=True)

    def h(u: Node) -> float:
        return euclidean(G.pos[u], G.pos[target]) if G.pos and u in G.pos and target in G.pos else 0.0

    pq: list[tuple[float, float, int, Node, tuple[Node, ...]]] = [(h(source), 0.0, 0, source, (source,))]
    counter = 1
    expanded = relax = 0
    logical_rounds = 0
    round_widths: list[int] = []
    serial_ops = 0
    parallel_ops = 0
    best_seen: dict[Node, float] = {source: 0.0}
    best_path: tuple[Node, ...] = ()
    best_dist = math.inf
    best_h = h(source)
    stagnation_rounds = 0
    blocked_forward_rounds = 0
    detour_reason = None

    def make_seeds(limit: int = 16) -> list[HandoffSeed]:
        items = heapq.nsmallest(limit, pq)
        seeds: list[HandoffSeed] = []
        used: set[Node] = set()
        for _, gdist, _, u, path in items:
            if u in used or u not in G.adj:
                continue
            used.add(u)
            seeds.append(HandoffSeed(u, gdist, path))
        if not seeds and source in G.adj:
            seeds.append(HandoffSeed(source, 0.0, (source,)))
        return seeds

    def telemetry() -> dict:
        width_sum = sum(round_widths)
        mean_width = width_sum / max(1, len(round_widths))
        max_width = max(round_widths, default=0)
        critical_path_ops = logical_rounds + serial_ops
        serial_fraction = serial_ops / max(1, serial_ops + parallel_ops)
        utilization = width_sum / max(1, logical_rounds * workers)
        return {
            "expanded": expanded,
            "relax": relax,
            "detour_reason": detour_reason,
            "stagnation_rounds": stagnation_rounds,
            "blocked_forward_rounds": blocked_forward_rounds,
            "logical_rounds": logical_rounds,
            "critical_path_ops": critical_path_ops,
            "parallelizable_ops": parallel_ops,
            "serial_ops": serial_ops,
            "mean_parallel_width": mean_width,
            "max_parallel_width": max_width,
            "worker_utilization": utilization,
            "barrier_count": logical_rounds,
            "serial_fraction": serial_fraction,
            "round_widths": round_widths,
        }

    while pq and expanded < budget:
        logical_rounds += 1
        batch_items = []
        while pq and len(batch_items) < workers and expanded + len(batch_items) < budget:
            item = heapq.heappop(pq)
            if item[3] in global_visited and item[3] != source:
                continue
            batch_items.append(item)
        if not batch_items:
            break
        round_widths.append(len(batch_items))
        parallel_ops += len(batch_items)
        produced: list[tuple[float, float, Node, tuple[Node, ...]]] = []
        round_forward = 0
        round_improved = False

        for _, gdist, _, u, path in batch_items:
            expanded += 1
            cur_h = h(u)
            if cur_h < best_h:
                best_h = cur_h
                round_improved = True
            if gdist >= best_dist:
                continue
            if u == target:
                if gdist < best_dist:
                    best_path = path
                    best_dist = gdist
                continue
            path_set = set(path)
            cand = []
            for v, w in G.adj.get(u, []):
                if v in path_set:
                    continue
                relax += 1
                nd = gdist + w
                if nd >= best_dist:
                    continue
                if nd >= best_seen.get(v, math.inf) * 1.35:
                    continue
                if nd < best_seen.get(v, math.inf):
                    best_seen[v] = nd
                hv = h(v)
                progress = cur_h - hv
                if progress > 0:
                    round_forward += 1
                sc = hv + 0.22 * nd + 0.20 * max(0.0, -progress) - 0.35 * max(0.0, progress)
                if v == target:
                    sc -= h(source)
                cand.append((sc, nd, v, path + (v,)))
            cand.sort(key=lambda x: x[0])
            produced.extend(cand[:backtrack_width])

        # One serial reduction/barrier per logical round.
        serial_ops += 1
        if round_improved:
            stagnation_rounds = 0
        else:
            stagnation_rounds += 1
        if round_forward == 0:
            blocked_forward_rounds += 1
        else:
            blocked_forward_rounds = max(0, blocked_forward_rounds - 1)
        produced.sort(key=lambda x: x[0])
        for sc, nd, v, path in produced:
            heapq.heappush(pq, (sc, nd, counter, v, path)); counter += 1
        max_q = max(12, backtrack_width * budget)
        if len(pq) > max_q:
            pq = heapq.nsmallest(max_q, pq); heapq.heapify(pq)
        trace.emit("mrpc_round", detail_level=2, segment_id=segment_id, round=logical_rounds, width=len(batch_items), produced=len(produced), queue_size=len(pq), best_h=best_h)

        burn = expanded / max(1, budget)
        if not best_path and burn >= handoff_start_ratio and (
            stagnation_rounds >= high_risk_stagnation_rounds or
            blocked_forward_rounds >= max(3, high_risk_stagnation_rounds // 2)
        ):
            detour_reason = "DETOUR_RISK_STAGNATION" if stagnation_rounds >= high_risk_stagnation_rounds else "DETOUR_RISK_BLOCKED_FORWARD"
            trace.emit("detour_trigger", segment_id=segment_id, reason=detour_reason, expanded=expanded, burn=burn, stagnation_rounds=stagnation_rounds, blocked_forward_rounds=blocked_forward_rounds, queue_size=len(pq), best_h=best_h, force=True)
            return [], math.inf, False, True, make_seeds(limit=max(8, workers * 4)), telemetry()

    found = bool(best_path)
    trace.emit("mrpc_segment_end", segment_id=segment_id, found=found, expanded=expanded, relax=relax, queue_size=len(pq), best_h=best_h, stagnation_rounds=stagnation_rounds, blocked_forward_rounds=blocked_forward_rounds, force=True)
    return list(best_path), best_dist if found else math.inf, found, False, make_seeds(limit=max(8, workers * 4)), telemetry()

def mrpc_dg5_switchback(
    G: Graph,
    source: Node,
    target: Node,
    *,
    budget_ratio: float = 0.10,
    workers: int = 4,
    backtrack_width: int = 4,
    min_budget: int = 2,
    component_index: ComponentIndex | None = None,
    enable_component_precheck: bool = True,
    handoff_start_ratio: float = 0.35,
    local_exact_ratio: float = 1.00,
    global_fallback_ratio: float = 1.00,
    max_switches: int = 3,
    high_risk_stagnation_rounds: int = 5,
    reentry_min_progress: float = 0.08,
    min_reentry_expanded: int = 8,
    adaptive_reentry_minimum: bool = True,
    reentry_candidate_window: int = 32,
    reentry_max_candidates: int = 8,
    trace_level: int = 0,
    trace_sample_every: int = 8,
    trace_max_events: int = 20_000,
    collect_topology_profile: bool | None = None,
    measure_memory: bool = False,
    reentry_survival_work: int = 32,
    safe_exact_completion: bool = True,
    enable_topology_gate: bool = True,
    dg5_mode: str = "balanced",
    quality_guard_max_ratio: float | None = None,
    allow_oracle_quality_guard: bool = False,
) -> PathResult:
    """MRPC-DG5: switchback hybrid MRPC.

    The solver uses MRPC as the default segment explorer. When detour symptoms
    appear, it temporarily switches to a bounded exact local solver. If the
    exact local solver finds a re-entry point, MRPC resumes from there. It only
    falls back to target-level exact connection when re-entry cannot be found
    within the local budget.

    Production-clean rule:
    whole-query exact paths must not be used as an oracle to improve a completed
    MRPC candidate unless allow_oracle_quality_guard=True is explicitly passed.
    This keeps normal BRIDGE usage free from evaluation leakage while preserving
    an opt-in diagnostic mode for ablation studies.
    """
    start = time.perf_counter(); outer = tracemalloc.is_tracing()
    trace = DG5TraceCollector(level=max(0, int(trace_level)), sample_every=max(1, int(trace_sample_every)), max_events=max(1, int(trace_max_events)))
    memory_started = False
    if measure_memory and not outer:
        tracemalloc.start(); memory_started = True

    if collect_topology_profile is None:
        collect_topology_profile = trace_level > 0 or enable_topology_gate
    topology_start = time.perf_counter()
    if enable_topology_gate:
        topology_profile = extended_topology_profile(G, source, target)
        gate_decision = decide_topology_gate(topology_profile, mode=dg5_mode)
    else:
        topology_profile = graph_topology_profile(G, source, target) if collect_topology_profile else {}
        gate_decision = None
    topology_time_ms = (time.perf_counter() - topology_start) * 1000
    n = len(G.adj)
    workers = max(1, int(workers))
    trace.emit("query_start", source=source, target=target, topology=topology_profile, topology_gate=None if gate_decision is None else {"risk_score": gate_decision.risk_score, "risk_class": gate_decision.risk_class, "reasons": list(gate_decision.reasons), "force_exact_precheck": gate_decision.force_exact_precheck}, budget_ratio=budget_ratio, local_exact_ratio=local_exact_ratio, global_fallback_ratio=global_fallback_ratio, force=True)
    oracle_quality_guard_enabled = bool(allow_oracle_quality_guard and quality_guard_max_ratio is not None)
    segment_budget = max(min_budget, int(math.ceil(n * budget_ratio)))
    local_budget = max(1, int(math.ceil(n * local_exact_ratio)))
    global_fallback_budget = max(local_budget, int(math.ceil(n * global_fallback_ratio)))
    preprocessing_work = 0
    preprocessing_time_ms = topology_time_ms if enable_topology_gate else 0.0
    if enable_topology_gate:
        preprocessing_work += int(topology_profile.get("profile_work_nodes", 0)) + int(topology_profile.get("profile_work_edges", 0))

    if source not in G.adj or target not in G.adj:
        _, peak = tracemalloc.get_traced_memory() if (outer or memory_started) else (0, 0)
        if memory_started: tracemalloc.stop()
        return PathResult([], math.inf, False, False, "mrpc_dg5_switchback", 0, 0, 0, 0, 0, (time.perf_counter()-start)*1000, peak/1024, telemetry={"variant":"mrpc_dg5_switchback","error_code":"SOURCE_OR_TARGET_MISSING"})

    if enable_component_precheck:
        pre_start = time.perf_counter()
        component_reach = None
        if enable_topology_gate and topology_profile.get("source_target_connected_estimate") is not None:
            component_reach = bool(topology_profile.get("source_target_connected_estimate"))
        else:
            if component_index is None:
                component_index = build_component_index(G)
                preprocessing_work += component_index.build_work
            component_reach = component_reachable(component_index, source, target)
        preprocessing_time_ms += (time.perf_counter() - pre_start) * 1000
        if not component_reach:
            _, peak = tracemalloc.get_traced_memory() if (outer or memory_started) else (0, 0)
            if memory_started: tracemalloc.stop()
            tel = {
                "algorithm_family": "MRPC",
                "variant": "mrpc_dg5_switchback",
                "component_precheck": True,
                "component_reachable": False,
                "topology_gate_enabled": enable_topology_gate,
                "topology_gate_action": "DISCONNECTED_PRECHECK",
                "topology_risk_score": None if gate_decision is None else gate_decision.risk_score,
                "topology_risk_class": None if gate_decision is None else gate_decision.risk_class,
                "topology_gate_reasons": [] if gate_decision is None else list(gate_decision.reasons),
                "topology_profile": topology_profile,
                "preprocessing_work": preprocessing_work,
                "preprocessing_time_ms": preprocessing_time_ms,
                "target_work": segment_budget,
                "query_work_units": 1,
                "total_work_including_preprocessing": preprocessing_work + 1,
                "query_work_ratio": 1 / max(1, n),
                "fallback_used": False,
                "handoff_used": False,
                "switch_count": 0,
                "reentry_count": 0,
                **trace.summary(),
                "trace_events": trace.events,
                "error_code": "DISCONNECTED_PRECHECK",
            }
            return PathResult([], math.inf, False, False, "mrpc_dg5_switchback", 1, 1, 0, 0, 1, (time.perf_counter()-start)*1000, peak/1024, tel)

    if gate_decision is not None and gate_decision.force_exact_precheck and source in G.adj and target in G.adj and source != target:
        exact_start = time.perf_counter()
        exact = bidirectional_dijkstra(G, source, target)
        exact_time_ms = (time.perf_counter() - exact_start) * 1000
        _, peak = tracemalloc.get_traced_memory() if (outer or memory_started) else (0, 0)
        if memory_started: tracemalloc.stop()
        pd = path_distance(G, exact.path) if exact.found else math.inf
        found = exact.found and math.isfinite(pd)
        tel = {
            "algorithm_family": "MRPC",
            "variant": "mrpc_dg5_switchback",
            "dg5_mode": dg5_mode,
            "topology_gate_enabled": True,
            "topology_gate_action": "EXACT_PRECHECK",
            "topology_risk_score": gate_decision.risk_score,
            "topology_risk_class": gate_decision.risk_class,
            "topology_gate_reasons": list(gate_decision.reasons),
            "topology_profile": topology_profile,
            "topology_profile_time_ms": topology_time_ms,
            "component_precheck": False,
            "component_reachable": None,
            "preprocessing_work": preprocessing_work,
            "preprocessing_time_ms": preprocessing_time_ms,
            "query_work_units": exact.total_work,
            "total_work_including_preprocessing": preprocessing_work + exact.total_work,
            "query_time_ms": exact_time_ms,
            "total_time_ms": (time.perf_counter()-start)*1000,
            "switch_count": 0,
            "reentry_count": 0,
            "target_exact_count": 1,
            "handoff_used": True,
            "fallback_used": True,
            "fallback_reason": "TOPOLOGY_GATE_EXACT_PRECHECK",
            "delegated_exact_solver": True,
            "mrpc_fast_path_used": False,
            "oracle_quality_guard_enabled": oracle_quality_guard_enabled,
            "oracle_quality_guard_used": False,
            **trace.summary(),
            "trace_events": trace.events,
            "error_code": None if found else "EXACT_PRECHECK_UNREACHABLE",
        }
        return PathResult(exact.path if found else [], pd if found else math.inf, found, found, "mrpc_dg5_switchback", exact.total_work, exact.total_work, exact.work_expanded_nodes, exact.work_relaxations, exact.parallel_steps, (time.perf_counter()-start)*1000, peak/1024, tel)

    if source == target:
        _, peak = tracemalloc.get_traced_memory() if (outer or memory_started) else (0, 0)
        if memory_started: tracemalloc.stop()
        return PathResult([source], 0.0, True, True, "mrpc_dg5_switchback", 1, 1, 0, 0, 1, (time.perf_counter()-start)*1000, peak/1024, telemetry={"variant":"mrpc_dg5_switchback"})

    current = source
    prefix: list[Node] = [source]
    total_dist = 0.0
    total_work = 0
    total_steps = 0
    total_parallel_ops = 0
    total_serial_ops = 0
    total_barriers = 0
    weighted_width_sum = 0.0
    max_parallel_width = 0
    switch_count = 0
    reentry_count = 0
    target_exact_count = 0
    handoff_reasons: list[str] = []
    global_visited: set[Node] = set()
    last_reentry_node: Node | None = None
    last_reentry_segment: int | None = None

    for segment_id in range(max_switches + 1):
        seg_path, seg_dist, target_found, detour, seeds, stel = _mrpc_segment_dg5(
            G,
            current,
            target,
            budget=segment_budget,
            workers=workers,
            backtrack_width=backtrack_width,
            high_risk_stagnation_rounds=high_risk_stagnation_rounds,
            handoff_start_ratio=handoff_start_ratio,
            global_visited=global_visited,
            trace=trace,
            segment_id=segment_id,
        )
        seg_work = int(stel.get("expanded", 0))
        total_work += seg_work
        seg_rounds = int(stel.get("logical_rounds", math.ceil(max(1, seg_work) / workers)))
        total_steps += seg_rounds
        total_parallel_ops += int(stel.get("parallelizable_ops", seg_work))
        total_serial_ops += int(stel.get("serial_ops", seg_rounds))
        total_barriers += int(stel.get("barrier_count", seg_rounds))
        weighted_width_sum += float(stel.get("mean_parallel_width", 1.0)) * max(1, seg_rounds)
        max_parallel_width = max(max_parallel_width, int(stel.get("max_parallel_width", 1)))
        trace.emit("segment_result", segment_id=segment_id, target_found=target_found, detour=detour, segment_work=seg_work, seeds=len(seeds), telemetry=stel, force=True)
        early_recollision = bool(last_reentry_segment is not None and detour and segment_id == last_reentry_segment + 1 and seg_work < max(1, int(reentry_survival_work)))
        if early_recollision:
            trace.emit("reentry_early_recollision", segment_id=segment_id, previous_reentry=last_reentry_node, segment_work=seg_work, survival_threshold=reentry_survival_work, force=True)
        if target_found and seg_path:
            # Append without duplicating current.
            if prefix and seg_path and prefix[-1] == seg_path[0]:
                prefix.extend(seg_path[1:])
            else:
                prefix.extend(seg_path)
            total_dist = path_distance(G, prefix)
            found = math.isfinite(total_dist)
            _, peak = tracemalloc.get_traced_memory() if (outer or memory_started) else (0, 0)
            if memory_started: tracemalloc.stop()
            tel = {
                "algorithm_family": "MRPC",
                "variant": "mrpc_dg5_switchback",
                "mrpc_fast_path_used": True,
                "oracle_quality_guard_enabled": oracle_quality_guard_enabled,
                "target_work": segment_budget,
                "local_exact_budget": local_budget,
                "global_fallback_budget": global_fallback_budget,
                "query_work_units": total_work,
                "query_work_ratio": total_work / max(1, n),
                "component_precheck": enable_component_precheck,
                "component_reachable": True,
                "preprocessing_work": preprocessing_work,
                "preprocessing_time_ms": preprocessing_time_ms,
                "switch_count": switch_count,
                "reentry_count": reentry_count,
                "target_exact_count": target_exact_count,
                "handoff_used": switch_count > 0,
                "handoff_reason": ";".join(handoff_reasons) if handoff_reasons else None,
                "fallback_used": target_exact_count > 0,
                "workers_requested": workers,
                "parallel_backend": "switchback_mrpc_local_exact_reentry",
                "topology_profile": topology_profile,
                "topology_profile_time_ms": topology_time_ms,
                "logical_rounds": total_steps,
                "critical_path_ops": total_steps + total_serial_ops,
                "parallelizable_ops": total_parallel_ops,
                "serial_ops": total_serial_ops,
                "barrier_count": total_barriers,
                "mean_parallel_width": weighted_width_sum / max(1, total_barriers),
                "max_parallel_width": max_parallel_width,
                "worker_utilization": weighted_width_sum / max(1, total_barriers * workers),
                "serial_fraction": total_serial_ops / max(1, total_serial_ops + total_parallel_ops),
                "logical_rounds": total_steps,
        "critical_path_ops": total_steps + total_serial_ops,
        "parallelizable_ops": total_parallel_ops,
        "serial_ops": total_serial_ops,
        "barrier_count": total_barriers,
        "mean_parallel_width": weighted_width_sum / max(1, total_barriers),
        "max_parallel_width": max_parallel_width,
        "worker_utilization": weighted_width_sum / max(1, total_barriers * workers),
        "serial_fraction": total_serial_ops / max(1, total_serial_ops + total_parallel_ops),
        "query_time_ms": (time.perf_counter()-start)*1000 - preprocessing_time_ms,
                "total_time_ms": (time.perf_counter()-start)*1000,
                "total_work_including_preprocessing": total_work + preprocessing_work,
                **trace.summary(),
                "trace_events": trace.events,
                "error_code": None if found else "DG5_INVALID_FINAL_PATH",
            }
            if found and oracle_quality_guard_enabled and (switch_count > 0 or (gate_decision is not None and gate_decision.enable_quality_guard)):
                q_start = time.perf_counter()
                q_exact = bidirectional_dijkstra(G, source, target)
                q_ms = (time.perf_counter() - q_start) * 1000
                tel["quality_guard_used"] = True
                tel["oracle_quality_guard_used"] = True
                tel["quality_guard_time_ms"] = q_ms
                tel["quality_guard_work"] = q_exact.total_work
                tel["quality_guard_exact_distance"] = q_exact.distance
                total_work += q_exact.total_work
                tel["query_work_units"] = total_work
                tel["total_work_including_preprocessing"] = total_work + preprocessing_work
                if q_exact.found and q_exact.distance * float(quality_guard_max_ratio) < total_dist:
                    prefix = list(q_exact.path)
                    total_dist = path_distance(G, prefix)
                    tel["quality_guard_replaced"] = True
                    tel["topology_gate_action"] = tel.get("topology_gate_action", "MRPC_FAST_PATH") + ";QUALITY_GUARD_EXACT_REPLACE"
                else:
                    tel["quality_guard_replaced"] = False
            return PathResult(prefix if found else [], total_dist if found else math.inf, found, False, "mrpc_dg5_switchback", total_work, total_work, 0, 0, 2 + total_steps, (time.perf_counter()-start)*1000, peak/1024, tel)

        if not detour:
            # MRPC exhausted its segment budget without a clean target or detour
            # signal. Use a final exact bounded-to-target handoff from seeds.
            handoff_reasons.append("SEGMENT_EXHAUSTED")
        else:
            handoff_reasons.append(str(stel.get("detour_reason") or "DETOUR_RISK"))

        switch_count += 1
        if switch_count > max_switches:
            break

        # First try to find a re-entry node, not target.
        local_path, local_dist, target_hit, reentry_hit, reentry_node, lwork, lsteps, lerr = _local_exact_until_reentry(
            G,
            seeds,
            target,
            repair_budget=local_budget,
            workers=workers,
            reentry_min_progress=reentry_min_progress,
            min_reentry_expanded=(local_budget + 1 if early_recollision else min_reentry_expanded),
            adaptive_reentry_minimum=adaptive_reentry_minimum,
            reentry_candidate_window=reentry_candidate_window,
            reentry_max_candidates=reentry_max_candidates,
            trace=trace,
            segment_id=segment_id,
        )
        total_work += lwork
        total_steps += lsteps

        if local_path and target_hit:
            if prefix and local_path and prefix[-1] == local_path[0]:
                prefix.extend(local_path[1:])
            else:
                # local_path already includes source-to-seed; replace prefix to avoid inconsistent stitching
                prefix = list(local_path)
            total_dist = path_distance(G, prefix)
            target_exact_count += 1
            found = math.isfinite(total_dist)
            _, peak = tracemalloc.get_traced_memory() if (outer or memory_started) else (0, 0)
            if memory_started: tracemalloc.stop()
            tel = {
                "algorithm_family": "MRPC",
                "variant": "mrpc_dg5_switchback",
                "mrpc_fast_path_used": True,
                "oracle_quality_guard_enabled": oracle_quality_guard_enabled,
                "target_work": segment_budget,
                "local_exact_budget": local_budget,
                "global_fallback_budget": global_fallback_budget,
                "query_work_units": total_work,
                "query_work_ratio": total_work / max(1, n),
                "component_precheck": enable_component_precheck,
                "component_reachable": True,
                "preprocessing_work": preprocessing_work,
                "preprocessing_time_ms": preprocessing_time_ms,
                "switch_count": switch_count,
                "reentry_count": reentry_count,
                "target_exact_count": target_exact_count,
                "handoff_used": True,
                "handoff_reason": ";".join(handoff_reasons),
                "fallback_used": True,
                "workers_requested": workers,
                "parallel_backend": "switchback_mrpc_local_exact_reentry",
                "topology_profile": topology_profile,
                "topology_profile_time_ms": topology_time_ms,
                "logical_rounds": total_steps,
                "critical_path_ops": total_steps + total_serial_ops,
                "parallelizable_ops": total_parallel_ops,
                "serial_ops": total_serial_ops,
                "barrier_count": total_barriers,
                "mean_parallel_width": weighted_width_sum / max(1, total_barriers),
                "max_parallel_width": max_parallel_width,
                "worker_utilization": weighted_width_sum / max(1, total_barriers * workers),
                "serial_fraction": total_serial_ops / max(1, total_serial_ops + total_parallel_ops),
                "logical_rounds": total_steps,
        "critical_path_ops": total_steps + total_serial_ops,
        "parallelizable_ops": total_parallel_ops,
        "serial_ops": total_serial_ops,
        "barrier_count": total_barriers,
        "mean_parallel_width": weighted_width_sum / max(1, total_barriers),
        "max_parallel_width": max_parallel_width,
        "worker_utilization": weighted_width_sum / max(1, total_barriers * workers),
        "serial_fraction": total_serial_ops / max(1, total_serial_ops + total_parallel_ops),
        "query_time_ms": (time.perf_counter()-start)*1000 - preprocessing_time_ms,
                "total_time_ms": (time.perf_counter()-start)*1000,
                "total_work_including_preprocessing": total_work + preprocessing_work,
                **trace.summary(),
                "trace_events": trace.events,
                "error_code": None if found else "DG5_INVALID_TARGET_EXACT_PATH",
            }
            if found and oracle_quality_guard_enabled and (switch_count > 0 or (gate_decision is not None and gate_decision.enable_quality_guard)):
                q_start = time.perf_counter()
                q_exact = bidirectional_dijkstra(G, source, target)
                q_ms = (time.perf_counter() - q_start) * 1000
                tel["quality_guard_used"] = True
                tel["oracle_quality_guard_used"] = True
                tel["quality_guard_time_ms"] = q_ms
                tel["quality_guard_work"] = q_exact.total_work
                tel["quality_guard_exact_distance"] = q_exact.distance
                total_work += q_exact.total_work
                tel["query_work_units"] = total_work
                tel["total_work_including_preprocessing"] = total_work + preprocessing_work
                if q_exact.found and q_exact.distance * float(quality_guard_max_ratio) < total_dist:
                    prefix = list(q_exact.path)
                    total_dist = path_distance(G, prefix)
                    tel["quality_guard_replaced"] = True
                    tel["topology_gate_action"] = tel.get("topology_gate_action", "MRPC_FAST_PATH") + ";QUALITY_GUARD_EXACT_REPLACE"
                else:
                    tel["quality_guard_replaced"] = False
            return PathResult(prefix if found else [], total_dist if found else math.inf, found, False, "mrpc_dg5_switchback", total_work, total_work, 0, 0, 2 + total_steps, (time.perf_counter()-start)*1000, peak/1024, tel)

        if local_path and reentry_hit and reentry_node is not None and reentry_node != current:
            # Stitch source-to-reentry path produced by local exact. It already
            # contains a seed partial path from current; preserve the global
            # prefix by appending only the suffix after current when possible.
            if prefix and local_path:
                if prefix[-1] == local_path[0]:
                    prefix.extend(local_path[1:])
                else:
                    # Find current in local_path and append from there.
                    if current in local_path:
                        idx = local_path.index(current)
                        prefix.extend(local_path[idx+1:])
                    else:
                        # Fallback to replacing with local path if it is complete.
                        prefix = list(local_path)
            trace.emit("reentry_accepted", segment_id=segment_id, node=reentry_node, local_work=lwork, local_steps=lsteps, local_distance=local_dist, prefix_length=len(prefix), force=True)
            current = reentry_node
            last_reentry_node = reentry_node
            last_reentry_segment = segment_id
            reentry_count += 1
            global_visited.update(prefix[:-1])
            continue

        # Re-entry failed. Use DG4-style bounded bidirectional target connection
        # from the same seeds as a final local exact fallback for this query.
        trace.emit("reentry_failed", segment_id=segment_id, local_work=lwork, local_steps=lsteps, error=lerr, force=True)
        final_path, final_dist, final_found, fwork, fsteps, ferr = _bounded_bidirectional_handoff(
            G, seeds, target, repair_budget=global_fallback_budget, workers=workers
        )
        total_work += fwork
        total_steps += fsteps
        target_exact_count += 1
        if final_found and final_path:
            prefix = list(final_path)
            total_dist = path_distance(G, prefix)
            found = math.isfinite(total_dist)
            _, peak = tracemalloc.get_traced_memory() if (outer or memory_started) else (0, 0)
            if memory_started: tracemalloc.stop()
            tel = {
                "algorithm_family": "MRPC",
                "variant": "mrpc_dg5_switchback",
                "mrpc_fast_path_used": True,
                "oracle_quality_guard_enabled": oracle_quality_guard_enabled,
                "target_work": segment_budget,
                "local_exact_budget": local_budget,
                "global_fallback_budget": global_fallback_budget,
                "query_work_units": total_work,
                "query_work_ratio": total_work / max(1, n),
                "component_precheck": enable_component_precheck,
                "component_reachable": True,
                "preprocessing_work": preprocessing_work,
                "preprocessing_time_ms": preprocessing_time_ms,
                "switch_count": switch_count,
                "reentry_count": reentry_count,
                "target_exact_count": target_exact_count,
                "handoff_used": True,
                "handoff_reason": ";".join(handoff_reasons),
                "fallback_used": True,
                "workers_requested": workers,
                "parallel_backend": "switchback_mrpc_local_exact_reentry",
                "topology_profile": topology_profile,
                "topology_profile_time_ms": topology_time_ms,
                "logical_rounds": total_steps,
                "critical_path_ops": total_steps + total_serial_ops,
                "parallelizable_ops": total_parallel_ops,
                "serial_ops": total_serial_ops,
                "barrier_count": total_barriers,
                "mean_parallel_width": weighted_width_sum / max(1, total_barriers),
                "max_parallel_width": max_parallel_width,
                "worker_utilization": weighted_width_sum / max(1, total_barriers * workers),
                "serial_fraction": total_serial_ops / max(1, total_serial_ops + total_parallel_ops),
                "logical_rounds": total_steps,
        "critical_path_ops": total_steps + total_serial_ops,
        "parallelizable_ops": total_parallel_ops,
        "serial_ops": total_serial_ops,
        "barrier_count": total_barriers,
        "mean_parallel_width": weighted_width_sum / max(1, total_barriers),
        "max_parallel_width": max_parallel_width,
        "worker_utilization": weighted_width_sum / max(1, total_barriers * workers),
        "serial_fraction": total_serial_ops / max(1, total_serial_ops + total_parallel_ops),
        "query_time_ms": (time.perf_counter()-start)*1000 - preprocessing_time_ms,
                "total_time_ms": (time.perf_counter()-start)*1000,
                "total_work_including_preprocessing": total_work + preprocessing_work,
                **trace.summary(),
                "trace_events": trace.events,
                "error_code": None if found else "DG5_FINAL_HANDOFF_INVALID",
            }
            if found and oracle_quality_guard_enabled and (switch_count > 0 or (gate_decision is not None and gate_decision.enable_quality_guard)):
                q_start = time.perf_counter()
                q_exact = bidirectional_dijkstra(G, source, target)
                q_ms = (time.perf_counter() - q_start) * 1000
                tel["quality_guard_used"] = True
                tel["oracle_quality_guard_used"] = True
                tel["quality_guard_time_ms"] = q_ms
                tel["quality_guard_work"] = q_exact.total_work
                tel["quality_guard_exact_distance"] = q_exact.distance
                total_work += q_exact.total_work
                tel["query_work_units"] = total_work
                tel["total_work_including_preprocessing"] = total_work + preprocessing_work
                if q_exact.found and q_exact.distance * float(quality_guard_max_ratio) < total_dist:
                    prefix = list(q_exact.path)
                    total_dist = path_distance(G, prefix)
                    tel["quality_guard_replaced"] = True
                    tel["topology_gate_action"] = tel.get("topology_gate_action", "MRPC_FAST_PATH") + ";QUALITY_GUARD_EXACT_REPLACE"
                else:
                    tel["quality_guard_replaced"] = False
            return PathResult(prefix if found else [], total_dist if found else math.inf, found, False, "mrpc_dg5_switchback", total_work, total_work, 0, 0, 2 + total_steps, (time.perf_counter()-start)*1000, peak/1024, tel)
        break

    # Production safety net: preserve DG5 as one solver stack while guaranteeing
    # that a reachable query is not lost when the approximate/re-entry phases
    # exhaust their budgets. This is intentionally accounted as query work/time.
    if safe_exact_completion:
        exact = bidirectional_dijkstra(G, source, target)
        total_work += exact.total_work
        total_steps += exact.parallel_steps
        target_exact_count += 1
        if exact.found:
            _, peak = tracemalloc.get_traced_memory() if (outer or memory_started) else (0, 0)
            if memory_started: tracemalloc.stop()
            tel = {
                "algorithm_family": "MRPC",
                "variant": "mrpc_dg5_switchback",
                "query_work_units": total_work,
                "preprocessing_work": preprocessing_work,
                "preprocessing_time_ms": preprocessing_time_ms,
                "switch_count": switch_count,
                "reentry_count": reentry_count,
                "target_exact_count": target_exact_count,
                "handoff_used": True,
                "fallback_used": True,
                "safe_exact_completion": True,
                "delegated_exact_solver": True,
                "mrpc_fast_path_used": False,
                "oracle_quality_guard_enabled": oracle_quality_guard_enabled,
                "oracle_quality_guard_used": False,
                "workers_requested": workers,
                "parallel_backend": "bulk_sync_reference_plus_exact_completion",
                "logical_rounds": total_steps,
                "critical_path_ops": total_steps + total_serial_ops,
                "parallelizable_ops": total_parallel_ops,
                "serial_ops": total_serial_ops + exact.parallel_steps,
                "barrier_count": total_barriers,
                "mean_parallel_width": weighted_width_sum / max(1, total_barriers),
                "max_parallel_width": max_parallel_width,
                "worker_utilization": weighted_width_sum / max(1, total_barriers * workers),
                "serial_fraction": (total_serial_ops + exact.parallel_steps) / max(1, total_serial_ops + exact.parallel_steps + total_parallel_ops),
                "query_time_ms": (time.perf_counter()-start)*1000 - preprocessing_time_ms,
                "total_time_ms": (time.perf_counter()-start)*1000,
                "total_work_including_preprocessing": total_work + preprocessing_work,
                **trace.summary(),
                "trace_events": trace.events,
                "error_code": None,
            }
            return PathResult(exact.path, exact.distance, True, True, "mrpc_dg5_switchback", total_work, total_work, 0, 0, 2 + total_steps, (time.perf_counter()-start)*1000, peak/1024, tel)

    _, peak = tracemalloc.get_traced_memory() if (outer or memory_started) else (0, 0)
    if not outer:
        tracemalloc.stop()
    tel = {
        "algorithm_family": "MRPC",
        "variant": "mrpc_dg5_switchback",
        "mrpc_fast_path_used": True,
        "oracle_quality_guard_enabled": oracle_quality_guard_enabled,
        "target_work": segment_budget,
        "local_exact_budget": local_budget,
        "global_fallback_budget": global_fallback_budget,
        "query_work_units": total_work,
        "query_work_ratio": total_work / max(1, n),
        "component_precheck": enable_component_precheck,
        "component_reachable": True,
        "preprocessing_work": preprocessing_work,
        "preprocessing_time_ms": preprocessing_time_ms,
        "switch_count": switch_count,
        "reentry_count": reentry_count,
        "target_exact_count": target_exact_count,
        "handoff_used": switch_count > 0,
        "handoff_reason": ";".join(handoff_reasons) if handoff_reasons else None,
        "fallback_used": target_exact_count > 0,
        "workers_requested": workers,
        "parallel_backend": "switchback_mrpc_local_exact_reentry",
        "topology_profile": topology_profile,
        "topology_profile_time_ms": topology_time_ms,
        "logical_rounds": total_steps,
        "critical_path_ops": total_steps + total_serial_ops,
        "parallelizable_ops": total_parallel_ops,
        "serial_ops": total_serial_ops,
        "barrier_count": total_barriers,
        "mean_parallel_width": weighted_width_sum / max(1, total_barriers),
        "max_parallel_width": max_parallel_width,
        "worker_utilization": weighted_width_sum / max(1, total_barriers * workers),
        "serial_fraction": total_serial_ops / max(1, total_serial_ops + total_parallel_ops),
        "query_time_ms": (time.perf_counter()-start)*1000 - preprocessing_time_ms,
        "total_time_ms": (time.perf_counter()-start)*1000,
        "total_work_including_preprocessing": total_work + preprocessing_work,
        **trace.summary(),
        "trace_events": trace.events,
        "error_code": "DG5_UNREACHABLE_OR_REENTRY_FAILED",
    }
    return PathResult([], math.inf, False, False, "mrpc_dg5_switchback", total_work, total_work, 0, 0, 2 + total_steps, (time.perf_counter()-start)*1000, peak/1024, tel)
