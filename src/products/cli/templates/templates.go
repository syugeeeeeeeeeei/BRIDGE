package templates

import (
	"fmt"
	"strings"
)

type CommentLevel int

const (
	None CommentLevel = iota
	Summary
	Full
)

func ParseLevel(s string) (CommentLevel, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "none":
		return None, nil
	case "summary":
		return Summary, nil
	case "full", "":
		return Full, nil
	default:
		return None, fmt.Errorf("comments must be one of: full, summary, none")
	}
}

func comments(level CommentLevel, summary []string, full []string) string {
	if level == None {
		return ""
	}
	lines := append([]string{}, summary...)
	if level >= Full {
		lines = append(lines, full...)
	}
	var b strings.Builder
	for _, line := range lines {
		b.WriteString("# ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func Server(level CommentLevel) string {
	var b strings.Builder
	b.WriteString(comments(level, []string{"BRIDGE HTTPサーバー設定です。"}, []string{"bridge serve --config bridge-server.yaml で読み込みます。"}))
	b.WriteString("schema_version: bridge.server.config.v1\n\nserver:\n")
	b.WriteString(indent(comments(level, []string{"接続を受け付けるアドレスです。"}, []string{"既定値はローカル端末からのみ接続できます。外部公開時は認証・TLS・ネットワーク制限を構成してください。"}), 2))
	b.WriteString("  listen: 127.0.0.1:8080\n")
	b.WriteString(indent(comments(level, []string{"1リクエストの最大処理時間です。"}, []string{"探索側の期限より短い場合、この値が実効上限になります。"}), 2))
	b.WriteString("  request_timeout: 30s\n")
	b.WriteString(indent(comments(level, []string{"終了時に実行中リクエストを待つ最大時間です。"}, nil), 2))
	b.WriteString("  shutdown_timeout: 10s\n\nlimits:\n")
	b.WriteString(indent(comments(level, []string{"HTTP本文の最大サイズです。単位はバイトです。"}, nil), 2))
	b.WriteString("  max_request_bytes: 16777216\n")
	b.WriteString(indent(comments(level, []string{"同時に処理する探索リクエスト数です。"}, []string{"値を大きくするとCPUとメモリの競合が増加します。"}), 2))
	b.WriteString("  max_concurrent_requests: 4\n")
	b.WriteString("  max_nodes: 1000000\n  max_edges: 10000000\n  max_logical_workers: 16\n  max_work_budget: 100000000\n\nlogging:\n")
	b.WriteString(indent(comments(level, []string{"ログ詳細度です。debug, info, warn, errorを指定できます。"}, nil), 2))
	b.WriteString("  level: info\n  format: text\n")
	return b.String()
}

func Scenario(preset string, level CommentLevel) (string, error) {
	preset = strings.ToLower(strings.TrimSpace(preset))
	if preset == "" {
		preset = "minimal"
	}
	nodes, reps, obs := 100, 1, "minimum"
	algs := []string{"bridge"}
	switch preset {
	case "minimal":
	case "comparison":
		nodes, reps, algs = 1000, 5, []string{"dijkstra", "bidirectional_dijkstra", "astar", "weighted_astar", "anchor", "bridge"}
	case "scalability":
		nodes, reps, algs = 10000, 3, []string{"dijkstra", "astar", "anchor", "bridge"}
	case "anytime":
		nodes, reps, algs = 2000, 5, []string{"weighted_astar", "anchor", "bridge"}
	case "trace":
		nodes, reps, obs = 1000, 1, "trace"
	default:
		return "", fmt.Errorf("unknown preset %q", preset)
	}
	var b strings.Builder
	b.WriteString(comments(level, []string{"BRIDGEベンチマークScenarioです。"}, []string{"生成後に bridge scenario validate と bridge scenario inspect で確認できます。"}))
	b.WriteString("schema_version: bridge.benchmark.v1\n\nsuite:\n  id: " + preset + "\n")
	b.WriteString(indent(comments(level, []string{"計測回数と再現用seedです。"}, []string{"warmup_runsは正式計測へ含まれません。"}), 0))
	b.WriteString(fmt.Sprintf("execution:\n  repetitions: %d\n  warmup_runs: 1\n  seeds: [42]\n\nalgorithms:\n", reps))
	for _, a := range algs {
		b.WriteString("  - " + a + "\n")
	}
	b.WriteString("\nobservation:\n")
	b.WriteString(indent(comments(level, []string{"収集する観測情報の詳細度です。"}, []string{"traceは成果物容量を増加させます。"}), 2))
	b.WriteString("  mode: " + obs + "\n\noutput:\n  directory: ./artifacts\n\nscenarios:\n  - id: " + preset + "-grid\n    graph:\n      generator: grid\n")
	b.WriteString(fmt.Sprintf("      requested_node_count: %d\n", nodes))
	b.WriteString("      topology: open\n    queries:\n      - id: default\n        selection:\n          method: generator_default\n    route:\n      mode: balanced\n      logical_worker_count: 1\n    budget:\n      work_limit: 10000000\n")
	return b.String(), nil
}

func Presets() []string { return []string{"minimal", "comparison", "scalability", "anytime", "trace"} }
func indent(s string, n int) string {
	if s == "" {
		return ""
	}
	p := strings.Repeat(" ", n)
	return p + strings.ReplaceAll(strings.TrimSuffix(s, "\n"), "\n", "\n"+p) + "\n"
}
