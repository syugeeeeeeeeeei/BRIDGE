from __future__ import annotations
import csv, math, statistics
from pathlib import Path
from bridge_py.graphs.generators import random_geometric_graph, diagonal_extreme_pair
from bridge_py.solvers.dijkstra import dijkstra, bidirectional_dijkstra
from bridge_py.solvers.mrpc_cg import mrpc_directional_backtrack_v2, mrpc_directional_backtrack_v3
from bridge_py.types import PathResult
NODE_SIZES=[10,25,50,100,250,500,1000,2000,3000,5000]
TRIALS=3
SEED=20260710
OUT=Path('/mnt/data')

def bidir_p2(exact):
    tel=dict(exact.telemetry); tel.update({'variant':'bidirectional_dijkstra_p2_model','query_work_units':exact.total_work})
    return PathResult(exact.path, exact.distance, exact.found, True, 'bidirectional_dijkstra_p2_model', exact.work_relaxations, exact.work_expanded_nodes, exact.queue_pushes, exact.queue_pops, max(1,math.ceil(exact.parallel_steps/2)), exact.time_ms/2, exact.peak_memory_kib, tel)

def row(n,trial,solver,res,exact_dist,edges):
    ratio=res.distance/exact_dist if res.found and exact_dist>0 and math.isfinite(exact_dist) else math.inf
    t=res.telemetry; qwu=int(t.get('query_work_units',res.total_work) or res.total_work); target=max(1,math.ceil(n/10))
    return {'nodes':n,'trial':trial,'edges':edges,'solver':solver,'found':bool(res.found),
            'distance':res.distance,'exact_distance':exact_dist,'distance_ratio':ratio,
            'within_10pct':bool(res.found and ratio<=1.10),'target_success':bool(res.found and ratio<=1.10),
            'exact_match':bool(res.found and abs(res.distance-exact_dist)<=1e-9*max(1,exact_dist)),
            'query_work_units':qwu,'target_work':target,'target_work_met':qwu<=target,
            'parallel_steps':res.parallel_steps,'time_ms':res.time_ms,'fallback_used':bool(t.get('fallback_used',False)),
            'error_code':t.get('error_code',''),'variant':t.get('variant',''),'workers':t.get('workers_requested','')}

def aggregate(raw):
    groups={}
    for r in raw: groups.setdefault((r['nodes'],r['solver']),[]).append(r)
    summary=[]
    for (n,solver),rs in sorted(groups.items()):
      finite=[float(r['distance_ratio']) for r in rs if math.isfinite(float(r['distance_ratio']))]
      summary.append({'nodes':n,'solver':solver,'trials':len(rs),
        'found_rate':sum(r['found'] for r in rs)/len(rs),
        'exact_rate':sum(r['exact_match'] for r in rs)/len(rs),
        'within_10pct_rate':sum(r['within_10pct'] for r in rs)/len(rs),
        'target_success_rate':sum(r['target_success'] for r in rs)/len(rs),
        'mean_distance_ratio':statistics.fmean(finite) if finite else math.inf,
        'worst_distance_ratio':max(finite) if finite else math.inf,
        'mean_query_work_units':statistics.fmean(float(r['query_work_units']) for r in rs),
        'work_per_node':statistics.fmean(float(r['query_work_units']) for r in rs)/n,
        'target_work_met_rate':sum(r['target_work_met'] for r in rs)/len(rs),
        'mean_parallel_steps':statistics.fmean(float(r['parallel_steps']) for r in rs),
        'mean_time_ms':statistics.fmean(float(r['time_ms']) for r in rs),
        'fallback_rate':sum(r['fallback_used'] for r in rs)/len(rs),
        'unreachable_rate':sum(not r['found'] for r in rs)/len(rs)})
    return summary

def main():
    raw=[]
    for n in NODE_SIZES:
      for trial in range(1,TRIALS+1):
        G=random_geometric_graph(n,seed=SEED+n*100+trial,k_neighbors=12); s,t=diagonal_extreme_pair(G)
        bd=bidirectional_dijkstra(G,s,t); exd=bd.distance; bp=bidir_p2(bd)
        solvers=[('dijkstra',dijkstra(G,s,t)),('bidirectional_dijkstra',bd),('bidirectional_dijkstra_p2_model',bp),
          ('mrpc_dg2_w4', mrpc_directional_backtrack_v2(G,s,t,workers=4,backtrack_width=3,budget_ratio=0.1)),
          ('mrpc_dg3_w1', mrpc_directional_backtrack_v3(G,s,t,workers=1,backtrack_width=3,budget_ratio=0.1)),
          ('mrpc_dg3_w4', mrpc_directional_backtrack_v3(G,s,t,workers=4,backtrack_width=3,budget_ratio=0.1)),
          ('mrpc_dg3_w8', mrpc_directional_backtrack_v3(G,s,t,workers=8,backtrack_width=3,budget_ratio=0.1))]
        for name,res in solvers: raw.append(row(n,trial,name,res,exd,G.edge_count()))
        print('done',n,trial,flush=True)
    raw_path=OUT/'mrpc_dg3_bench_raw.csv'
    with raw_path.open('w',newline='') as f:
      w=csv.DictWriter(f,fieldnames=list(raw[0].keys())); w.writeheader(); w.writerows(raw)
    summary=aggregate(raw)
    sum_path=OUT/'mrpc_dg3_bench_summary.csv'
    with sum_path.open('w',newline='') as f:
      w=csv.DictWriter(f,fieldnames=list(summary[0].keys())); w.writeheader(); w.writerows(summary)
    by={(r['nodes'],r['solver']):r for r in summary}
    ratios=[]
    for n in NODE_SIZES:
      base=by[(n,'bidirectional_dijkstra_p2_model')]
      for solver in ['dijkstra','bidirectional_dijkstra','mrpc_dg2_w4','mrpc_dg3_w1','mrpc_dg3_w4','mrpc_dg3_w8']:
        s=by[(n,solver)]
        ratios.append({'nodes':n,'solver':solver,
          'time_over_bidir_p2':s['mean_time_ms']/base['mean_time_ms'] if base['mean_time_ms'] else math.inf,
          'work_over_bidir_p2':s['mean_query_work_units']/base['mean_query_work_units'] if base['mean_query_work_units'] else math.inf,
          'steps_over_bidir_p2':s['mean_parallel_steps']/base['mean_parallel_steps'] if base['mean_parallel_steps'] else math.inf,
          'found_rate':s['found_rate'],'exact_rate':s['exact_rate'],'within_10pct_rate':s['within_10pct_rate'],
          'target_success_rate':s['target_success_rate'],'mean_distance_ratio':s['mean_distance_ratio'],
          'worst_distance_ratio':s['worst_distance_ratio'],'work_per_node':s['work_per_node'],
          'target_work_met_rate':s['target_work_met_rate'],'unreachable_rate':s['unreachable_rate'],'fallback_rate':s['fallback_rate']})
    rat_path=OUT/'mrpc_dg3_bench_ratios.csv'
    with rat_path.open('w',newline='') as f:
      w=csv.DictWriter(f,fieldnames=list(ratios[0].keys())); w.writeheader(); w.writerows(ratios)
    print(raw_path,sum_path,rat_path)
if __name__=='__main__': main()
