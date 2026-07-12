from __future__ import annotations
import csv, math, statistics
from pathlib import Path
from bridge_py.graphs.generators import random_geometric_graph, diagonal_extreme_pair
from bridge_py.solvers.dijkstra import dijkstra, bidirectional_dijkstra
from bridge_py.solvers.mprc import mprc
from bridge_py.solvers.mrpc_cg import mrpc_directed_greedy
NODE_SIZES=[10,25,50,100,250,500,1000,2000,3000,5000]
TRIALS=3
SEED=20260710

def row(n, trial, solver, res, exact_dist, edges):
    ratio=res.distance/exact_dist if res.found and math.isfinite(exact_dist) and exact_dist>0 else math.inf
    t=res.telemetry
    qwu=int(t.get('query_work_units',res.total_work) or res.total_work)
    target=max(1,n//10)
    return {'nodes':n,'trial':trial,'edges':edges,'solver':solver,'found':res.found,
            'distance':res.distance,'exact_distance':exact_dist,'distance_ratio':ratio,
            'within_10pct':bool(res.found and 0.9<=ratio<=1.1),
            'exact_match':bool(res.found and abs(res.distance-exact_dist)<=1e-9*max(1,exact_dist)),
            'total_work':res.total_work,'query_work_units':qwu,'target_work':target,'target_work_met':qwu<=target,
            'parallel_steps':res.parallel_steps,'time_ms':res.time_ms,'fallback_used':bool(t.get('fallback_used',False)),
            'error_code':t.get('error_code',''),'variant':t.get('variant',''), 'preprocessing_work':t.get('preprocessing_work','')}

def main():
    raw=[]
    for n in NODE_SIZES:
      for trial in range(1,TRIALS+1):
        G=random_geometric_graph(n, seed=SEED+n*100+trial, k_neighbors=12)
        s,t=diagonal_extreme_pair(G); exact=bidirectional_dijkstra(G,s,t); exd=exact.distance
        solvers=[('dijkstra', dijkstra(G,s,t)),('bidirectional_dijkstra', exact),
                 ('mrpc_greedy_w1', mrpc_directed_greedy(G,s,t,workers=1,exact_fallback=False,backtrack_width=1)),
                 ('mrpc_greedy_w4', mrpc_directed_greedy(G,s,t,workers=4,exact_fallback=False,backtrack_width=1)),
                 ('mrpc_greedy_safe', mrpc_directed_greedy(G,s,t,workers=4,exact_fallback=True,backtrack_width=1))]
        for name,res in solvers: raw.append(row(n,trial,name,res,exd,G.edge_count()))
        print('done',n,trial)
    with Path('/mnt/data/mrpc_target_bench_raw.csv').open('w',newline='') as f:
      w=csv.DictWriter(f,fieldnames=list(raw[0].keys()));w.writeheader();w.writerows(raw)
    groups={}
    for r in raw: groups.setdefault((r['nodes'],r['solver']),[]).append(r)
    summary=[]
    for (n,solver),rs in sorted(groups.items()):
      finite=[float(r['distance_ratio']) for r in rs if math.isfinite(float(r['distance_ratio']))]
      summary.append({'nodes':n,'solver':solver,'trials':len(rs),
        'found_rate':sum(bool(r['found']) for r in rs)/len(rs),
        'exact_rate':sum(bool(r['exact_match']) for r in rs)/len(rs),
        'within_10pct_rate':sum(bool(r['within_10pct']) for r in rs)/len(rs),
        'mean_distance_ratio':statistics.fmean(finite) if finite else math.inf,
        'worst_distance_ratio':max(finite) if finite else math.inf,
        'mean_query_work_units':statistics.fmean(r['query_work_units'] for r in rs),
        'work_per_node':statistics.fmean(r['query_work_units'] for r in rs)/n,
        'target_work_met_rate':sum(bool(r['target_work_met']) for r in rs)/len(rs),
        'mean_parallel_steps':statistics.fmean(r['parallel_steps'] for r in rs),
        'mean_time_ms':statistics.fmean(r['time_ms'] for r in rs),
        'fallback_rate':sum(bool(r['fallback_used']) for r in rs)/len(rs),
        'unreachable_rate':sum(not bool(r['found']) for r in rs)/len(rs)})
    with Path('/mnt/data/mrpc_target_bench_summary.csv').open('w',newline='') as f:
      w=csv.DictWriter(f,fieldnames=list(summary[0].keys()));w.writeheader();w.writerows(summary)
    by={(r['nodes'],r['solver']):r for r in summary}
    ratios=[]
    for n in NODE_SIZES:
      b=by[(n,'bidirectional_dijkstra')]
      for solver in ['mrpc_greedy_w1','mrpc_greedy_w4','mrpc_greedy_safe']:
        s=by[(n,solver)]
        ratios.append({'nodes':n,'solver':solver,
          'time_over_bidir':s['mean_time_ms']/b['mean_time_ms'],
          'work_over_bidir':s['mean_query_work_units']/b['mean_query_work_units'] if b['mean_query_work_units'] else math.inf,
          'steps_over_bidir':s['mean_parallel_steps']/b['mean_parallel_steps'] if b['mean_parallel_steps'] else math.inf,
          'exact_rate':s['exact_rate'],'within_10pct_rate':s['within_10pct_rate'],
          'found_rate':s['found_rate'],'unreachable_rate':s['unreachable_rate'],
          'mean_distance_ratio':s['mean_distance_ratio'],'worst_distance_ratio':s['worst_distance_ratio'],
          'work_per_node':s['work_per_node'],'target_work_met_rate':s['target_work_met_rate'],
          'fallback_rate':s['fallback_rate']})
    with Path('/mnt/data/mrpc_target_bench_ratios.csv').open('w',newline='') as f:
      w=csv.DictWriter(f,fieldnames=list(ratios[0].keys()));w.writeheader();w.writerows(ratios)
if __name__=='__main__': main()
