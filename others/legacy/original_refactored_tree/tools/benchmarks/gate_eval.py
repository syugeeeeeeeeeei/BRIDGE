from __future__ import annotations
import csv, json, math, random, statistics, time, tracemalloc
from collections import defaultdict
from pathlib import Path
from bridge_py.graph import path_distance
from bridge_py.solvers.dijkstra import dijkstra, bidirectional_dijkstra
from bridge_py.solvers.astar import astar
from bridge_py.solvers.mrpc_cg import mrpc_dg5_switchback
from broad_eval import grid_graph, random_geometric, clustered, scale_free


def run_solver(name,G,s,t):
    if name=='dg5':
        fn=lambda: mrpc_dg5_switchback(G,s,t,workers=1,trace_level=0,measure_memory=False,enable_topology_gate=True,dg5_mode='balanced',quality_guard_max_ratio=1.0)
    elif name=='astar': fn=lambda: astar(G,s,t)
    elif name=='dijkstra': fn=lambda: dijkstra(G,s,t)
    else: fn=lambda: bidirectional_dijkstra(G,s,t)
    tracemalloc.start(); start=time.perf_counter(); r=fn(); elapsed=(time.perf_counter()-start)*1000; _,peak=tracemalloc.get_traced_memory(); tracemalloc.stop()
    tel=r.telemetry or {}; pd=path_distance(G,r.path) if r.found else math.inf
    valid=(not r.found) or (r.path and r.path[0]==s and r.path[-1]==t and math.isfinite(pd) and abs(pd-r.distance)<=1e-7*max(1,pd))
    return r,elapsed,peak/1024,valid,pd,int(tel.get('total_work_including_preprocessing', int(tel.get('preprocessing_work',0)) + int(tel.get('query_work_units',r.total_work))))


