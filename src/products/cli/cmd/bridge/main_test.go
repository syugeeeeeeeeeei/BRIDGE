package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"version"}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("code=%d err=%s", code, errOut.String())
	}
	if strings.TrimSpace(out.String()) != "0.14.3" {
		t.Fatalf("version=%q", out.String())
	}
}

func TestRouteStdin(t *testing.T) {
	req := `{"schema_version":"bridge.route.request.v2","graph":{"type":"inline","nodes":[{"id":0},{"id":1}],"edges":[{"from":0,"to":1,"weight":1}]},"route":{"source":0,"target":1}}`
	var out, errOut bytes.Buffer
	if code := run([]string{"route"}, strings.NewReader(req), &out, &errOut); code != 0 {
		t.Fatalf("code=%d err=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"status": "found"`) {
		t.Fatalf("out=%s", out.String())
	}
}

func TestBenchmarkValidateRejectsInvalidScenario(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	data := `{"schema_version":"bridge.benchmark.v3","suite":{"id":"x"},"execution":{"repetitions":1,"seeds":[1]},"algorithms":["bridge"],"observation":{"mode":"invalid"},"output":{"directory":"artifacts"},"scenarios":[{"id":"c","graph":{"generator":"grid","requested_node_count":5,"topology":"open"},"queries":[{"id":"default","selection":{"method":"generator_default"}}]}]}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"benchmark", "validate", path}, strings.NewReader(""), &out, &errOut); code != exitUsage {
		t.Fatalf("code=%d err=%s", code, errOut.String())
	}
}

func TestBenchmarkRunReturnsSuccessWithoutTrafficSideAcceptance(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "run.json")
	data := `{"schema_version":"bridge.benchmark.v3","suite":{"id":"x"},"execution":{"repetitions":1,"seeds":[1]},"algorithms":["bridge"],"observation":{"mode":"minimum"},"output":{"directory":"artifacts"},"scenarios":[{"id":"c","graph":{"generator":"grid","requested_node_count":5,"topology":"open"},"queries":[{"id":"default","selection":{"method":"generator_default"}}]}]}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"benchmark", path}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("code=%d err=%s out=%s", code, errOut.String(), out.String())
	}
	if !strings.Contains(errOut.String(), "run=") {
		t.Fatalf("expected progress output, got %s", errOut.String())
	}
}

func TestRouteTraceOutput(t *testing.T) {
	dir := t.TempDir()
	trace := filepath.Join(dir, "trace.jsonl")
	req := `{"schema_version":"bridge.route.request.v2","graph":{"type":"inline","nodes":[{"id":0},{"id":1}],"edges":[{"from":0,"to":1,"weight":1}]},"route":{"source":0,"target":1},"observation_config":{"level":"trace"}}`
	var out, errOut bytes.Buffer
	if code := run([]string{"route", "--trace-output", trace}, strings.NewReader(req), &out, &errOut); code != 0 {
		t.Fatalf("code=%d err=%s", code, errOut.String())
	}
	b, err := os.ReadFile(trace)
	if err != nil {
		t.Fatal(err)
	}
	if len(bytes.TrimSpace(b)) == 0 {
		t.Fatal("trace file is empty")
	}
	if !strings.Contains(out.String(), `"observation_data"`) {
		t.Fatalf("out=%s", out.String())
	}
}

func TestRouteDoesNotCreateTraceImplicitly(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer os.Chdir(old)
	req := `{"schema_version":"bridge.route.request.v2","graph":{"type":"inline","nodes":[{"id":0},{"id":1}],"edges":[{"from":0,"to":1,"weight":1}]},"route":{"source":0,"target":1},"observation_config":{"level":"trace"}}`
	var out, errOut bytes.Buffer
	if code := run([]string{"route"}, strings.NewReader(req), &out, &errOut); code != 0 {
		t.Fatalf("code=%d err=%s", code, errOut.String())
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("unexpected files: %v", entries)
	}
}

func TestBenchmarkRunCreatesArchiveInOutputDir(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "benchmark-comparison-heavy-v2")
	path := filepath.Join(dir, "bench.json")
	data := `{
  "schema_version":"bridge.benchmark.v3",
  "suite":{"id":"zip-test"},
  "execution":{"repetitions":1,"seeds":[1]},
  "algorithms":["bridge"],
  "observation":{"mode":"minimum"},
  "output":{"directory":"` + filepath.ToSlash(outputDir) + `"},
  "scenarios":[{"id":"c","graph":{"generator":"grid","requested_node_count":5,"topology":"open"},"queries":[{"id":"default","selection":{"method":"generator_default"}}]}]
}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"benchmark", path}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("code=%d err=%s out=%s", code, errOut.String(), out.String())
	}
	entries, err := os.ReadDir(filepath.Join(outputDir, "zip-test"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		t.Fatalf("execution directory missing: %v", entries)
	}
}
