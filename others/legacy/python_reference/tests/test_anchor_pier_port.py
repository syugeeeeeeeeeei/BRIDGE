from __future__ import annotations

import math

from bridge_py import Graph, route
from bridge_py.anchor.algorithm import run_anchor_algorithm


def grid(width: int, height: int) -> Graph:
    edges = []
    pos = {}
    for y in range(height):
        for x in range(width):
            node = y * width + x
            pos[node] = (float(x), float(y))
            if x:
                edges.append((node, node - 1, 1.0))
            if y:
                edges.append((node, node - width, 1.0))
    return Graph.from_edges(edges, pos=pos)


def test_anchor_algorithm_returns_valid_grid_path() -> None:
    graph = grid(12, 12)
    result = run_anchor_algorithm(graph, 0, 143, strategy="geometric_corridor")
    assert result.found
    assert result.path[0] == 0
    assert result.path[-1] == 143
    assert math.isclose(result.distance, 22.0)
    assert result.telemetry["strategy"] in {
        "grid_detour", "geometric_corridor", "portal", "weighted_cost", "hub_aware"
    }


def test_gate_uses_anchor_for_fast_mode() -> None:
    graph = grid(10, 10)
    result = route(graph, 0, 99, mode="fast")
    assert result.found
    assert result.telemetry["solver_trace"][0]["solver"] == "anchor"
    assert result.telemetry["responsibility_split"]["primary_search"] == "ANCHOR"


def test_missing_endpoint_is_safe() -> None:
    graph = grid(4, 4)
    result = run_anchor_algorithm(graph, -1, 15)
    assert not result.found
    assert math.isinf(result.distance)
