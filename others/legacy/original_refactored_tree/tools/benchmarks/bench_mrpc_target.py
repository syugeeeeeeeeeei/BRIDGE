from __future__ import annotations
import csv, math, statistics
from pathlib import Path
from bridge_py.graphs.generators import random_geometric_graph, diagonal_extreme_pair
from bridge_py.solvers.dijkstra import dijkstra, bidirectional_dijkstra
from bridge_py.solvers.mprc import mprc
from bridge_py.solvers.mrpc_cg import mrpc_cg, mrpc_cg_target

NODE_SIZES=[10,25,50,100,250,500,1000,2000,3000,5000]
TRIALS=3
SEED=20260710


def row(n, trial, solver, res, exact_dist, edges):
    ratio = res.distance/exact_dist if res.found and math.isfinite(exact_dist) and exact_dist>0 else math.inf
    t=res.telemetry
    exact_match = bool(res.found and abs(res.distance-exact_dist) <= 1e-9*max(1, exact_dist))
    within_10 = bool(res.found and 0.90 <= ratio <= 1.10)
    target_work = max(1, n//10)
    total_work = res.total_work
    query_work_units = int(t.get('query_work_units', total_work) or total_work)
    return {
        'nodes':n,'trial':trial,'edges':edges,'solver':solver,
        'found':res.found,'distance':res.distance,'exact_distance':exact_dist,
        'distance_ratio':ratio,'within_10pct':within_10,'exact_match':exact_match,
        'total_work':total_work,'query_work_units':query_work_units,'target_work':target_work,
        'target_work_met':query_work_units <= target_work,
        'work_expanded_nodes':res.work_expanded_nodes,'work_relaxations':res.work_relaxations,
        'parallel_steps':res.parallel_steps,'time_ms':res.time_ms,'peak_memory_kib':res.peak_memory_kib,
        'compression_ratio':t.get('compression_ratio',''),'portal_count':t.get('portal_count',''),
        'supernode_count':t.get('supernode_count',''),'preprocessing_work':t.get('preprocessing_work', t.get('compression_work','')),
        'query_work_ratio':t.get('query_work_ratio',''),
        'compressed_reachable':t.get('compressed_reachable',''),'expanded_valid':t.get('expanded_valid',''),
        'repair_triggered':t.get('repair_triggered', False),'repair_success':t.get('repair_success',''),
        'fallback_used':t.get('fallback_used', False),'error_code':t.get('error_code',''),
        'portal_cap':t.get('portal_cap',''), 'portal_path_length':t.get('portal_path_length',''),
        'candidate_count':t.get('candidate_count',''), 'k_corridors':t.get('k_corridors','')
    }

def summarize(raw):
    groups={}
    for r in raw:
        groups.setdefault((r['nodes'], r['solver']), []).append(r)
    summary=[]
    for (n, solver), rs in sorted(groups.items()):
        finite=[float(r['distance_ratio']) for r in rs if math.isfinite(float(r['distance_ratio']))]
        summary.append({
            'nodes':n,'solver':solver,'trials':len(rs),
            'found_rate':sum(bool(r['found']) for r in rs)/len(rs),
            'exact_rate':sum(bool(r['exact_match']) for r in rs)/len(rs),
            'within_10pct_rate':sum(bool(r['within_10pct']) for r in rs)/len(rs),
            'mean_distance_ratio':statistics.fmean(finite) if finite else math.inf,
            'worst_distance_ratio':max(finite) if finite else math.inf,
            'mean_work':statistics.fmean(r['total_work'] for r in rs),
            'mean_query_work_units':statistics.fmean(r['query_work_units'] for r in rs),
            'work_per_node':statistics.fmean(r['query_work_units'] for r in rs)/n,
            'target_work_met_rate':sum(bool(r['target_work_met']) for r in rs)/len(rs),
            'mean_parallel_steps':statistics.fmean(r['parallel_steps'] for r in rs),
            'mean_time_ms':statistics.fmean(r['time_ms'] for r in rs),
            'fallback_rate':sum(bool(r['fallback_used']) for r in rs)/len(rs),
            'unreachable_rate':sum(not bool(r['found']) for r in rs)/len(rs),
        })
    return summary

def main():
    raw=[]
    for n in NODE_SIZES:
        for trial in range(1, TRIALS+1):
            G=random_geometric_graph(n, seed=SEED+n*100+trial, k_neighbors=12)
            s,t=diagonal_extreme_pair(G)
            exact=bidirectional_dijkstra(G,s,t)
            exact_dist=exact.distance
            solvers=[
                ('dijkstra', dijkstra(G,s,t)),
                ('bidirectional_dijkstra', exact),
                ('mprc_v1', mprc(G,s,t,workers=1)),
                ('mrpc_cg_fast', mrpc_cg(G,s,t,workers=4, max_distance_ratio=None, exact_fallback=False)),
                ('mrpc_cg_target_w1', mrpc_cg_target(G,s,t,workers=1, max_distance_ratio=1.10, exact_fallback=False)),
                ('mrpc_cg_target_w4', mrpc_cg_target(G,s,t,workers=4, max_distance_ratio=1.10, exact_fallback=False)),
                ('mrpc_cg_target_safe', mrpc_cg_target(G,s,t,workers=4, max_distance_ratio=1.10, exact_fallback=True)),
            ]
            for name,res in solvers:
                raw.append(row(n,trial,name,res,exact_dist,G.edge_count()))
            print('done', n, trial)
    fields=list(raw[0].keys())
    with Path('/mnt/data/mrpc_target_bench_raw.csv').open('w', newline='') as f:
        w=csv.DictWriter(f, fieldnames=fields); w.writeheader(); w.writerows(raw)
    summary=summarize(raw)
    with Path('/mnt/data/mrpc_target_bench_summary.csv').open('w', newline='') as f:
        w=csv.DictWriter(f, fieldnames=list(summary[0].keys())); w.writeheader(); w.writerows(summary)
    by={(r['nodes'],r['solver']):r for r in summary}
    ratios=[]
    for n in NODE_SIZES:
        b=by[(n,'bidirectional_dijkstra')]
        for solver in ['mprc_v1','mrpc_cg_fast','mrpc_cg_target_w1','mrpc_cg_target_w4','mrpc_cg_target_safe']:
            s=by[(n,solver)]
            ratios.append({'nodes':n,'solver':solver,
                'time_over_bidir':s['mean_time_ms']/b['mean_time_ms'],
                'work_over_bidir':s['mean_query_work_units']/b['mean_work'] if b['mean_work'] else math.inf,
                'steps_over_bidir':s['mean_parallel_steps']/b['mean_parallel_steps'] if b['mean_parallel_steps'] else math.inf,
                'exact_rate':s['exact_rate'],
                'within_10pct_rate':s['within_10pct_rate'],
                'mean_distance_ratio':s['mean_distance_ratio'],
                'worst_distance_ratio':s['worst_distance_ratio'],
                'fallback_rate':s['fallback_rate'],
                'unreachable_rate':s['unreachable_rate'],
                'work_per_node':s['work_per_node'],
                'target_work_met_rate':s['target_work_met_rate']})
    with Path('/mnt/data/mrpc_target_bench_ratios.csv').open('w', newline='') as f:
        w=csv.DictWriter(f, fieldnames=list(ratios[0].keys())); w.writeheader(); w.writerows(ratios)
if __name__=='__main__': main()
