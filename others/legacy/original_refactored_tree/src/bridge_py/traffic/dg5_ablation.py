from __future__ import annotations
import csv, json, math, statistics, time
from pathlib import Path
from collections import defaultdict
from ..graph import Graph
from ..solvers.dijkstra import dijkstra, bidirectional_dijkstra
from ..solvers.astar import astar
from ..solvers.mrpc_cg import mrpc_dg5_switchback, path_distance

def wall_gap(n:int, gap_side:str='top'):
    w=max(20,int(math.sqrt(n))); h=w; x=w//2
    gap_y=1 if gap_side=='top' else h-2
    blocked={(x,y) for y in range(h) if y != gap_y}
    edges=[]; pos={}
    for y in range(h):
      for xx in range(w):
        if (xx,y) in blocked: continue
        u=y*w+xx; pos[u]=(float(xx),float(y))
        for dx,dy in ((1,0),(0,1)):
          q=(xx+dx,y+dy)
          if 0<=q[0]<w and 0<=q[1]<h and q not in blocked:
            edges.append((u,q[1]*w+q[0],1.0))
    G=Graph.from_edges(edges,pos=pos)
    sy=h//2; ty=h//2
    s=sy*w+1; t=ty*w+(w-2)
    return G,s,t

VARIANTS={
 'legacy_current':dict(adaptive_reentry_minimum=False,reentry_candidate_window=0,reentry_max_candidates=1,
                       reentry_min_progress=.08,min_reentry_expanded=8,local_exact_ratio=.20,max_switches=3),
 'dg5_adaptive_window':dict(reentry_min_progress=.08,min_reentry_expanded=8,local_exact_ratio=1.00,max_switches=3),
 'delayed_32':dict(reentry_min_progress=.08,min_reentry_expanded=32,local_exact_ratio=.50,max_switches=3),
 'delayed_128':dict(reentry_min_progress=.08,min_reentry_expanded=128,local_exact_ratio=1.00,max_switches=3),
 'progress_40':dict(reentry_min_progress=.40,min_reentry_expanded=32,local_exact_ratio=1.00,max_switches=3),
 'no_reentry_target':dict(reentry_min_progress=2.0,min_reentry_expanded=10**9,local_exact_ratio=1.50,max_switches=1),
}

def valid(G,r,s,t):
    if not r.found: return False, math.inf
    d=path_distance(G,r.path)
    ok=bool(r.path and r.path[0]==s and r.path[-1]==t and math.isfinite(d) and abs(d-r.distance)<=1e-9)
    return ok,d

def main():
 out=Path('evaluation_results/dg5_ablation'); out.mkdir(parents=True,exist_ok=True)
 rows=[]; traces=[]
 for n in (400,1600,4900):
  for side in ('top','bottom'):
   G,s,t=wall_gap(n,side); exact=dijkstra(G,s,t)
   baselines=[exact,bidirectional_dijkstra(G,s,t),astar(G,s,t)]
   for r in baselines:
    ok,rd=valid(G,r,s,t)
    rows.append(dict(nodes=n,active_nodes=len(G.adj),gap_side=side,variant=r.solver_name,found=r.found,path_valid=ok,distance=r.distance,recomputed_distance=rd,exact_distance=exact.distance,distance_ratio=(rd/exact.distance if ok else math.inf),total_work=r.total_work,total_time_ms=r.time_ms,switch_count=0,reentry_count=0,target_exact_count=0,error_code=(r.telemetry or {}).get('error_code')))
   for name,kw in VARIANTS.items():
    st=time.perf_counter(); r=mrpc_dg5_switchback(G,s,t,workers=1,trace_level=2,trace_sample_every=1,trace_max_events=300000,**kw); wall=(time.perf_counter()-st)*1000
    tel=r.telemetry or {}; ok,rd=valid(G,r,s,t)
    rows.append(dict(nodes=n,active_nodes=len(G.adj),gap_side=side,variant=name,found=r.found,path_valid=ok,distance=r.distance,recomputed_distance=rd,exact_distance=exact.distance,distance_ratio=(rd/exact.distance if ok else math.inf),query_work=r.total_work,preprocessing_work=tel.get('preprocessing_work',0),total_work=tel.get('total_work_including_preprocessing',r.total_work),query_time_ms=tel.get('query_time_ms',r.time_ms),preprocessing_time_ms=tel.get('preprocessing_time_ms',0),total_time_ms=tel.get('total_time_ms',wall),switch_count=tel.get('switch_count',0),reentry_count=tel.get('reentry_count',0),target_exact_count=tel.get('target_exact_count',0),error_code=tel.get('error_code')))
    traces.append({'nodes':n,'gap_side':side,'variant':name,'events':tel.get('trace_events',[])})
 fields=sorted({k for r in rows for k in r})
 with (out/'raw.csv').open('w',newline='') as f:
  w=csv.DictWriter(f,fieldnames=fields); w.writeheader(); w.writerows(rows)
 groups=defaultdict(list)
 for r in rows: groups[r['variant']].append(r)
 summ=[]
 for k,rs in groups.items():
  found=[r for r in rs if r['found'] and r['path_valid']]
  summ.append(dict(variant=k,cases=len(rs),valid_found_rate=len(found)/len(rs),mean_distance_ratio=statistics.mean(r['distance_ratio'] for r in found) if found else math.inf,mean_total_work=statistics.mean(float(r.get('total_work',0)) for r in rs),mean_total_time_ms=statistics.mean(float(r.get('total_time_ms',0)) for r in rs),mean_switches=statistics.mean(float(r.get('switch_count',0)) for r in rs),mean_reentries=statistics.mean(float(r.get('reentry_count',0)) for r in rs),invalid_found_count=sum(bool(r['found']) and not bool(r['path_valid']) for r in rs)))
 sf=sorted({k for r in summ for k in r})
 with (out/'summary.csv').open('w',newline='') as f:
  w=csv.DictWriter(f,fieldnames=sf); w.writeheader(); w.writerows(summ)
 (out/'traces.json').write_text(json.dumps(traces,ensure_ascii=False,indent=2,default=str))
 print(json.dumps(summ,indent=2,ensure_ascii=False))
if __name__=='__main__': main()
