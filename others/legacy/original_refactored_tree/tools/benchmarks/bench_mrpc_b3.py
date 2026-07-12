from __future__ import annotations
import csv, math, statistics, time
from pathlib import Path
from bridge_py.graphs.generators import random_geometric_graph, diagonal_extreme_pair
from bridge_py.solvers.dijkstra import dijkstra, bidirectional_dijkstra
from bridge_py.solvers.mrpc_cg import mrpc_bidirectional_beam, mrpc_directed_greedy

NODE_SIZES=[10,25,50,100,250,500,1000,2000,3000,5000]
TRIALS=3
SEED=20260710
OUT=Path('/mnt/data')

def make_bidir_p2(exact):
    # Algorithmic 2-lane model: bidirectional search has two independent frontiers.
    # We keep work identical, but halve logical depth and use idealized wall time.
    from bridge_py.types import PathResult
    tel=dict(exact.telemetry)
    tel.update({'variant':'bidirectional_dijkstra_p2_model','parallel_model':'ideal_two_frontiers','query_work_units':exact.total_work})
    return PathResult(exact.path, exact.distance, exact.found, exact.exact, 'bidirectional_dijkstra_p2_model',
                      exact.work_relaxations, exact.work_expanded_nodes, exact.queue_pushes, exact.queue_pops,
                      max(1, math.ceil(exact.parallel_steps/2)), exact.time_ms/2, exact.peak_memory_kib, tel)

def row(n, trial, solver, res, exact_dist, edges):
    ratio=res.distance/exact_dist if res.found and math.isfinite(exact_dist) and exact_dist>0 else math.inf
    t=res.telemetry
    qwu=int(t.get('query_work_units',res.total_work) or res.total_work)
    target=max(1, math.ceil(n/10))
    return {'nodes':n,'trial':trial,'edges':edges,'solver':solver,'found':bool(res.found),
            'distance':res.distance,'exact_distance':exact_dist,'distance_ratio':ratio,
            'within_10pct':bool(res.found and 0.9<=ratio<=1.1),
            'exact_match':bool(res.found and abs(res.distance-exact_dist)<=1e-9*max(1,exact_dist)),
            'total_work':res.total_work,'query_work_units':qwu,'target_work':target,'target_work_met':qwu<=target,
            'parallel_steps':res.parallel_steps,'time_ms':res.time_ms,'fallback_used':bool(t.get('fallback_used',False)),
            'error_code':t.get('error_code',''),'variant':t.get('variant',''),
            'workers':t.get('workers_requested',''),'raw_relaxations':t.get('raw_relaxations',''),
            'beam_rounds':t.get('beam_rounds','')}

def aggregate(raw):
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
        'mean_query_work_units':statistics.fmean(float(r['query_work_units']) for r in rs),
        'work_per_node':statistics.fmean(float(r['query_work_units']) for r in rs)/n,
        'target_work_met_rate':sum(bool(r['target_work_met']) for r in rs)/len(rs),
        'mean_parallel_steps':statistics.fmean(float(r['parallel_steps']) for r in rs),
        'mean_time_ms':statistics.fmean(float(r['time_ms']) for r in rs),
        'fallback_rate':sum(bool(r['fallback_used']) for r in rs)/len(rs),
        'unreachable_rate':sum(not bool(r['found']) for r in rs)/len(rs)})
    return summary

def main():
    raw=[]
    for n in NODE_SIZES:
      for trial in range(1,TRIALS+1):
        G=random_geometric_graph(n, seed=SEED+n*100+trial, k_neighbors=12)
        s,t=diagonal_extreme_pair(G)
        exact=bidirectional_dijkstra(G,s,t); exd=exact.distance
        solvers=[
          ('dijkstra', dijkstra(G,s,t)),
          ('bidirectional_dijkstra', exact),
          ('bidirectional_dijkstra_p2_model', make_bidir_p2(exact)),
          ('mrpc_dg_old_w4', mrpc_directed_greedy(G,s,t,workers=4,exact_fallback=False,backtrack_width=1,budget_ratio=0.1)),
          ('mrpc_b3_w1', mrpc_bidirectional_beam(G,s,t,workers=1,exact_fallback=False,budget_ratio=0.1,branch_cap=10)),
          ('mrpc_b3_w4', mrpc_bidirectional_beam(G,s,t,workers=4,exact_fallback=False,budget_ratio=0.1,branch_cap=10)),
          ('mrpc_b3_w8', mrpc_bidirectional_beam(G,s,t,workers=8,exact_fallback=False,budget_ratio=0.1,branch_cap=10)),
          ('mrpc_b3_safe', mrpc_bidirectional_beam(G,s,t,workers=4,exact_fallback=True,budget_ratio=0.1,branch_cap=10)),
        ]
        for name,res in solvers: raw.append(row(n,trial,name,res,exd,G.edge_count()))
        print('done',n,trial, flush=True)
    raw_path=OUT/'mrpc_b3_bench_raw.csv'
    with raw_path.open('w',newline='') as f:
      w=csv.DictWriter(f,fieldnames=list(raw[0].keys()));w.writeheader();w.writerows(raw)
    summary=aggregate(raw)
    sum_path=OUT/'mrpc_b3_bench_summary.csv'
    with sum_path.open('w',newline='') as f:
      w=csv.DictWriter(f,fieldnames=list(summary[0].keys()));w.writeheader();w.writerows(summary)
    by={(r['nodes'],r['solver']):r for r in summary}
    ratios=[]
    for n in NODE_SIZES:
      base=by[(n,'bidirectional_dijkstra_p2_model')]
      for solver in ['dijkstra','bidirectional_dijkstra','mrpc_dg_old_w4','mrpc_b3_w1','mrpc_b3_w4','mrpc_b3_w8','mrpc_b3_safe']:
        s=by[(n,solver)]
        ratios.append({'nodes':n,'solver':solver,
          'time_over_bidir_p2':s['mean_time_ms']/base['mean_time_ms'] if base['mean_time_ms'] else math.inf,
          'work_over_bidir_p2':s['mean_query_work_units']/base['mean_query_work_units'] if base['mean_query_work_units'] else math.inf,
          'steps_over_bidir_p2':s['mean_parallel_steps']/base['mean_parallel_steps'] if base['mean_parallel_steps'] else math.inf,
          'found_rate':s['found_rate'],'exact_rate':s['exact_rate'],'within_10pct_rate':s['within_10pct_rate'],
          'mean_distance_ratio':s['mean_distance_ratio'],'worst_distance_ratio':s['worst_distance_ratio'],
          'work_per_node':s['work_per_node'],'target_work_met_rate':s['target_work_met_rate'],
          'unreachable_rate':s['unreachable_rate'],'fallback_rate':s['fallback_rate']})
    rat_path=OUT/'mrpc_b3_bench_ratios.csv'
    with rat_path.open('w',newline='') as f:
      w=csv.DictWriter(f,fieldnames=list(ratios[0].keys()));w.writeheader();w.writerows(ratios)
    print(raw_path, sum_path, rat_path)
if __name__=='__main__': main()
