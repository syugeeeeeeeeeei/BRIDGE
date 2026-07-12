import math

from bridge_py import Graph, route
from bridge_py.graphs.generators import diagonal_extreme_pair, random_geometric_graph
from bridge_py.solvers.dijkstra import bidirectional_dijkstra
from bridge_py.solvers.mprc import mprc


def test_exact_route_simple_graph():
    G = Graph.from_edges([("A", "B", 1), ("B", "C", 1), ("A", "C", 3)])
    r = route(G, "A", "C", mode="exact")
    assert r.found
    assert r.path == ["A", "B", "C"]
    assert r.distance == 2


def test_mprc_returns_valid_candidate_on_geometric_graph():
    G = random_geometric_graph(120, seed=7, k_neighbors=10)
    s, t = diagonal_extreme_pair(G)
    exact = bidirectional_dijkstra(G, s, t)
    approx = mprc(G, s, t)
    assert approx.found
    assert approx.distance >= exact.distance - 1e-9
    assert approx.telemetry["k_corridors"] > 0
    assert approx.parallel_steps <= 7


def test_mprc_process_workers_match_sequential_distance():
    G = random_geometric_graph(160, seed=11, k_neighbors=10)
    s, t = diagonal_extreme_pair(G)
    seq = mprc(G, s, t, workers=1)
    par = mprc(G, s, t, workers=2, min_parallel_nodes=0, min_parallel_corridor_nodes=0)
    assert par.found == seq.found
    assert math.isclose(par.distance, seq.distance, rel_tol=1e-12, abs_tol=1e-12)
    assert par.telemetry["workers_requested"] == 2
    assert "process" in par.telemetry["parallel_backend"]
