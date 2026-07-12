from __future__ import annotations

import math
from dataclasses import dataclass
from typing import Optional

from .graph import Graph, euclidean
from .result import Node


@dataclass(frozen=True)
class QualityBounds:
    lower_bound: float = 0.0
    upper_bound: float = math.inf
    certified_ratio: Optional[float] = None
    quality_certified: bool = False
    method: str = "none"


def geometric_lower_bound(G: Graph, source: Node, target: Node, sample_cap: int = 2048) -> tuple[float, str]:
    """Return a conservative admissible geometric lower bound.

    For every sampled edge, weight / Euclidean-length is measured.  The minimum
    observed ratio is only safe when all edges are inspected, therefore bounded
    sampling deliberately returns zero.  This prevents false certification.
    """
    if not G.pos or source not in G.pos or target not in G.pos:
        return 0.0, "none"
    ratios = []
    seen = set()
    total_unique = G.edge_count()
    for u, nbrs in G.adj.items():
        if u not in G.pos:
            continue
        for v, w in nbrs:
            if v not in G.pos:
                continue
            key = (u, v) if G.directed else tuple(sorted((u, v), key=repr))
            if key in seen:
                continue
            seen.add(key)
            length = euclidean(G.pos[u], G.pos[v])
            if length > 1e-12:
                ratios.append(float(w) / length)
            if len(seen) >= sample_cap and total_unique > sample_cap:
                return 0.0, "sampled_geometric_unverified"
    if not ratios:
        return 0.0, "none"
    scale = max(0.0, min(ratios))
    return scale * euclidean(G.pos[source], G.pos[target]), "global_geometric"


def make_bounds(G: Graph, source: Node, target: Node, upper_bound: float, target_ratio: Optional[float] = None) -> QualityBounds:
    lb, method = geometric_lower_bound(G, source, target)
    if not math.isfinite(upper_bound):
        return QualityBounds(lower_bound=lb, upper_bound=upper_bound, method=method)
    ratio = upper_bound / lb if lb > 0 else None
    certified = ratio is not None and target_ratio is not None and ratio <= target_ratio + 1e-12
    return QualityBounds(lb, upper_bound, ratio, certified, method)
