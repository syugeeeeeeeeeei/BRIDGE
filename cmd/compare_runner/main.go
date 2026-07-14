package main

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
    "github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
    "github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
    "github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/ultrasound"
)

type graphDoc struct {
    SchemaVersion string `json:"schema_version"`
    ID string `json:"id"`
    Directed bool `json:"directed"`
    Nodes int `json:"nodes"`
    Edges []gate.GraphEdge `json:"edges"`
    DefaultQuery map[string]uint32 `json:"default_query"`
}

type runSummary struct {
    Algorithm string `json:"algorithm"`
    Status string `json:"status"`
    Found bool `json:"found"`
    Exact bool `json:"exact"`
    Distance *float64 `json:"distance,omitempty"`
    Path []uint32 `json:"path"`
    Work core.WorkMetrics `json:"work"`
    SolverTimeMS float64 `json:"solver_time_ms"`
    EndToEndMS float64 `json:"end_to_end_time_ms"`
    TracePath string `json:"trace_path"`
    EventCount uint64 `json:"event_count"`
}

func writeJSON(path string, v any) error {
    b, err := json.MarshalIndent(v, "", "  ")
    if err != nil { return err }
    return os.WriteFile(path, append(b, '\n'), 0644)
}

func main() {
    out := "/mnt/data/compare6"
    if len(os.Args) > 1 { out = os.Args[1] }
    if err := os.MkdirAll(out, 0755); err != nil { panic(err) }

    const nodes = 300
    const communities = 6
    const seed int64 = 20260714
    g, source, target, err := traffic.CommunityGraph(nodes, communities, seed)
    if err != nil { panic(err) }

    nodesIn := make([]gate.GraphNode, nodes)
    for i := 0; i < nodes; i++ {
        n := core.NodeID(i)
        gn := gate.GraphNode{ID:uint32(i)}
        if p, ok := g.Position(n); ok {
            x,y := p.X,p.Y; gn.X=&x; gn.Y=&y
        }
        nodesIn[i]=gn
    }
    edges := make([]gate.GraphEdge,0,g.EdgeCount())
    for u:=0;u<nodes;u++ {
        for _,e := range g.EdgesFrom(core.NodeID(u)) {
            if !g.Directed() && core.NodeID(u) > e.To { continue }
            edges=append(edges,gate.GraphEdge{From:uint32(u),To:uint32(e.To),Weight:e.Weight})
        }
    }
    graphInput := gate.GraphInput{Type:"inline",Directed:g.Directed(),Nodes:nodesIn,Edges:edges}
    if err := writeJSON(filepath.Join(out,"graph.json"), graphDoc{SchemaVersion:"bridge.dataset.v1",ID:"community-300-seed-20260714",Directed:g.Directed(),Nodes:nodes,Edges:edges,DefaultQuery:map[string]uint32{"source":uint32(source),"target":uint32(target)}}); err != nil { panic(err) }

    algorithms := []string{"dijkstra","bidirectional_dijkstra","astar","weighted_astar","bridge","anchor"}
    summaries := make([]runSummary,0,len(algorithms))
    router := gate.NewRouter()
    budget := uint64(2_000_000)
    timeout := float64(30_000)

    for _, algorithm := range algorithms {
        tracePath := filepath.Join(out, algorithm+".trace.jsonl")
        sink, err := ultrasound.NewFileSink(tracePath,true); if err != nil { panic(err) }
        collector := ultrasound.NewCollector("trace",sink)
        opts := gate.RouteOptions{Observation:gate.ObservationOptions{Mode:gate.ObservationTrace,Observer:collector,Reporter:collector}}
        started := time.Now()
        var summary runSummary
        summary.Algorithm=algorithm; summary.TracePath=tracePath
        if algorithm=="bridge" {
            result, runErr := router.Route(context.Background(), gate.RouteRequest{
                SchemaVersion:gate.RouteRequestSchemaV1,RequestID:"compare-"+algorithm,Graph:graphInput,
                Route:gate.RouteInput{Source:uint32(source),Target:uint32(target),Mode:core.ModeBalanced,Workers:1,Seed:uint64(seed)},
                Budget:gate.BudgetInput{TotalWork:&budget,TimeoutMS:&timeout},Observation:gate.ObservationInput{Mode:gate.ObservationTrace},
            },opts)
            if runErr != nil { panic(runErr) }
            summary.Status=result.Status;summary.Found=result.Found;summary.Exact=result.Exact;summary.Distance=result.Distance;summary.Path=result.Path;summary.Work=result.Work;summary.SolverTimeMS=result.SolverTimeMS;summary.EndToEndMS=result.TimeMS
        } else {
            result, runErr := router.ExecuteOnce(context.Background(), gate.ExecuteRequest{
                SchemaVersion:gate.ExecuteRequestSchemaV1,RequestID:"compare-"+algorithm,Target:gate.ExecuteTargetInput{ID:algorithm},Graph:graphInput,
                Route:gate.RouteInput{Source:uint32(source),Target:uint32(target),Mode:core.ModeBalanced,Workers:1,Seed:uint64(seed)},
                Budget:gate.BudgetInput{TotalWork:&budget,TimeoutMS:&timeout},Observation:gate.ObservationInput{Mode:gate.ObservationTrace},
            },opts)
            if runErr != nil { panic(runErr) }
            summary.Status=result.Status;summary.Found=result.Found;summary.Exact=result.Exact;summary.Distance=result.Distance;summary.Path=result.Path;summary.Work=result.Work;summary.SolverTimeMS=result.SolverTimeMS;summary.EndToEndMS=result.EndToEndMS
        }
        if err := collector.Close(context.Background()); err != nil { panic(err) }
        summary.EventCount=collector.ObservationEventCount()
        if summary.EndToEndMS==0 { summary.EndToEndMS=float64(time.Since(started).Nanoseconds())/1e6 }
        if err := writeJSON(filepath.Join(out,algorithm+".result.json"),summary); err != nil { panic(err) }
        summaries=append(summaries,summary)
        fmt.Printf("%s found=%v cost=%v work=%d events=%d\n",algorithm,summary.Found,summary.Distance,summary.Work.TotalActions,summary.EventCount)
    }
    if err := writeJSON(filepath.Join(out,"benchmark_summary.json"),map[string]any{
        "schema_version":"bridge.simulation.comparison.v1","graph":"community-300","nodes":nodes,"edges":len(edges),"seed":seed,"source":source,"target":target,"algorithms":summaries,
    }); err != nil { panic(err) }
}
