import math

from bridge_py import Graph, RouteRequest, pier, route
from bridge_py.cable import cable_route
from bridge_py.graph import path_distance
from bridge_py.graphs.generators import diagonal_extreme_pair, random_geometric_graph


def test_pier_identity_and_telemetry():
    g = random_geometric_graph(120, seed=21, k_neighbors=10)
    s, t = diagonal_extreme_pair(g)
    result = pier(g, s, t, target_work_ratio=0.7)
    assert result.solver_name == "pier"
    assert result.telemetry["algorithm_family"] == "PIER"
    assert result.telemetry["legacy_name"] == "MRPC-DG6"
    assert result.found
    assert math.isclose(path_distance(g, result.path), result.distance)


def test_balanced_uses_cable_contract():
    g = Graph.from_edges([("A", "B", 1), ("B", "C", 1), ("A", "C", 3)], pos={"A": (0, 0), "B": (1, 0), "C": (2, 0)})
    result = route(g, "A", "C", mode="balanced", max_suboptimality=1.05)
    assert result.found
    assert result.solver_name == "bridge_balanced"
    assert result.solver_trace
    assert result.telemetry["cable_version"] == "0.1.0"


def test_quality_mode_returns_certified_exact_result_when_needed():
    g = Graph.from_edges([("A", "B", 1), ("B", "C", 1), ("A", "C", 4)], pos={"A": (0, 0), "B": (1, 0), "C": (2, 0)})
    req = RouteRequest("A", "C", mode="quality", max_suboptimality=1.03)
    result = cable_route(g, req)
    assert result.distance == 2
    assert result.quality_certified
    assert result.certified_ratio == 1.0


def test_exact_mode_is_certified():
    g = Graph.from_edges([(0, 1, 1), (1, 2, 1), (0, 2, 5)])
    result = route(g, 0, 2, mode="exact")
    assert result.exact
    assert result.quality_certified
    assert result.lower_bound == result.distance
