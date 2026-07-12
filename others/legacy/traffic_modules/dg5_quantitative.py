from __future__ import annotations
import csv, json, math, statistics, time
from collections import Counter, defaultdict
from pathlib import Path
from typing import Any
from ..graph import Graph, euclidean
from ..solvers.dijkstra import dijkstra, bidirectional_dijkstra
from ..solvers.astar import astar
from ..solvers.mrpc_cg import mrpc_dg5_switchback


def make_grid(n:int, topology:str, seed:int=0):
    w=max(10,int(math.sqrt(n))); h=w
    blocked=set()
    if topology=="wall":
        x=w//2; gap=max(1,h//10)
        for y in range(h-gap): blocked.add((x,y))
    elif topology=="u_shape":
        x0,x1=w//3,2*w//3; y0=h//4; y1=3*h//4
        for y in range(y0,y1): blocked.add((x0,y)); blocked.add((x1,y))
        for x in range(x0,x1+1): blocked.add((x,y1))
    elif topology=="culdesac":
        x0,x1=w//3,2*w//3; y0=h//4; y1=3*h//4
        for y in range(y0,y1): blocked.add((x0,y)); blocked.add((x1,y))
        for x in range(x0,x1+1): blocked.add((x,y0))
    elif topology=="disconnected":
        x=w//2
        for y in range(h): blocked.add((x,y))
    edges=[]; pos={}
    for y in range(h):
        for x in range(w):
            if (x,y) in blocked: continue
            u=y*w+x; pos[u]=(float(x),float(y))
            for dx,dy in ((1,0),(0,1)):
                q=(x+dx,y+dy)
                if 0<=q[0]<w and 0<=q[1]<h and q not in blocked:
                    v=q[1]*w+q[0]; edges.append((u,v,1.0))
    G=Graph.from_edges(edges,pos=pos)
    src=min(pos,key=lambda u:pos[u][0]+pos[u][1])
    tgt=max(pos,key=lambda u:pos[u][0]+pos[u][1])
    return G,src,tgt


def trace_metrics(events, G, exact_path):
    counts=Counter(e.get('event') for e in events)
    expansions=[e for e in events if e.get('event') in ('mrpc_expand','local_exact_expand','segment_expand') and 'node' in e]
    nodes=[e['node'] for e in expansions]
    revisit=1-len(set(map(str,nodes)))/max(1,len(nodes))
    reentries=[e for e in events if e.get('event')=='reentry_accepted']
    segstarts=[e for e in events if e.get('event')=='mrpc_segment_start']
    detours=[e for e in events if e.get('event') in ('detour_detected','mrpc_segment_end') and (e.get('detour') or e.get('detour_reason'))]
    survival=[]
    for r in reentries:
        seq=r['seq']; nxt=min([d['seq'] for d in detours if d['seq']>seq],default=max([e['seq'] for e in events],default=seq))
        survival.append(sum(1 for e in expansions if seq<e['seq']<=nxt and e.get('segment_id',-1)>r.get('segment_id',-1)))
    switch_nodes=[e.get('node') for e in events if e.get('event') in ('reentry_accepted','detour_detected') and e.get('node') in (G.pos or {})]
    switch_dist=[]
    for a,b in zip(switch_nodes,switch_nodes[1:]): switch_dist.append(euclidean(G.pos[a],G.pos[b]))
    exact_set=set(exact_path)
    overlap=sum(1 for n in nodes if n in exact_set)/max(1,len(nodes))
    hs=[float(e['h']) for e in expansions if isinstance(e.get('h'),(int,float))]
    h_reversals=sum(1 for a,b in zip(hs,hs[1:]) if b>a+1e-12)
    return {
      'trace_events':len(events),'frontier_revisit_rate':revisit,'exact_path_overlap_rate':overlap,
      'reentry_count_trace':len(reentries),'reentry_survival_mean':statistics.mean(survival) if survival else 0,
      'reentry_survival_min':min(survival) if survival else 0,'switch_distance_mean':statistics.mean(switch_dist) if switch_dist else 0,
      'heuristic_reversal_rate':h_reversals/max(1,len(hs)-1),'event_counts':dict(counts)
    }


