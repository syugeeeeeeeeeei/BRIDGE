from __future__ import annotations
import random
from ..core.graph import Graph

def grid_graph(width:int,height:int,*,seed:int=0,noise:float=0.0,barrier_x:int|None=None,gap_y:int|None=None)->Graph:
    rng=random.Random(seed); edges=[]; pos={}
    for y in range(height):
        for x in range(width): pos[y*width+x]=(float(x),float(y))
    for y in range(height):
        for x in range(width):
            u=y*width+x
            for dx,dy in ((1,0),(0,1)):
                nx,ny=x+dx,y+dy
                if nx>=width or ny>=height: continue
                if barrier_x is not None and x==barrier_x-1 and nx==barrier_x and y != gap_y: continue
                v=ny*width+nx; w=1.0+rng.random()*noise
                edges.append((u,v,w))
    return Graph.from_edges(edges,directed=False,pos=pos)

def disconnected_graph()->Graph:
    return Graph.from_edges([(0,1,1.0),(1,2,1.0),(10,11,1.0)],directed=False,pos={0:(0,0),1:(1,0),2:(2,0),10:(10,0),11:(11,0)})
