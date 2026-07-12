from __future__ import annotations
import csv, math, statistics, time
from collections import defaultdict
from pathlib import Path
from broad_eval import grid_graph, random_geometric, clustered, scale_free
from bridge_py.graph import path_distance
from bridge_py.solvers.dijkstra import dijkstra, bidirectional_dijkstra
from bridge_py.solvers.astar import astar
from bridge_py.solvers.mrpc_cg import mrpc_dg5_switchback
from bridge_py.solvers.mrpc_dg6 import mrpc_dg6

def cases():
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

def run_solver(name,G,s,t):
    if name=='dg6': fn=lambda: mrpc_dg6(G,s,t,fallback_exact=False,measure_memory=False)
    elif name=='dg5_clean': fn=lambda: mrpc_dg5_switchback(G,s,t,workers=1,trace_level=0,measure_memory=False,enable_topology_gate=True,dg5_mode='balanced',quality_guard_max_ratio=None,allow_oracle_quality_guard=False)
    elif name=='astar': fn=lambda: astar(G,s,t)
    elif name=='dijkstra': fn=lambda: dijkstra(G,s,t)
    elif name=='bidir': fn=lambda: bidirectional_dijkstra(G,s,t)
    t0=time.perf_counter(); r=fn(); tm=(time.perf_counter()-t0)*1000
    tel=r.telemetry or {}; pd=path_distance(G,r.path) if r.found else math.inf
    valid=(not r.found) or (r.path and r.path[0]==s and r.path[-1]==t and math.isfinite(pd) and abs(pd-r.distance)<=1e-7*max(1,pd))
    work=int(tel.get('total_work_including_preprocessing', int(tel.get('precheck_work',0))+int(tel.get('query_work_units',r.total_work))))
    return r,tm,valid,pd,work

def main():
    rows=[]; solvers=['dijkstra','bidir','astar','dg5_clean','dg6']
    for idx,(topo,nreq,seed,maker) in enumerate(cases()):
        G,s,t=maker(); results={}
        for name in solvers:
            r,tm,valid,pd,work=run_solver(name,G,s,t); results[name]=r
            tel=r.telemetry or {}
            rows.append({'case_id':idx,'topology':topo,'nodes':len(G.adj),'edges':G.edge_count(),'seed':seed,'solver':name,'found':r.found,'valid':valid,'distance':r.distance,'path_distance':pd,'time_ms':tm,'work':work,'steps':r.parallel_steps,'strategy':tel.get('strategy',''),'oracle':tel.get('oracle_used',False),'emergency':tel.get('emergency_path_used',False)})
        exact=results['dijkstra']
        for rr in rows[-len(solvers):]:
            if exact.found and rr['found'] and exact.distance>0: rr['ratio']=rr['distance']/exact.distance
            elif exact.found==rr['found']: rr['ratio']=1.0
            else: rr['ratio']=math.inf
            rr['within10']=math.isfinite(rr['ratio']) and rr['ratio']<=1.1+1e-9
            rr['exact']=math.isfinite(rr['ratio']) and abs(rr['ratio']-1.0)<=1e-9
    out=Path('evaluation_results/dg6_no_mem'); out.mkdir(parents=True,exist_ok=True)
    with (out/'raw.csv').open('w',newline='') as f:
        fields=sorted({k for r in rows for k in r}); w=csv.DictWriter(f,fieldnames=fields); w.writeheader(); w.writerows(rows)
    gs=[]
    for solver in solvers:
        rs=[r for r in rows if r['solver']==solver]; finite=[r['ratio'] for r in rs if math.isfinite(r['ratio'])]
        gs.append({'solver':solver,'cases':len(rs),'found_rate':sum(r['found'] for r in rs)/len(rs),'valid_rate':sum(r['valid'] for r in rs)/len(rs),'exact_rate':sum(r['exact'] for r in rs)/len(rs),'within_10pct_rate':sum(r['within10'] for r in rs)/len(rs),'mean_distance_ratio':statistics.mean(finite),'worst_distance_ratio':max(finite),'mean_work':statistics.mean(r['work'] for r in rs),'mean_time_ms':statistics.mean(r['time_ms'] for r in rs),'median_time_ms':statistics.median(r['time_ms'] for r in rs),'mean_steps':statistics.mean(r['steps'] for r in rs),'oracle_rate':sum(bool(r['oracle']) for r in rs)/len(rs)})
    with (out/'global_summary.csv').open('w',newline='') as f:
        fields=sorted({k for r in gs for k in r}); w=csv.DictWriter(f,fieldnames=fields); w.writeheader(); w.writerows(gs)
    topo=[]
    for (topology,solver),rs in sorted(defaultdict(list,{}).items()): pass
    by=defaultdict(list)
    for r in rows:
        if r['solver']=='dg6': by[r['topology']].append(r)
    trs=[]
    for topo_name,rs in sorted(by.items()):
        finite=[r['ratio'] for r in rs if math.isfinite(r['ratio'])]
        trs.append({'topology':topo_name,'cases':len(rs),'within_10pct_rate':sum(r['within10'] for r in rs)/len(rs),'mean_distance_ratio':statistics.mean(finite),'worst_distance_ratio':max(finite),'mean_work':statistics.mean(r['work'] for r in rs),'mean_time_ms':statistics.mean(r['time_ms'] for r in rs),'median_time_ms':statistics.median(r['time_ms'] for r in rs),'mean_steps':statistics.mean(r['steps'] for r in rs),'emergency_rate':sum(r['emergency'] for r in rs)/len(rs)})
    with (out/'dg6_topology_summary.csv').open('w',newline='') as f:
        fields=sorted({k for r in trs for k in r}); w=csv.DictWriter(f,fieldnames=fields); w.writeheader(); w.writerows(trs)
    print(gs)
if __name__=='__main__': main()