def run_case(topology,n,seed,trace_level=2):
    G,s,t=make_grid(n,topology,seed)
    t0=time.perf_counter(); dg=mrpc_dg5_switchback(G,s,t,workers=1,component_index=None,enable_component_precheck=True,trace_level=trace_level,trace_sample_every=1,trace_max_events=200000); dg_wall=(time.perf_counter()-t0)*1000
    bas=[dijkstra(G,s,t),bidirectional_dijkstra(G,s,t),astar(G,s,t)]
    exact=bas[0]
    rows=[]
    for r in [dg]+bas:
        tel=r.telemetry or {}
        total_time=float(tel.get('total_time_ms',r.time_ms))
        total_work=int(tel.get('total_work_including_preprocessing',r.total_work))
        rows.append({'topology':topology,'requested_nodes':n,'nodes':len(G.adj),'seed':seed,'solver':r.solver_name,'found':r.found,'distance':r.distance,'distance_ratio':(r.distance/exact.distance if r.found and exact.found and exact.distance else (1.0 if r.found==exact.found else math.inf)),'query_work':r.total_work,'preprocessing_work':int(tel.get('preprocessing_work',0)),'total_work':total_work,'query_time_ms':float(tel.get('query_time_ms',r.time_ms)),'preprocessing_time_ms':float(tel.get('preprocessing_time_ms',0)),'total_time_ms':total_time,'parallel_steps':r.parallel_steps,'peak_memory_kib':r.peak_memory_kib,'switch_count':int(tel.get('switch_count',0)),'reentry_count':int(tel.get('reentry_count',0)),'fallback_used':bool(tel.get('fallback_used',False)),'error_code':tel.get('error_code')})
    tm=trace_metrics(dg.telemetry.get('trace_events',[]),G,exact.path)
    rows[0].update({k:v for k,v in tm.items() if k!='event_counts'})
    return rows, {'case':{'topology':topology,'n':n,'seed':seed},'dg5':rows[0],'event_counts':tm['event_counts'],'events':dg.telemetry.get('trace_events',[])}


def main():
    out=Path('evaluation_results/dg5_quantitative'); out.mkdir(parents=True,exist_ok=True)
    allrows=[]; traces=[]
    for topology in ['normal','wall','u_shape','culdesac','disconnected']:
      for n in [400,900,1600]:
       for seed in range(3):
        rows,tr=run_case(topology,n,seed); allrows.extend(rows); traces.append(tr)
    fields=sorted({k for r in allrows for k in r})
    with (out/'raw.csv').open('w',newline='',encoding='utf-8') as f:
      w=csv.DictWriter(f,fieldnames=fields); w.writeheader(); w.writerows(allrows)
    (out/'traces.json').write_text(json.dumps(traces,ensure_ascii=False,indent=2,default=str),encoding='utf-8')
    groups=defaultdict(list)
    for r in allrows: groups[(r['topology'],r['requested_nodes'],r['solver'])].append(r)
    summary=[]
    for key,rs in groups.items():
      summary.append({'topology':key[0],'requested_nodes':key[1],'solver':key[2],'trials':len(rs),'found_rate':sum(r['found'] for r in rs)/len(rs),'mean_distance_ratio':statistics.mean(r['distance_ratio'] for r in rs if math.isfinite(r['distance_ratio'])) if any(math.isfinite(r['distance_ratio']) for r in rs) else math.inf,'mean_total_work':statistics.mean(r['total_work'] for r in rs),'mean_total_time_ms':statistics.mean(r['total_time_ms'] for r in rs),'mean_steps':statistics.mean(r['parallel_steps'] for r in rs),'mean_reentry_count':statistics.mean(r.get('reentry_count',0) for r in rs),'mean_reentry_survival':statistics.mean(r.get('reentry_survival_mean',0) for r in rs),'mean_revisit_rate':statistics.mean(r.get('frontier_revisit_rate',0) for r in rs),'mean_exact_overlap':statistics.mean(r.get('exact_path_overlap_rate',0) for r in rs),'mean_h_reversal_rate':statistics.mean(r.get('heuristic_reversal_rate',0) for r in rs)})
    sf=sorted({k for r in summary for k in r})
    with (out/'summary.csv').open('w',newline='',encoding='utf-8') as f:
      w=csv.DictWriter(f,fieldnames=sf); w.writeheader(); w.writerows(summary)
    print(json.dumps({'rows':len(allrows),'cases':len(traces),'out':str(out)},ensure_ascii=False))
if __name__=='__main__': main()