def main():
    out=Path('evaluation_results/dg5_gate'); out.mkdir(parents=True,exist_ok=True)
    cases=[]
    for topo in ['open','weighted_noise','wall','double_wall','u_shape','culdesac','spiral','random_obstacles','disconnected']:
        for side in [20,40]:
            for seed in range(3):
                cases.append((topo,side*side,seed,lambda topo=topo,side=side,seed=seed:grid_graph(side,'normal' if topo=='open' else topo,seed,0.8 if topo=='weighted_noise' else 0.0)))
    for n in [400,900]:
        for seed in range(3):
            cases += [('random_geometric',n,seed,lambda n=n,seed=seed:random_geometric(n,seed)),('clustered',n,seed,lambda n=n,seed=seed:clustered(n,seed)),('scale_free_pos',n,seed,lambda n=n,seed=seed:scale_free(n,seed,True)),('scale_free_no_pos',n,seed,lambda n=n,seed=seed:scale_free(n,seed,False))]
    rows=[]
    for idx,(topo,nreq,seed,maker) in enumerate(cases):
        G,s,t=maker(); results={}
        for name in ['dijkstra','bidir','astar','dg5']:
            r,tm,pk,valid,pd,work=run_solver(name,G,s,t); tel=r.telemetry or {}; results[name]=r
            rows.append({'case_id':idx,'topology':topo,'requested_nodes':nreq,'nodes':len(G.adj),'edges':G.edge_count(),'seed':seed,'solver':name,'found':r.found,'valid':valid,'distance':r.distance,'path_distance':pd,'total_work':work,'total_time_ms':tm,'peak_kib_uniform':pk,'steps':r.parallel_steps,'switch_count':tel.get('switch_count',0),'reentry_count':tel.get('reentry_count',0),'fallback_used':tel.get('fallback_used',False),'error_code':tel.get('error_code'),'topology_gate_action':tel.get('topology_gate_action','BASELINE'),'topology_risk_score':tel.get('topology_risk_score'),'topology_risk_class':tel.get('topology_risk_class'),'topology_gate_reasons':';'.join(tel.get('topology_gate_reasons',[])) if isinstance(tel.get('topology_gate_reasons'),list) else tel.get('topology_gate_reasons'),'quality_guard_used':tel.get('quality_guard_used',False),'quality_guard_replaced':tel.get('quality_guard_replaced',False),'quality_guard_work':tel.get('quality_guard_work',0)})
        exact=results['dijkstra']
        for rr in rows[-4:]:
            if exact.found and rr['found'] and exact.distance>0: rr['distance_ratio']=rr['distance']/exact.distance
            elif exact.found==rr['found']: rr['distance_ratio']=1.0
            else: rr['distance_ratio']=math.inf
        if (idx+1)%15==0: print('cases',idx+1,'/',len(cases),flush=True)
    fields=sorted({k for r in rows for k in r})
    with (out/'raw.csv').open('w',newline='') as f: w=csv.DictWriter(f,fieldnames=fields); w.writeheader(); w.writerows(rows)
    groups=defaultdict(list)
    for r in rows: groups[(r['topology'],r['solver'])].append(r)
    summary=[]
    for (topo,solver),rs in sorted(groups.items()):
        finite=[r['distance_ratio'] for r in rs if math.isfinite(r['distance_ratio'])]
        summary.append({'topology':topo,'solver':solver,'cases':len(rs),'found_rate':sum(r['found'] for r in rs)/len(rs),'valid_rate':sum(r['valid'] for r in rs)/len(rs),'mean_distance_ratio':statistics.mean(finite) if finite else math.inf,'worst_distance_ratio':max(finite) if finite else math.inf,'mean_total_work':statistics.mean(r['total_work'] for r in rs),'median_total_work':statistics.median(r['total_work'] for r in rs),'mean_total_time_ms':statistics.mean(r['total_time_ms'] for r in rs),'median_total_time_ms':statistics.median(r['total_time_ms'] for r in rs),'mean_steps':statistics.mean(r['steps'] for r in rs),'switch_rate':sum(float(r['switch_count'] or 0)>0 for r in rs)/len(rs),'reentry_rate':sum(float(r['reentry_count'] or 0)>0 for r in rs)/len(rs),'gate_exact_rate':sum(r.get('topology_gate_action')=='EXACT_PRECHECK' for r in rs)/len(rs),'quality_guard_rate':sum(bool(r.get('quality_guard_used')) for r in rs)/len(rs),'quality_guard_replace_rate':sum(bool(r.get('quality_guard_replaced')) for r in rs)/len(rs)})
    sf=sorted({k for r in summary for k in r})
    with (out/'summary.csv').open('w',newline='') as f: w=csv.DictWriter(f,fieldnames=sf); w.writeheader(); w.writerows(summary)
    # compare dg5 vs best baseline per case
    comp=[]
    for topo,g in rows_by_topology(rows).items():
        vals=[]
        for cid,cg in rows_by_case(g).items():
            dg=next(x for x in cg if x['solver']=='dg5')
            base=[x for x in cg if x['solver']!='dg5' and x['valid'] and x['found']==dg['found']]
            if not base: continue
            vals.append({'time_ratio': dg['total_time_ms']/min(x['total_time_ms'] for x in base), 'work_ratio': dg['total_work']/min(x['total_work'] for x in base)})
        comp.append({'topology':topo,'cases':len(vals),'mean_time_ratio_vs_best_baseline':statistics.mean(v['time_ratio'] for v in vals) if vals else math.inf,'mean_work_ratio_vs_best_baseline':statistics.mean(v['work_ratio'] for v in vals) if vals else math.inf})
    cf=sorted({k for r in comp for k in r})
    with (out/'comparison.csv').open('w',newline='') as f: w=csv.DictWriter(f,fieldnames=cf); w.writeheader(); w.writerows(comp)
    print(json.dumps({'cases':len(cases),'runs':len(rows),'out':str(out)}))

def rows_by_topology(rows):
    d=defaultdict(list)
    for r in rows: d[r['topology']].append(r)
    return d

def rows_by_case(rows):
    d=defaultdict(list)
    for r in rows: d[r['case_id']].append(r)
    return d

if __name__=='__main__': main()
