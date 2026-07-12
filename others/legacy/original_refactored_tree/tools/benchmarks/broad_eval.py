from __future__ import annotations
import csv, json, math, random, statistics, time, tracemalloc
from collections import defaultdict
from pathlib import Path
from bridge_py.graph import Graph, path_distance
from bridge_py.solvers.dijkstra import dijkstra, bidirectional_dijkstra
from bridge_py.solvers.astar import astar
from bridge_py.solvers.mrpc_cg import mrpc_dg5_switchback


def grid_graph(side:int, topo:str, seed:int, noise:float=0.0):
    rng=random.Random(seed); blocked=set(); w=h=side
    if topo=='wall':
        x=w//2; gap_y=h-2-rng.randrange(max(1,h//5))
        blocked.update((x,y) for y in range(h) if abs(y-gap_y)>0)
    elif topo=='double_wall':
        for i,x in enumerate((w//3,2*w//3)):
            gap_y=(h-2 if i==0 else 1) if seed%2==0 else (1 if i==0 else h-2)
            blocked.update((x,y) for y in range(h) if y!=gap_y)
    elif topo=='u_shape':
        x0,x1=w//3,2*w//3; y0=h//5; y1=4*h//5
        blocked.update((x0,y) for y in range(y0,y1)); blocked.update((x1,y) for y in range(y0,y1)); blocked.update((x,y1) for x in range(x0,x1+1))
    elif topo=='culdesac':
        x0,x1=w//3,2*w//3; y0=h//5; y1=4*h//5
        blocked.update((x0,y) for y in range(y0,y1)); blocked.update((x1,y) for y in range(y0,y1)); blocked.update((x,y0) for x in range(x0,x1+1))
    elif topo=='spiral':
        # sparse rectangular spiral walls with one-cell gaps
        layers=max(2,side//10)
        for k in range(2, min(side//2-1,layers*2), 3):
            for x in range(k,side-k): blocked.add((x,k))
            for y in range(k,side-k): blocked.add((side-k-1,y))
            for x in range(k+2,side-k): blocked.add((x,side-k-1))
            for y in range(k+2,side-k-2): blocked.add((k,y))
            blocked.discard((k+1,k)); blocked.discard((k,side-k-3))
    elif topo=='random_obstacles':
        p=0.12
        for y in range(h):
            for x in range(w):
                if (x,y) not in ((0,0),(w-1,h-1)) and rng.random()<p: blocked.add((x,y))
    elif topo=='disconnected':
        blocked.update((w//2,y) for y in range(h))
    edges=[]; pos={}
    for y in range(h):
        for x in range(w):
            if (x,y) in blocked: continue
            u=y*w+x; pos[u]=(float(x),float(y))
            for dx,dy in ((1,0),(0,1)):
                xx,yy=x+dx,y+dy
                if xx<w and yy<h and (xx,yy) not in blocked:
                    wt=1.0
                    if noise: wt=max(0.05,1.0+rng.uniform(-noise,noise))
                    edges.append((u,yy*w+xx,wt))
    G=Graph.from_edges(edges,pos=pos)
    s=0 if 0 in G.adj else min(G.adj)
    t=(h-1)*w+w-1 if (h-1)*w+w-1 in G.adj else max(G.adj)
    return G,s,t


def random_geometric(n:int, seed:int, k:int=8):
    rng=random.Random(seed); pos={i:(rng.random(),rng.random()) for i in range(n)}
    # grid buckets for approximate k-nearest
    cells=max(2,int(math.sqrt(n/8))); buckets=defaultdict(list)
    for i,(x,y) in pos.items(): buckets[(min(cells-1,int(x*cells)),min(cells-1,int(y*cells)))].append(i)
    edges_set=set(); edges=[]
    for i,(x,y) in pos.items():
        cx,cy=min(cells-1,int(x*cells)),min(cells-1,int(y*cells)); cand=[]
        radius=1
        while len(cand)<k and radius<=cells:
            cand=[]
            for dx in range(-radius,radius+1):
                for dy in range(-radius,radius+1): cand.extend(buckets.get((cx+dx,cy+dy),[]))
            radius+=1
        cand=[j for j in cand if j!=i]
        cand.sort(key=lambda j:(pos[j][0]-x)**2+(pos[j][1]-y)**2)
        for j in cand[:k]:
            a,b=sorted((i,j))
            if (a,b) in edges_set: continue
            edges_set.add((a,b)); d=math.hypot(pos[a][0]-pos[b][0],pos[a][1]-pos[b][1]); edges.append((a,b,d))
    G=Graph.from_edges(edges,pos=pos)
    s=min(pos,key=lambda i:pos[i][0]+pos[i][1]); t=max(pos,key=lambda i:pos[i][0]+pos[i][1])
    return G,s,t


def clustered(n:int, seed:int):
    rng=random.Random(seed); clusters=4; per=n//clusters; pos={}; edges=[]
    centers=[(0.2,0.2),(0.8,0.2),(0.2,0.8),(0.8,0.8)]
    for c,(cx,cy) in enumerate(centers):
        ids=[]
        for j in range(per):
            i=c*per+j; pos[i]=(min(1,max(0,rng.gauss(cx,.08))),min(1,max(0,rng.gauss(cy,.08)))); ids.append(i)
        # ring + random chords
        for a,b in zip(ids,ids[1:]+ids[:1]): edges.append((a,b,math.dist(pos[a],pos[b])))
        for i in ids:
            for _ in range(4):
                j=rng.choice(ids)
                if i!=j: edges.append((i,j,math.dist(pos[i],pos[j])))
    # sparse bridges
    bridges=[(per-1,per),(2*per-1,3*per),(0,2*per)]
    for a,b in bridges: edges.append((a,b,math.dist(pos[a],pos[b])))
    G=Graph.from_edges(edges,pos=pos); return G,0,4*per-1


def scale_free(n:int, seed:int, with_pos:bool):
    rng=random.Random(seed); edges=[]; degrees=[0]*n
    for i in range(1,min(4,n)):
        for j in range(i): edges.append((i,j,1.0)); degrees[i]+=1; degrees[j]+=1
    for i in range(4,n):
        targets=set(); total=sum(degrees[:i])
        while len(targets)<3:
            r=rng.uniform(0,total); acc=0
            for j in range(i):
                acc+=degrees[j]
                if acc>=r: targets.add(j); break
        for j in targets: edges.append((i,j,1.0)); degrees[i]+=1; degrees[j]+=1
    pos={i:(rng.random(),rng.random()) for i in range(n)} if with_pos else None
    G=Graph.from_edges(edges,pos=pos); return G,n-1,n-2


def run_solver(name,G,s,t):
    fn={'dg5':lambda:mrpc_dg5_switchback(G,s,t,workers=1,trace_level=0,measure_memory=False), 'astar':lambda:astar(G,s,t), 'dijkstra':lambda:dijkstra(G,s,t), 'bidir':lambda:bidirectional_dijkstra(G,s,t)}[name]
    tracemalloc.start(); start=time.perf_counter(); r=fn(); elapsed=(time.perf_counter()-start)*1000; _,peak=tracemalloc.get_traced_memory(); tracemalloc.stop()
    tel=r.telemetry or {}; pd=path_distance(G,r.path) if r.found else math.inf
    valid=(not r.found) or (r.path and r.path[0]==s and r.path[-1]==t and math.isfinite(pd) and abs(pd-r.distance)<=1e-7*max(1,pd))
    return r,elapsed,peak/1024,valid,pd,int(tel.get('total_work_including_preprocessing', int(tel.get('preprocessing_work',0)) + int(tel.get('query_work_units',r.total_work))))


def main():
    out=Path('evaluation_results/dg5_broad'); out.mkdir(parents=True,exist_ok=True)
    cases=[]
    for topo in ['open','weighted_noise','wall','double_wall','u_shape','culdesac','spiral','random_obstacles','disconnected']:
        for side in [20,40,70]:
            for seed in range(3): cases.append((topo,side*side,seed,lambda topo=topo,side=side,seed=seed:grid_graph(side,'normal' if topo=='open' else topo,seed,0.8 if topo=='weighted_noise' else 0.0)))
    for n in [400,900,1600]:
        for seed in range(3):
            cases += [('random_geometric',n,seed,lambda n=n,seed=seed:random_geometric(n,seed)),('clustered',n,seed,lambda n=n,seed=seed:clustered(n,seed)),('scale_free_pos',n,seed,lambda n=n,seed=seed:scale_free(n,seed,True)),('scale_free_no_pos',n,seed,lambda n=n,seed=seed:scale_free(n,seed,False))]
    rows=[]
    for idx,(topo,nreq,seed,maker) in enumerate(cases):
        G,s,t=maker(); results={}
        for name in ['dijkstra','bidir','astar','dg5']:
            r,tm,pk,valid,pd,work=run_solver(name,G,s,t); results[name]=r
            rows.append({'case_id':idx,'topology':topo,'requested_nodes':nreq,'nodes':len(G.adj),'edges':G.edge_count(),'seed':seed,'solver':name,'found':r.found,'valid':valid,'distance':r.distance,'path_distance':pd,'total_work':work,'total_time_ms':tm,'peak_kib_uniform':pk,'steps':r.parallel_steps,'switch_count':(r.telemetry or {}).get('switch_count',0),'reentry_count':(r.telemetry or {}).get('reentry_count',0),'fallback_used':(r.telemetry or {}).get('fallback_used',False),'error_code':(r.telemetry or {}).get('error_code')})
        exact=results['dijkstra']
        for rr in rows[-4:]:
            if exact.found and rr['found'] and exact.distance>0: rr['distance_ratio']=rr['distance']/exact.distance
            elif exact.found==rr['found']: rr['distance_ratio']=1.0
            else: rr['distance_ratio']=math.inf
        if (idx+1)%20==0: print('cases',idx+1,'/',len(cases),flush=True)
    fields=sorted({k for r in rows for k in r})
    with (out/'raw.csv').open('w',newline='') as f: w=csv.DictWriter(f,fieldnames=fields); w.writeheader(); w.writerows(rows)
    groups=defaultdict(list)
    for r in rows: groups[(r['topology'],r['solver'])].append(r)
    summary=[]
    for (topo,solver),rs in sorted(groups.items()):
        finite=[r['distance_ratio'] for r in rs if math.isfinite(r['distance_ratio'])]
        summary.append({'topology':topo,'solver':solver,'cases':len(rs),'found_rate':sum(r['found'] for r in rs)/len(rs),'valid_rate':sum(r['valid'] for r in rs)/len(rs),'mean_distance_ratio':statistics.mean(finite) if finite else math.inf,'worst_distance_ratio':max(finite) if finite else math.inf,'mean_total_work':statistics.mean(r['total_work'] for r in rs),'median_total_work':statistics.median(r['total_work'] for r in rs),'mean_total_time_ms':statistics.mean(r['total_time_ms'] for r in rs),'median_total_time_ms':statistics.median(r['total_time_ms'] for r in rs),'mean_steps':statistics.mean(r['steps'] for r in rs),'switch_rate':sum(r['switch_count']>0 for r in rs)/len(rs),'reentry_rate':sum(r['reentry_count']>0 for r in rs)/len(rs)})
    sf=sorted({k for r in summary for k in r})
    with (out/'summary.csv').open('w',newline='') as f: w=csv.DictWriter(f,fieldnames=sf); w.writeheader(); w.writerows(summary)
    # per-case winner rates among valid solutions; lower time/work
    wins=[]
    for topo in sorted(set(r['topology'] for r in rows)):
        cr=[r for r in rows if r['topology']==topo]
        bycase=defaultdict(list)
        for r in cr: bycase[r['case_id']].append(r)
        dg_time=dg_work=eligible=0
        for rs in bycase.values():
            valid=[r for r in rs if r['valid'] and (r['found']==next(x for x in rs if x['solver']=='dijkstra')['found'])]
            dg=next(r for r in rs if r['solver']=='dg5')
            if dg in valid:
                eligible+=1; dg_time += dg['total_time_ms']<=min(r['total_time_ms'] for r in valid); dg_work += dg['total_work']<=min(r['total_work'] for r in valid)
        wins.append({'topology':topo,'eligible_cases':eligible,'dg5_time_win_rate':dg_time/max(1,eligible),'dg5_work_win_rate':dg_work/max(1,eligible)})
    wf=sorted({k for r in wins for k in r})
    with (out/'wins.csv').open('w',newline='') as f: w=csv.DictWriter(f,fieldnames=wf); w.writeheader(); w.writerows(wins)
    print(json.dumps({'cases':len(cases),'runs':len(rows),'out':str(out)}))
if __name__=='__main__': main()
