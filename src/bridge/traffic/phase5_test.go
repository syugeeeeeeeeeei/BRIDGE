package traffic

import (
	"context"
	"path/filepath"
	"testing"
)

func TestPhase5GeneratorsAreDeterministicAndConnected(t *testing.T) {
	for _, tc := range []struct {
		name      string
		makeGraph func() (int, int, error)
	}{
		{"community", func() (int, int, error) {
			g, _, _, e := CommunityGraph(40, 4, 7)
			if e != nil {
				return 0, 0, e
			}
			return g.NodeCount(), g.EdgeCount(), nil
		}},
		{"maze", func() (int, int, error) {
			g, _, _, e := MazeGraph(40, 7)
			if e != nil {
				return 0, 0, e
			}
			return g.NodeCount(), g.EdgeCount(), nil
		}},
		{"adversarial", func() (int, int, error) {
			g, _, _, e := AdversarialGraph(40, 7)
			if e != nil {
				return 0, 0, e
			}
			return g.NodeCount(), g.EdgeCount(), nil
		}},
	} {
		n1, e1, err := tc.makeGraph()
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		n2, e2, err := tc.makeGraph()
		if err != nil {
			t.Fatalf("%s repeat: %v", tc.name, err)
		}
		if n1 != 40 || n1 != n2 || e1 != e2 || e1 < 39 {
			t.Fatalf("%s non-deterministic/invalid: %d %d / %d %d", tc.name, n1, e1, n2, e2)
		}
	}
}

func TestLoadDatasetCapturesProvenance(t *testing.T) {
	path := filepath.Join("..", "..", "..", "tests", "datasets", "tiny-road-network.json")
	d, err := LoadDataset(path)
	if err != nil {
		t.Fatal(err)
	}
	if d.Graph.NodeCount() != 6 || d.Graph.EdgeCount() != 6 {
		t.Fatalf("unexpected graph size: %d/%d", d.Graph.NodeCount(), d.Graph.EdgeCount())
	}
	if d.Metadata.ID != "tiny-road-network" || d.Metadata.License == "" || len(d.Metadata.SHA256) != 64 || len(d.Metadata.Preprocessing) != 2 {
		t.Fatalf("missing provenance: %+v", d.Metadata)
	}
}

func TestDatasetScenarioPersistsMetadata(t *testing.T) {
	path := filepath.Join("..", "..", "..", "tests", "datasets", "tiny-road-network.json")
	s := BenchmarkScenario{SchemaVersion: BenchmarkSchemaV1, Suite: SuiteSpec{ID: "phase5-dataset"}, Execution: ExecutionSpec{Repetitions: 1, Seeds: []int64{1}, Jobs: 1}, Algorithms: []string{"dijkstra"}, Observation: ObservationSpec{Mode: "off", SampleRate: 1}, Scenarios: []ScenarioCase{{ID: "dataset", Graph: GeneratorSpec{Generator: "dataset", DatasetPath: path, DatasetFormat: "bridge.dataset.v1.json"}, Queries: []QuerySpec{{ID: "default", Strategy: "opposite-corners"}}, Route: RouteSpec{Mode: "exact", Workers: 1}}}}
	result, err := RunScenario(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RawRuns) != 1 || result.RawRuns[0].Graph.Dataset == nil {
		t.Fatalf("dataset metadata missing: %+v", result.RawRuns)
	}
	if result.RawRuns[0].Graph.Dataset.License != "CC0-1.0" {
		t.Fatalf("license missing: %+v", result.RawRuns[0].Graph.Dataset)
	}
}
