from __future__ import annotations

import math
import sys
from dataclasses import dataclass
from typing import Any, Hashable

from ..graph import Graph, euclidean
from ..types import Node


@dataclass(frozen=True)
class TopologyGateDecision:
    allow_mrpc_fast_path: bool
    force_exact_precheck: bool
    enable_quality_guard: bool
    risk_score: float
    risk_class: str
    reasons: tuple[str, ...]
    profile: dict[str, Any]


def _percentile_index(n: int, p: float) -> int:
    return max(0, min(n - 1, int(math.floor((n - 1) * p))))


def extended_topology_profile(G: Graph, source: Node, target: Node, *, bridge_limit_nodes: int = 20000) -> dict[str, Any]:
    """Deterministic, query-local structural profile for DG5 gating.

    The profile is deliberately conservative: it is meant to decide whether the
    geometry-corridor fast path is likely to be unsafe, not to prove an optimum.
    Work is reported so cold-start comparisons can include this preprocessing.
    """
    nodes = list(G.adj.keys())
    n = len(nodes)
    degrees = [len(G.adj.get(u, [])) for u in nodes]
    edge_entries = sum(degrees)
    directed = bool(getattr(G, "directed", False))
    m = edge_entries if directed else edge_entries // 2
    mean_degree = edge_entries / max(1, n)
    degree_var = sum((d - mean_degree) ** 2 for d in degrees) / max(1, n)
    degree_std = math.sqrt(degree_var)
    degree_cv = degree_std / max(1e-9, mean_degree)
    sorted_degrees = sorted(degrees, reverse=True)
    top_k = max(1, int(math.ceil(n * 0.01))) if n else 1
    top1_share = sum(sorted_degrees[:top_k]) / max(1, edge_entries)
    max_degree = max(degrees, default=0)
    max_mean_ratio = max_degree / max(1e-9, mean_degree)

    profile: dict[str, Any] = {
        "nodes": n,
        "edges": m,
        "directed": directed,
        "mean_degree": mean_degree,
        "degree_std": degree_std,
        "degree_cv": degree_cv,
        "min_degree": min(degrees, default=0),
        "max_degree": max_degree,
        "max_mean_degree_ratio": max_mean_ratio,
        "top1_degree_share": top1_share,
        "source_degree": len(G.adj.get(source, [])),
        "target_degree": len(G.adj.get(target, [])),
        "has_positions": bool(G.pos),
        "profile_work_nodes": n,
        "profile_work_edges": edge_entries,
    }

    if G.pos and source in G.pos and target in G.pos:
        sx, sy = G.pos[source]
        tx, ty = G.pos[target]
        profile["source_target_euclidean"] = math.hypot(tx - sx, ty - sy)

    # Geometry/weight consistency.  Sample all edges once for deterministic small/medium cases.
    ratios: list[float] = []
    geom_lengths: list[float] = []
    if G.pos:
        seen: set[tuple[Hashable, Hashable]] = set()
        for u, nbrs in G.adj.items():
            pu = G.pos.get(u)
            if pu is None:
                continue
            for v, w in nbrs:
                if not directed:
                    key = (u, v) if repr(u) <= repr(v) else (v, u)
                    if key in seen:
                        continue
                    seen.add(key)
                pv = G.pos.get(v)
                if pv is None:
                    continue
                d = euclidean(pu, pv)
                if d > 1e-12 and math.isfinite(w):
                    ratios.append(float(w) / d)
                    geom_lengths.append(d)
        if ratios:
            rmean = sum(ratios) / len(ratios)
            rstd = math.sqrt(sum((r - rmean) ** 2 for r in ratios) / len(ratios))
            ratios_sorted = sorted(ratios)
            profile.update({
                "weight_geo_ratio_mean": rmean,
                "weight_geo_ratio_std": rstd,
                "weight_geo_ratio_cv": rstd / max(1e-9, abs(rmean)),
                "weight_geo_ratio_p05": ratios_sorted[_percentile_index(len(ratios_sorted), 0.05)],
                "weight_geo_ratio_p95": ratios_sorted[_percentile_index(len(ratios_sorted), 0.95)],
                "weight_geo_ratio_samples": len(ratios),
            })
        else:
            profile.update({"weight_geo_ratio_cv": math.inf, "weight_geo_ratio_samples": 0})
        if geom_lengths:
            gl = sorted(geom_lengths)
            gmean = sum(gl) / len(gl)
            gstd = math.sqrt(sum((x - gmean) ** 2 for x in gl) / len(gl))
            gmedian = gl[len(gl)//2]
            profile.update({
                "geometry_edge_length_mean": gmean,
                "geometry_edge_length_cv": gstd / max(1e-9, gmean),
                "geometry_edge_length_p95": gl[_percentile_index(len(gl), 0.95)],
                "geometry_edge_length_p95_median_ratio": gl[_percentile_index(len(gl), 0.95)] / max(1e-9, gmedian),
            })
    else:
        profile.update({"weight_geo_ratio_cv": math.inf, "weight_geo_ratio_samples": 0})

    # Bridge/articulation proxy for portal dependence.  Only for undirected graphs.
    bridge_count = 0
    articulation_count = 0
    component_count = 0
    if (not directed) and n <= bridge_limit_nodes:
        sys.setrecursionlimit(max(sys.getrecursionlimit(), n + 1000))
        index = 0
        disc: dict[Node, int] = {}
        low: dict[Node, int] = {}
        parent: dict[Node, Node | None] = {}
        comp_label: dict[Node, int] = {}
        current_component = 0
        arts: set[Node] = set()

        def dfs(u: Node) -> None:
            nonlocal index, bridge_count, current_component
            comp_label[u] = current_component
            disc[u] = low[u] = index
            index += 1
            child_count = 0
            for v, _ in G.adj.get(u, []):
                if v not in disc:
                    parent[v] = u
                    child_count += 1
                    dfs(v)
                    low[u] = min(low[u], low[v])
                    if low[v] > disc[u]:
                        bridge_count += 1
                    if parent.get(u) is None:
                        if child_count > 1:
                            arts.add(u)
                    elif low[v] >= disc[u]:
                        arts.add(u)
                elif parent.get(u) != v:
                    low[u] = min(low[u], disc[v])

        for u in nodes:
            if u not in disc:
                current_component = component_count
                component_count += 1
                parent[u] = None
                dfs(u)
        articulation_count = len(arts)
        profile["source_component_estimate"] = comp_label.get(source)
        profile["target_component_estimate"] = comp_label.get(target)
        profile["source_target_connected_estimate"] = (comp_label.get(source) is not None and comp_label.get(source) == comp_label.get(target))
    profile.update({
        "component_count_estimate": component_count if n <= bridge_limit_nodes else None,
        "bridge_count": bridge_count,
        "bridge_ratio": bridge_count / max(1, m),
        "articulation_count": articulation_count,
        "articulation_ratio": articulation_count / max(1, n),
    })

    # Source-target corridor occupancy: nodes near the straight segment.
    if G.pos and source in G.pos and target in G.pos and n > 0:
        sx, sy = G.pos[source]
        tx, ty = G.pos[target]
        dx, dy = tx - sx, ty - sy
        seg_len2 = dx * dx + dy * dy
        if seg_len2 > 1e-12:
            # Width scales with bounding box diagonal and node count.
            xs = [p[0] for p in G.pos.values()]
            ys = [p[1] for p in G.pos.values()]
            diag = math.hypot(max(xs)-min(xs), max(ys)-min(ys)) or 1.0
            width = diag / max(8.0, math.sqrt(n))
            in_corridor = 0
            for p in G.pos.values():
                px, py = p
                t = max(0.0, min(1.0, ((px - sx) * dx + (py - sy) * dy) / seg_len2))
                qx, qy = sx + t * dx, sy + t * dy
                if math.hypot(px - qx, py - qy) <= width:
                    in_corridor += 1
            profile["straight_corridor_node_ratio"] = in_corridor / max(1, n)
            profile["straight_corridor_width"] = width
    return profile


def decide_topology_gate(profile: dict[str, Any], *, mode: str = "balanced") -> TopologyGateDecision:
    reasons: list[str] = []
    risk = 0.0

    degree_cv = float(profile.get("degree_cv") or 0.0)
    top1 = float(profile.get("top1_degree_share") or 0.0)
    max_mean = float(profile.get("max_mean_degree_ratio") or 0.0)
    wg_raw = profile.get("weight_geo_ratio_cv")
    wg_cv = float(wg_raw) if wg_raw is not None else math.inf
    bridge_ratio = float(profile.get("bridge_ratio") or 0.0)
    articulation_ratio = float(profile.get("articulation_ratio") or 0.0)
    long_edge_ratio = float(profile.get("geometry_edge_length_p95_median_ratio") or 1.0)
    edge_len_cv = float(profile.get("geometry_edge_length_cv") or 0.0)
    has_pos = bool(profile.get("has_positions"))

    if not has_pos:
        reasons.append("NO_GEOMETRY")
        risk += 0.15
    n = int(profile.get("nodes") or 0)
    if n >= 100 and (degree_cv >= 0.80 or top1 >= 0.055 or max_mean >= 8.0):
        reasons.append("HUB_DOMINANT")
        risk += 0.45
    elif n >= 100 and (degree_cv >= 0.55 or top1 >= 0.035 or max_mean >= 5.0):
        reasons.append("HUB_RISK")
        risk += 0.22

    if math.isinf(wg_cv):
        if has_pos:
            reasons.append("GEOMETRY_WEIGHT_UNMEASURABLE")
            risk += 0.15
    elif wg_cv >= 0.30:
        reasons.append("GEOMETRY_WEIGHT_MISMATCH")
        risk += 0.35
    elif wg_cv >= 0.18:
        reasons.append("GEOMETRY_WEIGHT_NOISY")
        risk += 0.15

    if bridge_ratio >= 0.05 or articulation_ratio >= 0.08 or long_edge_ratio >= 3.0 or edge_len_cv >= 0.9:
        reasons.append("PORTAL_DOMINANT")
        risk += 0.25
    elif bridge_ratio >= 0.02 or articulation_ratio >= 0.04 or long_edge_ratio >= 2.0 or edge_len_cv >= 0.55:
        reasons.append("PORTAL_RISK")
        risk += 0.12

    corridor_ratio = profile.get("straight_corridor_node_ratio")
    if corridor_ratio is not None and float(corridor_ratio) < 0.02:
        reasons.append("LOW_STRAIGHT_CORRIDOR_SUPPORT")
        risk += 0.10

    risk = min(1.0, risk)
    risk_class = "low"
    if risk >= 0.65:
        risk_class = "high"
    elif risk >= 0.35:
        risk_class = "medium"

    # Balanced defaults: use exact early when the fast-path premise is clearly false.
    force_exact = False
    allow_fast = True
    enable_guard = risk >= 0.10
    if mode == "exact":
        force_exact = True
        allow_fast = False
        enable_guard = True
    elif mode == "fast":
        force_exact = False
        allow_fast = True
        enable_guard = risk >= 0.60
    else:
        if "HUB_DOMINANT" in reasons:
            force_exact = True
            allow_fast = False
        elif "GEOMETRY_WEIGHT_MISMATCH" in reasons:
            force_exact = True
            allow_fast = False
        elif "PORTAL_DOMINANT" in reasons and risk >= 0.60:
            force_exact = True
            allow_fast = False
        elif risk >= 0.70:
            force_exact = True
            allow_fast = False

    return TopologyGateDecision(
        allow_mrpc_fast_path=allow_fast,
        force_exact_precheck=force_exact,
        enable_quality_guard=enable_guard,
        risk_score=risk,
        risk_class=risk_class,
        reasons=tuple(reasons) if reasons else ("LOW_RISK",),
        profile=profile,
    )
