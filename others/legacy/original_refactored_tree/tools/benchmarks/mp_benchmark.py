from __future__ import annotations
import time, math, statistics
from concurrent.futures import ProcessPoolExecutor
from broad_eval import grid_graph, random_geometric, clustered, scale_free
from bridge_py.solvers.mrpc_dg6 import mrpc_dg6

def case_specs():
    specs=[]
    for topo in ['open','weighted_noise','wall','double_wall','u_shape','culdesac','spiral','random_obstacles','disconnected']:
        for side in [20,40]:
            for seed in range(3): specs.append(('grid', topo, side, seed))
    for n in [400,900]:
        for seed in range(3):
            specs += [('random_geometric','',n,seed),('clustered','',n,seed),('scale_free_pos','',n,seed),('scale_free_no_pos','',n,seed)]
    return specs

def make(spec):
    kind, topo, n, seed=spec
    if kind=='grid': return grid_graph(n, 'normal' if topo=='open' else topo, seed, 0.8 if topo=='weighted_noise' else 0.0)
    if kind=='random_geometric': return random_geometric(n, seed)
    if kind=='clustered': return clustered(n, seed)
    if kind=='scale_free_pos': return scale_free(n, seed, True)
    if kind=='scale_free_no_pos': return scale_free(n, seed, False)
    raise ValueError(spec)

def solve(spec):
    G,s,t=make(spec)
    r=mrpc_dg6(G,s,t,fallback_exact=False,measure_memory=False)
    return r.found, r.distance, r.time_ms, (r.telemetry or {}).get('total_work_including_preprocessing',r.total_work)

def run_seq(specs):
    t0=time.perf_counter(); out=[solve(s) for s in specs]; return (time.perf_counter()-t0)*1000,out

def run_mp(specs, workers):
    t0=time.perf_counter()
    with ProcessPoolExecutor(max_workers=workers) as ex:
        out=list(ex.map(solve,specs,chunksize=3))
    return (time.perf_counter()-t0)*1000,out

def main():
    specs=case_specs()
    seq,_=run_seq(specs)
    print('sequential_ms',seq)
    for w in [2,4,8]:
        tm,_=run_mp(specs,w)
        print('workers',w,'ms',tm,'speedup',seq/tm)
if __name__=='__main__': main()
