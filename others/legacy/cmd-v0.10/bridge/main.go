package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/traffic"
	"os"
)

func main() {
	w := flag.Int("width", 20, "grid width")
	h := flag.Int("height", 20, "grid height")
	mode := flag.String("mode", "balanced", "fast|balanced|quality|exact")
	budget := flag.Uint64("work-budget", 0, "0 means automatic")
	flag.Parse()
	g, err := traffic.Grid(*w, *h, 1)
	if err != nil {
		panic(err)
	}
	r := core.RouteRequest{Source: 0, Target: core.NodeID(*w**h - 1), Mode: core.RouteMode(*mode), Workers: 1}
	if *budget > 0 {
		r.WorkBudget = budget
	}
	res, err := gate.New(nil).Route(context.Background(), g, r)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(res)
}
