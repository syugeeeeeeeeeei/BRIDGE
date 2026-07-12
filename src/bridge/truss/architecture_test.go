package truss

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestArchitectureDependencies(t *testing.T) {
	root := ".."
	allowed := map[string]map[string]bool{
		"bearing":    {"core": true},
		"anchor":     {"core": true, "bearing": true},
		"bolts":      {"core": true, "bearing": true},
		"truss":      {"core": true, "anchor": true, "bolts": true, "bearing": true},
		"gate":       {"core": true, "truss": true, "bearing": true},
		"ultrasound": {"core": true, "bearing": true},
		"traffic":    {"core": true, "gate": true, "ultrasound": true},
	}
	fset := token.NewFileSet()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 2 {
			return nil
		}
		owner := parts[0]
		rules, ok := allowed[owner]
		if !ok {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			raw, _ := strconv.Unquote(imp.Path.Value)
			const prefix = "github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/"
			if !strings.HasPrefix(raw, prefix) {
				continue
			}
			dep := strings.Split(strings.TrimPrefix(raw, prefix), "/")[0]
			if dep == owner {
				continue
			}
			if !rules[dep] {
				t.Errorf("forbidden dependency: %s -> %s in %s", owner, dep, path)
			}
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), "others/legacy/"+"bridge_py") {
			t.Errorf("forbidden legacy dependency in %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBearingOwnsNoPersistenceImplementation(t *testing.T) {
	entries, err := os.ReadDir("../bearing")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if strings.Contains(name, "sink") || strings.Contains(name, "collector") || strings.Contains(name, "recorder") {
			t.Errorf("BEARING must not own persistence/collection implementation: %s", name)
		}
	}
}
