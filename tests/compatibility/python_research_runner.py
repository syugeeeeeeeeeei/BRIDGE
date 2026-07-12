from __future__ import annotations
import json, math, sys
from pathlib import Path

ROOT=Path(__file__).resolve().parents[2]
sys.path.insert(0,str(ROOT/'others'/'legacy'))
from bridge_py import Graph, route


def stable_noise(seed:int,u:int,v:int)->float:
    mask=(1<<64)-1
    x=(seed & mask) ^ ((u & 0xffffffff)<<32) ^ (v & 0xffffffff) ^ 0x9e3779b97f4a7c15
    x=(x+0x9e3779b97f4a7c15)&mask
    x=((x^(x>>30))*0xbf58476d1ce4e5b9)&mask
    x=((x^(x>>27))*0x94d049bb133111eb)&mask
    x=(x^(x>>31))&mask
    return (x>>11)/float(1<<53)

def make_grid(n:int, topology:str, seed:int):
    side=max(10,int(math.sqrt(n)))
    blocked=set()
    if topology=='wall':
        x=side//2; gap=max(1,side//10)
        for y in range(side-gap): blocked.add((x,y))
    elif topology=='u_shape':
        x0,x1=side//3,2*side//3; y0,y1=side//4,3*side//4
        for y in range(y0,y1): blocked.add((x0,y)); blocked.add((x1,y))
        for x in range(x0,x1+1): blocked.add((x,y1))
    elif topology=='culdesac':
        x0,x1=side//3,2*side//3; y0,y1=side//4,3*side//4
        for y in range(y0,y1): blocked.add((x0,y)); blocked.add((x1,y))
        for x in range(x0,x1+1): blocked.add((x,y0))
    elif topology=='disconnected':
        x=side//2
        for y in range(side): blocked.add((x,y))
    edges=[]; pos={}
    for y in range(side):
        for x in range(side):
            if (x,y) not in blocked: pos[y*side+x]=(float(x),float(y))
    for y in range(side):
        for x in range(side):
            if (x,y) in blocked: continue
            u=y*side+x
            for dx,dy in ((1,0),(0,1)):
                nx,ny=x+dx,y+dy
                if nx>=side or ny>=side or (nx,ny) in blocked: continue
                v=ny*side+nx
                edges.append((u,v,1+stable_noise(seed,u,v)*.05))
    g=Graph.from_edges(edges,directed=False,pos=pos)
    s=min(pos,key=lambda u:pos[u][0]+pos[u][1]); t=max(pos,key=lambda u:pos[u][0]+pos[u][1])
    return g,s,t


def main():
    out=Path(sys.argv[1] if len(sys.argv)>1 else 'python_research.json')
    rows=[]
    for topology in ('open','wall','u_shape','culdesac','disconnected'):
        for n in (100,225,400,625,900):
            for seed in (1,2,3):
                g,s,t=make_grid(n,topology,seed)
                exact=route(g,s,t,mode='exact')
                budget=len(g.adj)*40
                got=route(g,s,t,mode='balanced',work_budget=budget)
                if got.found and exact.found and exact.distance>0:
                    ratio=got.distance/exact.distance
                    match=abs(got.distance-exact.distance)<=1e-9*max(1,exact.distance)
                elif not got.found and not exact.found:
                    ratio=1.0; match=True
                else:
                    ratio=math.inf; match=False
                rows.append(dict(implementation='python',topology=topology,requested_node_count=side_nodes(n),seed=seed,route_mode='balanced',path_found=got.found,path_cost=got.distance,exact_distance=exact.distance,cost_ratio_to_exact_reference=ratio,matches_exact_reference=match,total_work=got.total_work,scheduled_steps=got.parallel_steps,end_to_end_time_ms=got.time_ms))
    out.write_text(json.dumps(rows,indent=2,allow_nan=True),encoding='utf-8')
    print(f'wrote {len(rows)} rows to {out}')

def side_nodes(n):
    s=max(10,int(math.sqrt(n))); return s*s

if __name__=='__main__': main()
