package bolts

import (
  "context"
  "testing"
  "github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
  "github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)
func benchmarkGrid() *core.AdjacencyGraph {
  const side=32
  g:=core.NewAdjacencyGraph(side*side,false)
  for y:=0;y<side;y++ { for x:=0;x<side;x++ { id:=core.NodeID(y*side+x); g.SetPosition(id,core.Point{X:float64(x),Y:float64(y)}); if x+1<side {_=g.AddEdge(id,id+1,1)}; if y+1<side {_=g.AddEdge(id,id+side,1)} } }
  return g
}
func benchSolver(b *testing.B,s Solver){g:=benchmarkGrid(); req:=core.RouteRequest{Source:0,Target:1023,Workers:1}; o:=bearing.NullObserver{}; b.ReportAllocs(); b.ResetTimer(); for i:=0;i<b.N;i++ {r:=s.Solve(context.Background(),g,req,core.WorkBudget{},o); if !r.Found {b.Fatal("not found")}}}
func BenchmarkDijkstraRef(b *testing.B){benchSolver(b,Dijkstra{})}
func BenchmarkAStarRef(b *testing.B){benchSolver(b,AStar{})}
func BenchmarkWeightedAStarRef(b *testing.B){benchSolver(b,WeightedAStar{ID:"weighted_astar",Weight:1.12})}
func BenchmarkBidirectionalDijkstraRef(b *testing.B){benchSolver(b,BidirectionalDijkstra{})}
func BenchmarkReachabilityRef(b *testing.B){benchSolver(b,Reachability{})}
