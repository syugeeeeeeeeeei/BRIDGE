package yamlmini

import (
	"encoding/json"
	"testing"
)

func TestToJSON(t *testing.T) {
	b, err := ToJSON([]byte("schema_version: bridge.benchmark.v1\nalgorithms:\n  - bridge\nexecution:\n  seeds: [1, 2]\n"))
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	if v["schema_version"] != "bridge.benchmark.v1" {
		t.Fatalf("%v", v)
	}
}
func TestDuplicateKey(t *testing.T) {
	if _, err := ToJSON([]byte("a: 1\na: 2\n")); err == nil {
		t.Fatal("expected error")
	}
}
