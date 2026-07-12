from __future__ import annotations

import math
import random
from typing import List, Tuple

from ..graph import Graph, euclidean


def random_geometric_graph(n: int, *, seed: int = 0, k_neighbors: int = 12, noise: float = 0.03) -> Graph:
    """Generate a deterministic k-nearest-neighbor random geometric graph.

    The first reference implementation used an all-pairs distance scan.  This
    version uses scipy.spatial.cKDTree when available, which keeps benchmark
    graph generation from dominating solver evaluation at 1k+ nodes.
    """
    rng = random.Random(seed)
    pos = {i: (rng.random(), rng.random()) for i in range(n)}
    if n <= 1:
        return Graph.from_edges([], directed=False, pos=pos)
    k = max(1, min(int(k_neighbors), n - 1))
    edges = set()
    try:
        from scipy.spatial import cKDTree  # type: ignore
        points = [pos[i] for i in range(n)]
        tree = cKDTree(points)
        distances, indices = tree.query(points, k=k + 1)
        for i in range(n):
            row_d = distances[i]
            row_j = indices[i]
            for d, j in zip(row_d[1:], row_j[1:]):
                u, v = min(i, int(j)), max(i, int(j))
                w = float(d) * (1.0 + rng.uniform(-noise, noise))
                edges.add((u, v, max(w, 1e-12)))
    except Exception:
        # Portable fallback, intentionally correct but slower.
        for i in range(n):
            ds = []
            for j in range(n):
                if i == j:
                    continue
                ds.append((euclidean(pos[i], pos[j]), j))
            ds.sort(key=lambda x: x[0])
            for d, j in ds[:k]:
                u, v = min(i, j), max(i, j)
                w = d * (1.0 + rng.uniform(-noise, noise))
                edges.add((u, v, max(w, 1e-12)))
    return Graph.from_edges(edges, directed=False, pos=pos)


def grid_graph(width: int, height: int, *, diagonal: bool = False) -> Graph:
    pos = {}
    edges = []
    def node(x, y): return y * width + x
    for y in range(height):
        for x in range(width):
            pos[node(x, y)] = (float(x), float(y))
            if x + 1 < width:
                edges.append((node(x, y), node(x + 1, y), 1.0))
            if y + 1 < height:
                edges.append((node(x, y), node(x, y + 1), 1.0))
            if diagonal:
                if x + 1 < width and y + 1 < height:
                    edges.append((node(x, y), node(x + 1, y + 1), math.sqrt(2)))
                if x + 1 < width and y - 1 >= 0:
                    edges.append((node(x, y), node(x + 1, y - 1), math.sqrt(2)))
    return Graph.from_edges(edges, directed=False, pos=pos)


def diagonal_extreme_pair(G: Graph):
    if not G.pos:
        nodes = G.nodes()
        return nodes[0], nodes[-1]
    s = min(G.pos, key=lambda n: G.pos[n][0] + G.pos[n][1])
    t = max(G.pos, key=lambda n: G.pos[n][0] + G.pos[n][1])
    return s, t
