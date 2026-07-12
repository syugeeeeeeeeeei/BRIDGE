from __future__ import annotations
import time, math, csv, statistics, json
from collections import defaultdict
from pathlib import Path

from broad_eval import grid_graph, random_geometric, clustered, scale_free
from bridge_py.graph import path_distance
from bridge_py.solvers.dijkstra import dijkstra
import bridge_py.solvers.mrpc_dg6 as dg6

# profile wrappers
stats = defaultdict(lambda: {'calls':0,'time_ms':0.0})
orig = {}

def wrap(name):
    fn=getattr(dg6,name)
    orig[name]=fn
    def w(*args, **kwargs):
        t0=time.perf_counter()
        try:
            return fn(*args, **kwargs)
        finally:
            dt=(time.perf_counter()-t0)*1000
            stats[name]['calls'] += 1
            stats[name]['time_ms'] += dt
    setattr(dg6,name,w)

for name in [
    '_memory_begin','_memory_end','_component_reachable','_graph_features','_choose_strategy',
    '_corridor_nodes','_expand_by_hops','_budgeted_dijkstra','_candidate_result',
    '_greedy_geometric_path','_weighted_astar','_beam_astar_path','_budgeted_bidirectional_dijkstra',
    '_local_repair','_geometric_corridor_strategy','_long_edge_portal_endpoints','_portal_strategy',
    '_hub_aware_strategy','_weighted_cost_strategy','path_distance'
]:
    if hasattr(dg6,name): wrap(name)


def make_cases():
    cs=[]
    for topo in ['open','weighted_noise','wall','double_wall','u_shape','culdesac','spiral','random_obstacles','disconnected']:
        for side in [20,40]:
            for seed in range(3):
                cs.append((topo, side*side, seed, lambda topo=topo, side=side, seed=seed: grid_graph(side, 'normal' if topo=='open' else topo, seed, 0.8 if topo=='weighted_noise' else 0.0)))
    for n in [400,900]:
        for seed in range(3):
            cs += [
                ('random_geometric', n, seed, lambda n=n, seed=seed: random_geometric(n, seed)),
                ('clustered', n, seed, lambda n=n, seed=seed: clustered(n, seed)),
                ('scale_free_pos', n, seed, lambda n=n, seed=seed: scale_free(n, seed, True)),
                ('scale_free_no_pos', n, seed, lambda n=n, seed=seed: scale_free(n, seed, False)),
            ]
    return cs

rows=[]
for idx,(topo,nreq,seed,maker) in enumerate(make_cases()):
    G,s,t=maker()
    exact=dijkstra(G,s,t)
    before={k:v.copy() for k,v in stats.items()}
    t0=time.perf_counter()
    r=dg6.mrpc_dg6(G,s,t,fallback_exact=False)
    elapsed=(time.perf_counter()-t0)*1000
    pd=path_distance(G,r.path) if r.found else math.inf
    ratio = (r.distance/exact.distance) if exact.found and r.found and exact.distance>0 else (1.0 if exact.found==r.found else math.inf)
    tel=r.telemetry or {}
    # per-case function deltas
    deltas={}
    for k,v in stats.items():
        b=before.get(k, {'calls':0,'time_ms':0.0})
        if v['calls']!=b['calls'] or abs(v['time_ms']-b['time_ms'])>1e-9:
            deltas[k+'_calls']=v['calls']-b['calls']
            deltas[k+'_ms']=v['time_ms']-b['time_ms']
    rows.append({
        'case_id':idx,'topology':topo,'nodes':len(G.adj),'edges':G.edge_count(),'seed':seed,
        'elapsed_ms':elapsed,'result_time_ms':r.time_ms,'found':r.found,'ratio':ratio,'within10':math.isfinite(ratio) and ratio<=1.1,
        'work':tel.get('total_work_including_preprocessing',r.total_work),'strategy':tel.get('strategy'),
        'emergency':tel.get('emergency_path_used',False),'repair':tel.get('repair_triggered',False),'quality_budget_used':tel.get('quality_budget_used',0),
        'first_path_work':tel.get('first_path_work',''),'target_work':tel.get('target_work',''), **deltas
    })

out=Path('evaluation_results/dg6_time_profile'); out.mkdir(parents=True, exist_ok=True)
fields=sorted({k for r in rows for k in r})
with (out/'raw_profile.csv').open('w', newline='') as f:
    w=csv.DictWriter(f, fieldnames=fields); w.writeheader(); w.writerows(rows)
# totals table
sum_rows=[]
total_elapsed=sum(r['elapsed_ms'] for r in rows)
for name,v in sorted(stats.items(), key=lambda kv: kv[1]['time_ms'], reverse=True):
    sum_rows.append({'function':name,'calls':v['calls'],'total_ms':v['time_ms'],'mean_ms':v['time_ms']/v['calls'] if v['calls'] else 0,'pct_of_elapsed':100*v['time_ms']/total_elapsed if total_elapsed else 0})
with (out/'function_totals.csv').open('w', newline='') as f:
    w=csv.DictWriter(f, fieldnames=['function','calls','total_ms','mean_ms','pct_of_elapsed']); w.writeheader(); w.writerows(sum_rows)
# topology summary
keys=['elapsed_ms','_graph_features_ms','_corridor_nodes_ms','_expand_by_hops_ms','_budgeted_dijkstra_ms','_weighted_astar_ms','_beam_astar_path_ms','_budgeted_bidirectional_dijkstra_ms','_local_repair_ms','_memory_begin_ms','_memory_end_ms']
by=defaultdict(list)
for r in rows: by[r['topology']].append(r)
toprows=[]
for topo,rs in sorted(by.items()):
    tr={'topology':topo,'cases':len(rs),'mean_elapsed_ms':statistics.mean(r['elapsed_ms'] for r in rs),'mean_work':statistics.mean(float(r['work']) for r in rs),'mean_ratio':statistics.mean(r['ratio'] for r in rs if math.isfinite(r['ratio']))}
    for k in keys[1:]:
        tr['mean'+k]=statistics.mean(float(r.get(k,0) or 0) for r in rs)
    toprows.append(tr)
with (out/'topology_profile.csv').open('w', newline='') as f:
    fields=sorted({k for r in toprows for k in r}); w=csv.DictWriter(f,fieldnames=fields); w.writeheader(); w.writerows(toprows)
print(json.dumps({'runs':len(rows),'total_elapsed_ms':total_elapsed,'out':str(out),'top_functions':sum_rows[:10]},indent=2))
