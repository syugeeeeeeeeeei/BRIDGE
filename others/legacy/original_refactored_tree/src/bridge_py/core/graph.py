from __future__ import annotations

import math
from dataclasses import dataclass
from typing import Dict, Hashable, Iterable, List, Mapping, MutableMapping, Sequence, Tuple

from .result import Adjacency, Node, Point


@dataclass
class Graph:
    adj: Dict[Node, List[Tuple[Node, float]]]
    pos: Dict[Node, Point] | None = None
    directed: bool = False

    def nodes(self) -> List[Node]:
        return list(self.adj.keys())

    def edge_count(self) -> int:
        m = sum(len(v) for v in self.adj.values())
        return m if self.directed else m // 2

    def reversed(self) -> "Graph":
        rev: Dict[Node, List[Tuple[Node, float]]] = {u: [] for u in self.adj}
        for u, nbrs in self.adj.items():
            rev.setdefault(u, [])
            for v, w in nbrs:
                rev.setdefault(v, []).append((u, w))
        return Graph(rev, self.pos, self.directed)

    @staticmethod
    def from_edges(edges: Iterable[Tuple[Node, Node, float]], directed: bool = False, pos: Dict[Node, Point] | None = None) -> "Graph":
        adj: Dict[Node, List[Tuple[Node, float]]] = {}
        for u, v, w in edges:
            if w < 0:
                raise ValueError("BRIDGE Python reference implementation requires non-negative weights.")
            adj.setdefault(u, []).append((v, float(w)))
            adj.setdefault(v, [])
            if not directed:
                adj[v].append((u, float(w)))
        return Graph(adj=adj, pos=pos, directed=directed)


def euclidean(a: Point, b: Point) -> float:
    return math.hypot(a[0] - b[0], a[1] - b[1])


def normalize_graph(G: Graph | Adjacency) -> Graph:
    if isinstance(G, Graph):
        return G
    return Graph(adj={u: [(v, float(w)) for v, w in nbrs] for u, nbrs in G.items()})


def path_distance(G: Graph, path: Sequence[Node]) -> float:
    if not path:
        return math.inf
    total = 0.0
    for u, v in zip(path, path[1:]):
        for x, w in G.adj.get(u, []):
            if x == v:
                total += w
                break
        else:
            return math.inf
    return total
