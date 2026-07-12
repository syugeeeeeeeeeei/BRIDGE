from __future__ import annotations
import csv, json, math, statistics, time, tracemalloc
from collections import defaultdict
from pathlib import Path
from bridge_py.graph import path_distance
from bridge_py.solvers.mrpc_cg import mrpc_dg5_switchback
from broad_eval import grid_graph, random_geometric, clustered, scale_free


def run_variant(var,G,s,t):
    if var=='base':
        fn=lambda: mrpc_dg5_switchback(G,s,t,workers=1,trace_level=0,measure_memory=False,enable_topology_gate=False,quality_guard_max_ratio=None)
    else:
        fn=lambda: mrpc_dg5_switchback(G,s,t,workers=1,trace_level=0,measure_memory=False,enable_topology_gate=True,dg5_mode='balanced',quality_guard_max_ratio=1.0)
    tracemalloc.start(); st=time.perf_counter(); r=fn(); tm=(time.perf_counter()-st)*1000; _,pk=tracemalloc.get_traced_memory(); tracemalloc.stop()
    tel=r.telemetry or {}; pd=path_distance(G,r.path) if r.found else math.inf
    valid=(not r.found) or (r.path and r.path[0]==s and r.path[-1]==t and math.isfinite(pd) and abs(pd-r.distance)<=1e-7*max(1,pd))
    return r,tm,valid,pd,int(tel.get('total_work_including_preprocessing',int(tel.get('preprocessing_work',0))+int(tel.get('query_work_units',r.total_work))))


def cases():
    out=[]
    for topo in ['open','weighted_noise','wall','double_wall','u_shape','culdesac','spiral','random_obstacles','disconnected']:
        for side in [20,40]:
            for seed in range(3): out.append((topo,side*side,seed,lambda topo=topo,side=side,seed=seed:grid_graph(side,'normal' if topo=='open' else topo,seed,0.8 if topo=='weighted_noise' else 0.0)))
    for n in [400,900]:
        for seed in range(3):
            out += [('random_geometric',n,seed,lambda n=n,seed=seed:random_geometric(n,seed)),('clustered',n,seed,lambda n=n,seed=seed:clustered(n,seed)),('scale_free_pos',n,seed,lambda n=n,seed=seed:scale_free(n,seed,True)),('scale_free_no_pos',n,seed,lambda n=n,seed=seed:scale_free(n,seed,False))]
    return out


def main():
    out=Path('evaluation_results/dg5_gate_ab'); out.mkdir(parents=True,exist_ok=True)
    rows=[]
    for idx,(topo,nreq,seed,maker) in enumerate(cases()):
        G,s,t=maker()
        exact=run_variant('base',G,s,t)[0]  # only for exact distance? no, use dijkstra? too slow import avoid, use base ratios not exact
        # use dijkstra from broad? just skip; compare base vs gate path distances directly relative to base if both found
        for var in ['base','gate']:
            r,tm,valid,pd,work=run_variant(var,G,s,t); tel=r.telemetry or {}
            rows.append({'case_id':idx,'topology':topo,'requested_nodes':nreq,'seed':seed,'variant':var,'found':r.found,'valid':valid,'distance':r.distance,'path_distance':pd,'total_work':work,'total_time_ms':tm,'switch_count':tel.get('switch_count',0),'reentry_count':tel.get('reentry_count',0),'gate_action':tel.get('topology_gate_action'),'gate_reasons':';'.join(tel.get('topology_gate_reasons',[])) if isinstance(tel.get('topology_gate_reasons'),list) else tel.get('topology_gate_reasons'),'quality_guard_used':tel.get('quality_guard_used',False),'quality_guard_replaced':tel.get('quality_guard_replaced',False)})
        if (idx+1)%20==0: print('cases',idx+1,flush=True)
    fields=sorted({k for r in rows for k in r})
    with (out/'raw.csv').open('w',newline='') as f: w=csv.DictWriter(f,fieldnames=fields); w.writeheader(); w.writerows(rows)
    groups=defaultdict(list)
    for r in rows: groups[(r['topology'],r['variant'])].append(r)
    summary=[]
    for (topo,var),rs in sorted(groups.items()):
        summary.append({'topology':topo,'variant':var,'cases':len(rs),'found_rate':sum(r['found'] for r in rs)/len(rs),'valid_rate':sum(r['valid'] for r in rs)/len(rs),'mean_distance':statistics.mean(r['distance'] for r in rs if math.isfinite(r['distance'])) if any(math.isfinite(r['distance']) for r in rs) else math.inf,'mean_work':statistics.mean(r['total_work'] for r in rs),'mean_time_ms':statistics.mean(r['total_time_ms'] for r in rs),'quality_guard_rate':sum(bool(r['quality_guard_used']) for r in rs)/len(rs),'quality_guard_replace_rate':sum(bool(r['quality_guard_replaced']) for r in rs)/len(rs),'gate_exact_rate':sum(r.get('gate_action')=='EXACT_PRECHECK' for r in rs)/len(rs)})
    sf=sorted({k for r in summary for k in r})
    with (out/'summary.csv').open('w',newline='') as f: w=csv.DictWriter(f,fieldnames=sf); w.writeheader(); w.writerows(summary)
    print(json.dumps({'cases':len(cases()),'runs':len(rows),'out':str(out)}))
if __name__=='__main__': main()
