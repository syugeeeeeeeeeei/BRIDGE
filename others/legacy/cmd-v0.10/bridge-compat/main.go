package main

import (
	"context"
	"encoding/json"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
	"math"
	"os"
)

type edge struct {
	u, v core.NodeID
	w    float64
}
type fixture struct {
	name     string
	n        int
	edges    []edge
	directed bool
	s, t     core.NodeID
}
type row struct {
	Case             string         `json:"case"`
	Mode             core.RouteMode `json:"mode"`
	Found            bool           `json:"found"`
	Distance         *float64       `json:"distance"`
	Exact            bool           `json:"exact"`
	QualityCertified bool           `json:"quality_certified"`
	Path             []core.NodeID  `json:"path"`
}

func main() {
	cases := []fixture{
		{"line", 5, []edge{{0, 1, 1}, {1, 2, 2}, {2, 3, 3}, {3, 4, 4}}, false, 0, 4},
		{"weighted_unique", 6, []edge{{0, 1, 1}, {1, 5, 9}, {0, 2, 2}, {2, 3, 2}, {3, 5, 2}, {0, 4, 20}, {4, 5, 1}}, false, 0, 5},
		{"directed", 5, []edge{{0, 1, 1}, {1, 2, 1}, {2, 4, 1}, {0, 3, 5}, {3, 4, 1}}, true, 0, 4},
		{"disconnected", 5, []edge{{0, 1, 1}, {1, 2, 1}, {3, 4, 1}}, false, 0, 4},
		{"source_target_equal", 3, []edge{{0, 1, 1}, {1, 2, 1}}, false, 1, 1},
	}
	out := []row{}
	for _, c := range cases {
		g := core.NewAdjacencyGraph(c.n, c.directed)
		for _, e := range c.edges {
			_ = g.AddEdge(e.u, e.v, e.w)
		}
		for _, m := range []core.RouteMode{core.ModeExact, core.ModeQuality} {
			r, err := gate.New(nil).Route(context.Background(), g, core.RouteRequest{Source: c.s, Target: c.t, Mode: m, Workers: 1})
			if err != nil {
				panic(err)
			}
			var d *float64
			if !math.IsInf(r.Distance, 1) {
				x := r.Distance
				d = &x
			}
			out = append(out, row{Case: c.name, Mode: m, Found: r.Found, Distance: d, Exact: r.Exact, QualityCertified: r.QualityCertified, Path: append([]core.NodeID{}, r.Path...)})
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		panic(err)
	}
}
