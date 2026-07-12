from __future__ import annotations
import math
from dataclasses import dataclass
from ..core.graph import euclidean

@dataclass(frozen=True, slots=True)
class AnchorPlan:
    strategy: str
    alternates: tuple[str, ...]
    reason: str
    features: dict[str, object]

def analyze_graph(graph, source, target) -> dict[str, object]:
    n=max(1,len(graph.adj)); deg=[len(v) for v in graph.adj.values()]
    mean=sum(deg)/n if deg else 0.0
    var=sum((d-mean)**2 for d in deg)/n if deg else 0.0
    cv=math.sqrt(var)/max(mean,1e-12)
    max_mean=max(deg)/max(mean,1e-12) if deg else 0.0
    ratio_cv=None
    if graph.pos:
        ratios=[]; seen=set()
        for u,nbrs in graph.adj.items():
            if u not in graph.pos: continue
            for v,w in nbrs:
                if v not in graph.pos: continue
                key=(u,v) if graph.directed else tuple(sorted((u,v),key=repr))
                if key in seen: continue
                seen.add(key); d=euclidean(graph.pos[u],graph.pos[v])
                if d>1e-12: ratios.append(float(w)/d)
                if len(ratios)>=768: break
            if len(ratios)>=768: break
        if ratios:
            m=sum(ratios)/len(ratios)
            ratio_cv=math.sqrt(sum((x-m)**2 for x in ratios)/len(ratios))/max(m,1e-12)
    return {"nodes":n,"edges":graph.edge_count(),"has_pos":graph.pos is not None,
            "mean_degree":mean,"degree_cv":cv,"max_mean_degree_ratio":max_mean,
            "weight_geo_ratio_cv":ratio_cv}

def make_anchor_plan(graph, source, target) -> AnchorPlan:
    f=analyze_graph(graph,source,target)
    if f["has_pos"] and (f["weight_geo_ratio_cv"] is None or f["weight_geo_ratio_cv"] < .35):
        primary="geometric_corridor"; reason="geometry_is_predictive"
    elif f["has_pos"] and f["mean_degree"] < 4.5:
        primary="portal"; reason="sparse_geometric_detours"
    elif f["max_mean_degree_ratio"] >= 4.0 or f["degree_cv"] >= .9:
        primary="hub_aware"; reason="hub_dominated_topology"
    else:
        primary="weighted_cost"; reason="generic_weighted_topology"
    order=("portal","geometric_corridor","weighted_cost","hub_aware")
    return AnchorPlan(primary, tuple(x for x in order if x!=primary), reason, f)
