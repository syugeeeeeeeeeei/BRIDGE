from __future__ import annotations
import json, math, pathlib, sys
ROOT = pathlib.Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "others" / "legacy"))
from bridge_py import Gate, Graph

CASES = [
    ("line", 5, [(0,1,1.0),(1,2,2.0),(2,3,3.0),(3,4,4.0)], False, 0, 4),
    ("weighted_unique", 6, [(0,1,1.0),(1,5,9.0),(0,2,2.0),(2,3,2.0),(3,5,2.0),(0,4,20.0),(4,5,1.0)], False, 0, 5),
    ("directed", 5, [(0,1,1.0),(1,2,1.0),(2,4,1.0),(0,3,5.0),(3,4,1.0)], True, 0, 4),
    ("disconnected", 5, [(0,1,1.0),(1,2,1.0),(3,4,1.0)], False, 0, 4),
    ("source_target_equal", 3, [(0,1,1.0),(1,2,1.0)], False, 1, 1),
]
out=[]
for name,n,edges,directed,s,t in CASES:
    g=Graph.from_edges(edges,directed=directed)
    for i in range(n): g.adj.setdefault(i,[])
    for mode in ("exact","quality"):
        r=Gate().route(g,s,t,mode=mode,workers=1)
        out.append({"case":name,"mode":mode,"found":r.found,"distance":None if math.isinf(r.distance) else r.distance,"exact":r.exact,"quality_certified":r.quality_certified,"path":list(r.path)})
print(json.dumps(out, sort_keys=True, separators=(",",":")))
