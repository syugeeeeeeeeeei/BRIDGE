import math

from bridge_py.graphs.generators import grid_graph, diagonal_extreme_pair
from bridge_py.route import route
from bridge_py.graph import path_distance


def test_mrpc_dg6_basic_grid_valid():
    G = grid_graph(12, 12)
    s, t = diagonal_extreme_pair(G)
    r = route(G, s, t, mode="mrpc_dg6")
    assert r.found
    assert r.path[0] == s and r.path[-1] == t
    assert math.isfinite(path_distance(G, r.path))
    assert r.telemetry["variant"] == "mrpc_dg6"
    assert not r.telemetry["exact_source_target_oracle_used"]


def test_mrpc_dg6_disconnected():
    G = grid_graph(8, 8)
    # Remove all edges crossing the middle vertical cut.
    cut_left = {y * 8 + 3 for y in range(8)}
    cut_right = {y * 8 + 4 for y in range(8)}
    for u in list(G.adj):
        G.adj[u] = [(v, w) for v, w in G.adj[u] if not ((u in cut_left and v in cut_right) or (u in cut_right and v in cut_left))]
    r = route(G, 0, 63, mode="mrpc_dg6")
    assert not r.found
    assert r.telemetry["emergency_path_used"]

    checked = route(G, 0, 63, mode="mrpc_dg6", constraints={"enable_component_precheck": True})
    assert not checked.found
    assert checked.exact
    assert checked.telemetry["strategy"] == "component_precheck"
