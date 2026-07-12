from __future__ import annotations
import csv, math, statistics, time
from pathlib import Path
from bridge_py.graphs.generators import random_geometric_graph, diagonal_extreme_pair
from bridge_py.solvers.dijkstra import dijkstra, bidirectional_dijkstra
from bridge_py.solvers.mprc import mprc
from bridge_py.solvers.mrpc_cg import mrpc_cg

NODE_SIZES=[10,25,50,100,250,500,1000,2000,3000,5000]
TRIALS=1
SEED=20260710

def row(n, trial, solver, res, exact_dist, edges):
    ratio = res.distance/exact_dist if res.found and math.isfinite(exact_dist) and exact_dist>0 else math.inf
    t=res.telemetry
    return {
        'nodes':n,'trial':trial,'edges':edges,'solver':solver,
        'found':res.found,'distance':res.distance,'exact_distance':exact_dist,
        'distance_ratio':ratio,'exact_match':abs(res.distance-exact_dist) <= 1e-9*max(1, exact_dist) if res.found else False,
        'total_work':res.total_work,'work_expanded_nodes':res.work_expanded_nodes,'work_relaxations':res.work_relaxations,
        'parallel_steps':res.parallel_steps,'time_ms':res.time_ms,'peak_memory_kib':res.peak_memory_kib,
        'compression_ratio':t.get('compression_ratio',''),'compressed_nodes':t.get('compressed_nodes',''),
        'compression_work':t.get('compression_work',''),'query_work_ratio':t.get('query_work_ratio',''),
        'compressed_reachable':t.get('compressed_reachable',''),'expanded_valid':t.get('expanded_valid',''),
        'repair_triggered':t.get('repair_triggered', False),'repair_success':t.get('repair_success',''),
        'fallback_used':t.get('fallback_used', False),'error_code':t.get('error_code',''),
        'candidate_count':t.get('candidate_count',''), 'k_corridors':t.get('k_corridors','')
    }

def main():
    raw=[]
    for n in NODE_SIZES:
        for trial in range(1, TRIALS+1):
            G=random_geometric_graph(n, seed=SEED+n*100+trial, k_neighbors=12)
            s,t=diagonal_extreme_pair(G)
            exact=bidirectional_dijkstra(G,s,t)
            exact_dist=exact.distance
            raw.append(row(n,trial,'dijkstra', dijkstra(G,s,t), exact_dist, G.edge_count()))
            raw.append(row(n,trial,'bidirectional_dijkstra', exact, exact_dist, G.edge_count()))
            raw.append(row(n,trial,'mprc_v1', mprc(G,s,t,workers=1), exact_dist, G.edge_count()))
            raw.append(row(n,trial,'mrpc_cg_fast', mrpc_cg(G,s,t,workers=4, max_distance_ratio=None, exact_fallback=False), exact_dist, G.edge_count()))
            raw.append(row(n,trial,'mrpc_cg_safe', mrpc_cg(G,s,t,workers=4, max_distance_ratio=1.02, exact_fallback=True), exact_dist, G.edge_count()))
            print('done', n, trial)
    out=Path('/mnt/data/mrpc_cg_bench_raw.csv')
    with out.open('w', newline='') as f:
        w=csv.DictWriter(f, fieldnames=list(raw[0].keys())); w.writeheader(); w.writerows(raw)
    groups={}
    for r in raw:
        groups.setdefault((r['nodes'], r['solver']), []).append(r)
    summary=[]
    for (n, solver), rs in sorted(groups.items()):
        finite=[float(r['distance_ratio']) for r in rs if math.isfinite(float(r['distance_ratio']))]
        summary.append({
            'nodes':n,'solver':solver,'trials':len(rs),
            'found_rate':sum(r['found'] for r in rs)/len(rs),
            'exact_rate':sum(r['exact_match'] for r in rs)/len(rs),
            'mean_distance_ratio':statistics.fmean(finite) if finite else math.inf,
            'worst_distance_ratio':max(finite) if finite else math.inf,
            'mean_work':statistics.fmean(r['total_work'] for r in rs),
            'work_per_node':statistics.fmean(r['total_work'] for r in rs)/n,
            'mean_parallel_steps':statistics.fmean(r['parallel_steps'] for r in rs),
            'mean_time_ms':statistics.fmean(r['time_ms'] for r in rs),
            'mean_compression_ratio':statistics.fmean(float(r['compression_ratio']) for r in rs if r['compression_ratio']!='') if any(r['compression_ratio']!='' for r in rs) else '',
            'mean_query_work_ratio':statistics.fmean(float(r['query_work_ratio']) for r in rs if r['query_work_ratio']!='') if any(r['query_work_ratio']!='' for r in rs) else '',
            'fallback_rate':sum(bool(r['fallback_used']) for r in rs)/len(rs),
            'repair_success_rate':sum(r['repair_success'] is True or r['repair_success']=='True' for r in rs)/max(1,sum(bool(r['repair_triggered']) for r in rs)),
        })
    with Path('/mnt/data/mrpc_cg_bench_summary.csv').open('w', newline='') as f:
        w=csv.DictWriter(f, fieldnames=list(summary[0].keys())); w.writeheader(); w.writerows(summary)
    # compact comparison table for main solvers
    by={(r['nodes'],r['solver']):r for r in summary}
    ratios=[]
    for n in NODE_SIZES:
        b=by[(n,'bidirectional_dijkstra')]
        for solver in ['mprc_v1','mrpc_cg_fast','mrpc_cg_safe']:
            s=by[(n,solver)]
            ratios.append({'nodes':n,'solver':solver,
                'time_over_bidir':s['mean_time_ms']/b['mean_time_ms'],
                'work_over_bidir':s['mean_work']/b['mean_work'] if b['mean_work'] else math.inf,
                'steps_over_bidir':s['mean_parallel_steps']/b['mean_parallel_steps'] if b['mean_parallel_steps'] else math.inf,
                'mean_distance_ratio':s['mean_distance_ratio'],
                'worst_distance_ratio':s['worst_distance_ratio'],
                'fallback_rate':s['fallback_rate'],
                'work_per_node':s['work_per_node']})
    with Path('/mnt/data/mrpc_cg_bench_ratios.csv').open('w', newline='') as f:
        w=csv.DictWriter(f, fieldnames=list(ratios[0].keys())); w.writeheader(); w.writerows(ratios)
if __name__=='__main__': main()
