import time, math, statistics
from broad_eval import grid_graph, random_geometric, clustered, scale_free
from bridge_py.solvers.mrpc_dg6 import mrpc_dg6
import bridge_py.solvers.mrpc_dg6 as dg6

def cases():
    cs=[]
    for topo in ['open','weighted_noise','wall','double_wall','u_shape','culdesac','spiral','random_obstacles','disconnected']:
        for side in [20,40]:
            for seed in range(3):
                cs.append(lambda topo=topo, side=side, seed=seed: grid_graph(side, 'normal' if topo=='open' else topo, seed, 0.8 if topo=='weighted_noise' else 0.0))
    for n in [400,900]:
        for seed in range(3):
            cs += [lambda n=n, seed=seed: random_geometric(n, seed), lambda n=n, seed=seed: clustered(n, seed), lambda n=n, seed=seed: scale_free(n, seed, True), lambda n=n, seed=seed: scale_free(n, seed, False)]
    return cs

def run():
    ts=[]
    for maker in cases():
        G,s,t=maker(); t0=time.perf_counter(); mrpc_dg6(G,s,t,fallback_exact=False); ts.append((time.perf_counter()-t0)*1000)
    return statistics.mean(ts), sum(ts)
print('normal',run())
origb, orige = dg6._memory_begin, dg6._memory_end
dg6._memory_begin=lambda: True
dg6._memory_end=lambda already: 0.0
print('no_tracemalloc',run())
