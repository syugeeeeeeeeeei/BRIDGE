from __future__ import annotations
import csv, math, statistics, time, json
from collections import defaultdict
from pathlib import Path
from bridge_py.graph import path_distance
from bridge_py.solvers.dijkstra import dijkstra
from bridge_py.solvers.mrpc_dg6 import mrpc_dg6
from broad_eval import grid_graph, random_geometric, clustered, scale_free

def cases():
    cs=[]
    for topo in ['open','weighted_noise','wall','double_wall','u_shape','culdesac','spiral','random_obstacles','disconnected']:
        for side in [20,40]:
            for seed in range(3):
                cs.append((topo, side*side, seed, lambda topo=topo, side=side, seed=seed: grid_graph(side, 'normal' if topo=='open' else topo, seed, 0.8 if topo=='weighted_noise' else 0.0)))
    for n in [400,900]:
        for seed in range(3):
            cs += [
                ('random_geometric', n, seed, lambda n=n, seed=seed: random_geometric(n, seed)),
                ('clustered', n, seed, lambda n=n, seed=seed: clustered(n, seed)),
                ('scale_free_pos', n, seed, lambda n=n, seed=seed: scale_free(n, seed, True)),
                ('scale_free_no_pos', n, seed, lambda n=n, seed=seed: scale_free(n, seed, False)),
            ]
    return cs

def eval_cfg(name, params, prebuilt):
    rows=[]
    t0=time.perf_counter()
    for idx,(topo,nreq,seed,G,s,t,exact_found,exact_dist) in enumerate(prebuilt):
        r=mrpc_dg6(G,s,t,fallback_exact=False,**params)
        pd=path_distance(G,r.path) if r.found else math.inf
        valid=(not r.found) or (r.path and r.path[0]==s and r.path[-1]==t and math.isfinite(pd) and abs(pd-r.distance)<=1e-7*max(1,pd))
        if exact_found and r.found and exact_dist>0:
            ratio=r.distance/exact_dist
        elif exact_found==r.found:
            ratio=1.0
        else:
            ratio=math.inf
        tel=r.telemetry or {}
        rows.append(dict(config=name,case_id=idx,topology=topo,nodes=len(G.adj),seed=seed,found=r.found,valid=valid,distance_ratio=ratio,within_10pct=math.isfinite(ratio) and ratio<=1.10+1e-9,exact_match=math.isfinite(ratio) and abs(ratio-1)<=1e-9,total_work=int(tel.get('total_work_including_preprocessing',r.total_work)),time_ms=r.time_ms,steps=r.parallel_steps,strategy=tel.get('strategy',''),emergency=tel.get('emergency_path_used',False),repair=tel.get('repair_triggered',False),first_path_work=tel.get('first_path_work',''),quality_budget_used=tel.get('quality_budget_used',''),target_work=tel.get('target_work',''),work_goal_ratio=tel.get('work_goal_ratio','')))
    finite=[r['distance_ratio'] for r in rows if math.isfinite(r['distance_ratio'])]
    reach=[r for r in rows if r['topology']!='disconnected']
    summary=dict(config=name,params=json.dumps(params,sort_keys=True),cases=len(rows),found_rate=sum(r['found'] for r in rows)/len(rows),valid_rate=sum(r['valid'] for r in rows)/len(rows),exact_rate=sum(r['exact_match'] for r in rows)/len(rows),within_10pct_rate=sum(r['within_10pct'] for r in rows)/len(rows),reachable_within_10pct_rate=sum(r['within_10pct'] for r in reach)/len(reach),mean_distance_ratio=statistics.mean(finite) if finite else math.inf,worst_distance_ratio=max(finite) if finite else math.inf,mean_work=statistics.mean(r['total_work'] for r in rows),median_work=statistics.median(r['total_work'] for r in rows),work_le_half_rate=sum((r['total_work']<=r['nodes']*0.5) for r in rows)/len(rows),mean_time_ms=statistics.mean(r['time_ms'] for r in rows),mean_steps=statistics.mean(r['steps'] for r in rows),emergency_rate=sum(r['emergency'] for r in rows)/len(rows),repair_rate=sum(r['repair'] for r in rows)/len(rows),elapsed=time.perf_counter()-t0)
    return summary, rows

def main():
    out=Path('evaluation_results/dg6_tuning'); out.mkdir(parents=True,exist_ok=True)
    pre=[]
    for idx,(topo,nreq,seed,maker) in enumerate(cases()):
        G,s,t=maker(); er=dijkstra(G,s,t); pre.append((topo,nreq,seed,G,s,t,er.found,er.distance))
    configs=[]
    base=dict(target_work_ratio=0.50,initial_path_budget_ratio=0.22,min_quality_budget_ratio=0.06,connector_budget_ratio=0.22,max_repair_nodes_ratio=0.22,repair_hops=1,base_width_scale=0.14,hub_count=8,weighted_astar_factor=1.12)
    configs.append(('current',base))
    # Coarse grid focused on budget distribution rather than total N/2 target.
    for init in [0.14,0.18,0.22,0.26,0.30]:
      for conn in [0.12,0.16,0.20,0.24,0.28]:
        for minq in [0.03,0.06,0.10]:
          params={**base,'initial_path_budget_ratio':init,'connector_budget_ratio':conn,'min_quality_budget_ratio':minq}
          configs.append((f'i{init:.2f}_c{conn:.2f}_q{minq:.2f}',params))
    # Some repair geometry variations.
    for rh in [1,2]:
      for rr in [0.10,0.16,0.22,0.30]:
        params={**base,'repair_hops':rh,'max_repair_nodes_ratio':rr}
        configs.append((f'rh{rh}_rr{rr:.2f}',params))
    summaries=[]; allrows=[]
    for i,(name,params) in enumerate(configs):
        s,rows=eval_cfg(name,params,pre); summaries.append(s); allrows.extend(rows)
        print(i+1,len(configs),name,s['within_10pct_rate'],s['mean_work'],s['worst_distance_ratio'],flush=True)
    fields=sorted({k for s in summaries for k in s})
    with (out/'tuning_summary.csv').open('w',newline='') as f:
        w=csv.DictWriter(f,fieldnames=fields); w.writeheader(); w.writerows(summaries)
    fields=sorted({k for r in allrows for k in r})
    with (out/'tuning_raw.csv').open('w',newline='') as f:
        w=csv.DictWriter(f,fieldnames=fields); w.writeheader(); w.writerows(allrows)
    # topology breakdown for top configs by lexicographic objective
    def score(s):
        penalty=max(0,0.95-s['within_10pct_rate'])*1000 + max(0,s['worst_distance_ratio']-1.25)*100 + max(0,s['mean_distance_ratio']-1.03)*100
        return (penalty, s['mean_work'], -s['work_le_half_rate'])
    top=sorted(summaries,key=score)[:8]
    print('TOP',json.dumps(top[:5],indent=2))
    with (out/'top_configs.json').open('w') as f: json.dump(top,f,indent=2)
if __name__=='__main__': main()
