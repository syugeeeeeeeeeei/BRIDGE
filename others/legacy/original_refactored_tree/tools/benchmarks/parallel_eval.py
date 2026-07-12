from __future__ import annotations
import csv, json, math, statistics, time
from collections import defaultdict
from pathlib import Path
from broad_eval import grid_graph, random_geometric, clustered, scale_free
from bridge_py.graph import path_distance
from bridge_py.solvers.dijkstra import dijkstra, bidirectional_dijkstra
from bridge_py.solvers.astar import astar
from bridge_py.solvers.mrpc_cg import mrpc_dg5_switchback

OUT=Path('evaluation_results/dg5_parallel_iteration3'); OUT.mkdir(parents=True,exist_ok=True)

def valid(G,s,t,r):
    if not r.found: return False
    pd=path_distance(G,r.path)
    return bool(r.path and r.path[0]==s and r.path[-1]==t and math.isfinite(pd) and abs(pd-r.distance)<=1e-7*max(1,pd))

def cases():
    for topo in ['open','wall','double_wall','spiral']:
        for side in [20,40,70]:
            for seed in [0,1]:
                yield topo,side*side,seed,lambda topo=topo,side=side,seed=seed:grid_graph(side,'normal' if topo=='open' else topo,seed)
    for n in [400,900,1600]:
        for seed in [0,1]:
            yield 'random_geometric',n,seed,lambda n=n,seed=seed:random_geometric(n,seed)
            yield 'clustered',n,seed,lambda n=n,seed=seed:clustered(n,seed)
            yield 'scale_free',n,seed,lambda n=n,seed=seed:scale_free(n,seed,False)

def main():
    rows=[]
    for cid,(topo,nreq,seed,maker) in enumerate(cases()):
        G,s,t=maker(); exact=dijkstra(G,s,t)
        for name,fn in [('dijkstra',lambda:dijkstra(G,s,t)),('bidir',lambda:bidirectional_dijkstra(G,s,t)),('astar',lambda:astar(G,s,t))]:
            ts=[]; last=None
            for _ in range(3):
                st=time.perf_counter(); last=fn(); ts.append((time.perf_counter()-st)*1000)
            rows.append(dict(case_id=cid,topology=topo,requested_nodes=nreq,nodes=len(G.adj),seed=seed,solver=name,workers=1,found=last.found,valid=valid(G,s,t,last),distance_ratio=(last.distance/exact.distance if last.found and exact.found and exact.distance else (1.0 if last.found==exact.found else math.inf)),total_work=last.total_work,wall_ms=statistics.median(ts),logical_rounds=last.parallel_steps,critical_path_ops=last.parallel_steps,mean_parallel_width=1,max_parallel_width=1,worker_utilization=1,serial_fraction=1,barrier_count=last.parallel_steps))
        for w in [1,2,4,8]:
            ts=[]; last=None
            for _ in range(3):
                st=time.perf_counter(); last=mrpc_dg5_switchback(G,s,t,workers=w,trace_level=0,measure_memory=False); ts.append((time.perf_counter()-st)*1000)
            tel=last.telemetry or {}
            rows.append(dict(case_id=cid,topology=topo,requested_nodes=nreq,nodes=len(G.adj),seed=seed,solver='dg5_bulk',workers=w,found=last.found,valid=valid(G,s,t,last),distance_ratio=(last.distance/exact.distance if last.found and exact.found and exact.distance else (1.0 if last.found==exact.found else math.inf)),total_work=int(tel.get('total_work_including_preprocessing',last.total_work)),wall_ms=statistics.median(ts),logical_rounds=int(tel.get('logical_rounds',last.parallel_steps)),critical_path_ops=int(tel.get('critical_path_ops',last.parallel_steps)),mean_parallel_width=float(tel.get('mean_parallel_width',0)),max_parallel_width=int(tel.get('max_parallel_width',0)),worker_utilization=float(tel.get('worker_utilization',0)),serial_fraction=float(tel.get('serial_fraction',0)),barrier_count=int(tel.get('barrier_count',0)),switch_count=int(tel.get('switch_count',0)),reentry_count=int(tel.get('reentry_count',0)),error_code=tel.get('error_code')))
        print(cid+1,topo,nreq,flush=True)
    fields=sorted({k for r in rows for k in r})
    with (OUT/'raw.csv').open('w',newline='') as f: w=csv.DictWriter(f,fieldnames=fields);w.writeheader();w.writerows(rows)
    groups=defaultdict(list)
    for r in rows: groups[(r['topology'],r['solver'],r['workers'])].append(r)
    summary=[]
    for (topo,solver,w),rs in sorted(groups.items()):
        finite=[r['distance_ratio'] for r in rs if math.isfinite(r['distance_ratio'])]
        summary.append(dict(topology=topo,solver=solver,workers=w,cases=len(rs),found_rate=sum(r['found'] for r in rs)/len(rs),valid_rate=sum(r['valid'] for r in rs)/len(rs),mean_distance_ratio=statistics.mean(finite) if finite else math.inf,worst_distance_ratio=max(finite) if finite else math.inf,mean_work=statistics.mean(r['total_work'] for r in rs),mean_wall_ms=statistics.mean(r['wall_ms'] for r in rs),mean_logical_rounds=statistics.mean(r['logical_rounds'] for r in rs),mean_critical_path_ops=statistics.mean(r['critical_path_ops'] for r in rs),mean_parallel_width=statistics.mean(r['mean_parallel_width'] for r in rs),mean_worker_utilization=statistics.mean(r['worker_utilization'] for r in rs),mean_serial_fraction=statistics.mean(r['serial_fraction'] for r in rs)))
    sf=sorted({k for r in summary for k in r})
    with (OUT/'summary.csv').open('w',newline='') as f: w=csv.DictWriter(f,fieldnames=sf);w.writeheader();w.writerows(summary)
    # worker scaling relative to DG5 w=1 per case
    scaling=[]
    bycase=defaultdict(list)
    for r in rows:
        if r['solver']=='dg5_bulk': bycase[r['case_id']].append(r)
    for cid,rs in bycase.items():
        base=next(r for r in rs if r['workers']==1)
        for r in rs:
            scaling.append(dict(case_id=cid,topology=r['topology'],nodes=r['nodes'],workers=r['workers'],wall_speedup=base['wall_ms']/max(1e-12,r['wall_ms']),logical_speedup=base['logical_rounds']/max(1,r['logical_rounds']),critical_path_speedup=base['critical_path_ops']/max(1,r['critical_path_ops']),efficiency=(base['logical_rounds']/max(1,r['logical_rounds']))/r['workers'],worker_utilization=r['worker_utilization'],serial_fraction=r['serial_fraction'],found=r['found'],valid=r['valid'],distance_ratio=r['distance_ratio']))
    scf=sorted({k for r in scaling for k in r})
    with (OUT/'scaling.csv').open('w',newline='') as f: w=csv.DictWriter(f,fieldnames=scf);w.writeheader();w.writerows(scaling)
    print(json.dumps({'cases':len(set(r['case_id'] for r in rows)),'runs':len(rows),'out':str(OUT)}))
if __name__=='__main__': main()
