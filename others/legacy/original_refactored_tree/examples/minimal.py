from bridge_py import route
from bridge_py.graphs.generators import random_geometric_graph, diagonal_extreme_pair

G = random_geometric_graph(250, seed=42, k_neighbors=12)
s, t = diagonal_extreme_pair(G)

exact = route(G, s, t, mode="exact")
fast = route(G, s, t, mode="fast")

print("exact", exact.distance, exact.path[:5], "...")
print("mprc ", fast.distance, fast.telemetry.get("best_corridor_id"))
print("distance ratio", fast.distance / exact.distance)
print("steps", {"exact": exact.parallel_steps, "mprc": fast.parallel_steps})
