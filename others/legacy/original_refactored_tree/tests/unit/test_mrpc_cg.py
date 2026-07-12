from bridge_py.graphs.generators import diagonal_extreme_pair, random_geometric_graph
from bridge_py.solvers.dijkstra import bidirectional_dijkstra
from bridge_py.solvers.mrpc_cg import build_compressed_graph, mrpc_cg


def test_compressed_graph_has_witness_edges():
    G = random_geometric_graph(80, seed=42, k_neighbors=8)
    CG = build_compressed_graph(G)
    assert CG.supernodes
    for u, nbrs in CG.adj.items():
        for v, _, e in nbrs:
            assert e.witness_u in G.adj
            assert any(x == e.witness_v for x, _ in G.adj[e.witness_u])


def test_mrpc_cg_safe_finds_reference_path_when_guarded():
    G = random_geometric_graph(120, seed=7, k_neighbors=10)
    s, t = diagonal_extreme_pair(G)
    exact = bidirectional_dijkstra(G, s, t)
    r = mrpc_cg(G, s, t, max_distance_ratio=1.02, exact_fallback=True, workers=4)
    assert r.found
    assert r.distance <= exact.distance * 1.02
