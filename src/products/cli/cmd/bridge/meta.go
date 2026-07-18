package main

import (
	"encoding/json"
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/buildinfo"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func schemaCommand(args []string, stdout, stderr io.Writer) int {
	root := "src/contracts/json-schema"
	if len(args) == 0 || args[0] == "list" {
		entries, err := os.ReadDir(root)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return exitIO
		}
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".schema.json") {
				fmt.Fprintln(stdout, strings.TrimSuffix(e.Name(), ".schema.json"))
			}
		}
		return 0
	}
	if args[0] == "show" && len(args) == 2 {
		b, err := os.ReadFile(filepath.Join(root, args[1]+".schema.json"))
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return exitIO
		}
		_, _ = stdout.Write(b)
		return 0
	}
	return exitUsage
}
func capabilities(args []string, stdout, stderr io.Writer) int {
	if len(args) > 2 {
		return exitUsage
	}
	if err := json.NewEncoder(stdout).Encode(map[string]any{"application_version": buildinfo.Version, "api_versions": []string{"v1"}, "schemas": map[string][]string{"route_request": {gate.RouteRequestSchemaV1}, "route_response": {gate.RouteResultSchemaV1}}, "features": map[string]bool{"route": true, "serve": true, "benchmark": true, "trace": true}, "algorithms": []string{"bridge", "anchor", "dijkstra", "bidirectional_dijkstra", "astar", "weighted_astar"}}); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	return 0
}
func completion(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "error: shell is required")
		return exitUsage
	}
	switch args[0] {
	case "bash":
		fmt.Fprintln(stdout, "complete -W 'route serve scenario benchmark artifact schema capabilities completion version help' bridge")
	case "zsh", "fish", "powershell":
		fmt.Fprintf(stdout, "# BRIDGE completion for %s\n", args[0])
	default:
		fmt.Fprintln(stderr, "error: unsupported shell")
		return exitUsage
	}
	return 0
}
func writeTextOutput(stdout, stderr io.Writer, path string, overwrite bool, text string) int {
	if path == "-" {
		_, _ = io.WriteString(stdout, text)
		return 0
	}
	out, closeFn, err := outputWriter(stdout, path, overwrite)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	defer closeFn()
	if _, err = io.WriteString(out, text); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	fmt.Fprintln(stderr, "created:", path)
	return 0
}
