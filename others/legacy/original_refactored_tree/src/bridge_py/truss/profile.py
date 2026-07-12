from __future__ import annotations

from dataclasses import dataclass

from ..core.graph import Graph
from ..core.result import Node


@dataclass(frozen=True, slots=True)
class QueryProfile:
    nodes: int
    edges: int
    has_positions: bool
    mean_degree: float
    source_degree: int
    target_degree: int


def profile_query(graph: Graph, source: Node, target: Node) -> QueryProfile:
    node_count = len(graph.adj)
    degree_sum = sum(len(neighbors) for neighbors in graph.adj.values())
    return QueryProfile(
        nodes=node_count,
        edges=graph.edge_count(),
        has_positions=graph.pos is not None,
        mean_degree=degree_sum / max(1, node_count),
        source_degree=len(graph.adj.get(source, ())),
        target_degree=len(graph.adj.get(target, ())),
    )
