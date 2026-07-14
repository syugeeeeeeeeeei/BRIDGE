package traffic

import "testing"

func validScenario() BenchmarkScenario {
	s := BenchmarkScenario{SchemaVersion: BenchmarkSchemaV1, Suite: SuiteSpec{ID: "test"}, Execution: ExecutionSpec{Repetitions: 1, Seeds: []int64{1}}, Algorithms: []string{"bridge"}, Observation: ObservationSpec{Mode: "minimum"}, Scenarios: []ScenarioCase{{ID: "case", Graph: GeneratorSpec{Generator: "grid", Nodes: 5, Topology: "open"}, Queries: []QuerySpec{{ID: "default", Selection: QuerySelectionSpec{Method: "generator_default"}}}}}}
	s.ApplyDefaults()
	return s
}

func TestBuildScenarioGraphExactNodeCount(t *testing.T) {
	g, err := BuildScenarioGraph(GeneratorSpec{Generator: "grid", Nodes: 5, Topology: "open"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if g.NodeCount() != 5 {
		t.Fatalf("got %d nodes, want 5", g.NodeCount())
	}
}

func TestBuildScenarioGraphWallTopology(t *testing.T) {
	g, err := BuildScenarioGraph(GeneratorSpec{Generator: "grid", Nodes: 400, Topology: "wall", Noise: 0.05}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if g.NodeCount() == 0 {
		t.Fatal("expected non-empty graph")
	}
}

func TestBuildScenarioGraphRandomGeometric(t *testing.T) {
	g, err := BuildScenarioGraph(GeneratorSpec{Generator: "random_geometric", Nodes: 50, K: 6}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if g.NodeCount() != 50 {
		t.Fatalf("got %d nodes, want 50", g.NodeCount())
	}
}

func TestScenarioValidationRejectsUnsupportedJobs(t *testing.T) {
	s := validScenario()
	s.Execution.RunTimeout = "invalid"
	if err := s.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestScenarioValidationRejectsBadObservation(t *testing.T) {
	s := validScenario()
	s.Observation.Mode = "unknown"
	if err := s.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestScenarioValidationRejectsOutOfRangeEndpoint(t *testing.T) {
	s := validScenario()
	source, target := uint32(0), uint32(9)
	s.Scenarios[0].Queries = []QuerySpec{{ID: "default", Selection: QuerySelectionSpec{Method: "explicit", Source: &source, Target: &target}}}
	if err := s.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestScenarioValidationAcceptsRandomGeometric(t *testing.T) {
	s := validScenario()
	s.Scenarios[0].Graph = GeneratorSpec{Generator: "random_geometric", Nodes: 50, K: 6}
	if err := s.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScenarioValidationRejectsRandomGeometricWidthHeight(t *testing.T) {
	s := validScenario()
	s.Scenarios[0].Graph = GeneratorSpec{Generator: "random_geometric", Nodes: 50, Width: 10, Height: 5}
	if err := s.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestScenarioValidationRejectsWallWithWidthHeight(t *testing.T) {
	s := validScenario()
	s.Scenarios[0].Graph = GeneratorSpec{Generator: "grid", Nodes: 100, Width: 10, Height: 10, Topology: "wall"}
	if err := s.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestScenarioValidationRejectsRawOutputWithoutDir(t *testing.T) {
	s := validScenario()
	s.Output.Directory = ""
	if err := s.Validate(); err == nil {
		t.Fatal("expected error")
	}
}
