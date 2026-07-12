from __future__ import annotations
import heapq, math, time, tracemalloc
from typing import Dict, List, Set, Tuple
from ..graph import Graph, euclidean
from ..types import Node, PathResult


def astar(G: Graph, source: Node, target: Node) -> PathResult:
    start=time.perf_counter(); outer=tracemalloc.is_tracing()
    if not outer: tracemalloc.start()
    if source not in G.adj or target not in G.adj:
        _,peak=tracemalloc.get_traced_memory();
        if not outer: tracemalloc.stop()
        return PathResult([],math.inf,False,True,"astar",time_ms=(time.perf_counter()-start)*1000,peak_memory_kib=peak/1024)
    def h(u):
        return euclidean(G.pos[u],G.pos[target]) if G.pos and u in G.pos and target in G.pos else 0.0
    g={source:0.0}; prev={}; pq=[(h(source),0,source)]; c=1
    closed:set[Node]=set(); expanded=relax=pushes=pops=0
    while pq:
        _,_,u=heapq.heappop(pq); pops+=1
        if u in closed: continue
        closed.add(u); expanded+=1
        if u==target: break
        du=g[u]
        for v,w in G.adj.get(u,[]):
            relax+=1; nd=du+w
            if nd<g.get(v,math.inf):
                g[v]=nd; prev[v]=u
                heapq.heappush(pq,(nd+h(v),c,v)); c+=1; pushes+=1
    path=[]
    if target in g:
        cur=target; path=[cur]
        while cur!=source:
            cur=prev[cur]; path.append(cur)
        path.reverse()
    _,peak=tracemalloc.get_traced_memory()
    if not outer: tracemalloc.stop()
    return PathResult(path,g.get(target,math.inf),bool(path),True,"astar",relax,expanded,pushes,pops,expanded,(time.perf_counter()-start)*1000,peak/1024)
