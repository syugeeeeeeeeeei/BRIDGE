package traffic

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

const DatasetSchemaV1 = "bridge.dataset.v1"

type DatasetDocument struct {
	SchemaVersion string              `json:"schema_version"`
	ID            string              `json:"id"`
	Source        string              `json:"source"`
	License       string              `json:"license"`
	Directed      bool                `json:"directed"`
	Nodes         int                 `json:"nodes"`
	Edges         []DatasetEdge       `json:"edges"`
	Positions     []DatasetPosition   `json:"positions,omitempty"`
	DefaultQuery  *DatasetQuery       `json:"default_query,omitempty"`
	Preprocessing []PreprocessingStep `json:"preprocessing,omitempty"`
}

type DatasetEdge struct {
	From   uint32  `json:"from"`
	To     uint32  `json:"to"`
	Weight float64 `json:"weight"`
}
type DatasetPosition struct {
	Node uint32  `json:"node"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}
type DatasetQuery struct {
	Source uint32 `json:"source"`
	Target uint32 `json:"target"`
}
type PreprocessingStep struct {
	Name       string            `json:"name"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

type DatasetMetadata struct {
	ID            string              `json:"id"`
	Source        string              `json:"source"`
	License       string              `json:"license"`
	Path          string              `json:"path"`
	SHA256        string              `json:"sha256"`
	Preprocessing []PreprocessingStep `json:"preprocessing,omitempty"`
}

type LoadedDataset struct {
	Graph    *core.AdjacencyGraph
	Source   core.NodeID
	Target   core.NodeID
	Metadata DatasetMetadata
}

func LoadDataset(path string) (LoadedDataset, error) {
	if path == "" {
		return LoadedDataset{}, fmt.Errorf("dataset path is required")
	}
	clean := filepath.Clean(path)
	payload, err := os.ReadFile(clean)
	if err != nil {
		return LoadedDataset{}, fmt.Errorf("read dataset: %w", err)
	}
	var doc DatasetDocument
	if err := json.Unmarshal(payload, &doc); err != nil {
		return LoadedDataset{}, fmt.Errorf("decode dataset: %w", err)
	}
	if doc.SchemaVersion != DatasetSchemaV1 {
		return LoadedDataset{}, fmt.Errorf("schema_version must be %q", DatasetSchemaV1)
	}
	if doc.ID == "" || doc.Source == "" || doc.License == "" {
		return LoadedDataset{}, fmt.Errorf("dataset id, source, and license are required")
	}
	if doc.Nodes < 2 {
		return LoadedDataset{}, fmt.Errorf("dataset nodes must be >=2")
	}
	edges := append([]DatasetEdge(nil), doc.Edges...)
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		return edges[i].Weight < edges[j].Weight
	})
	g := core.NewAdjacencyGraph(doc.Nodes, doc.Directed)
	for _, e := range edges {
		if int(e.From) >= doc.Nodes || int(e.To) >= doc.Nodes {
			return LoadedDataset{}, fmt.Errorf("edge endpoint outside node range")
		}
		if err := g.AddEdge(core.NodeID(e.From), core.NodeID(e.To), e.Weight); err != nil {
			return LoadedDataset{}, err
		}
	}
	for _, p := range doc.Positions {
		if int(p.Node) >= doc.Nodes {
			return LoadedDataset{}, fmt.Errorf("position node outside node range")
		}
		if err := g.SetPosition(core.NodeID(p.Node), core.Point{X: p.X, Y: p.Y}); err != nil {
			return LoadedDataset{}, err
		}
	}
	source, target := core.NodeID(0), core.NodeID(doc.Nodes-1)
	if doc.DefaultQuery != nil {
		if int(doc.DefaultQuery.Source) >= doc.Nodes || int(doc.DefaultQuery.Target) >= doc.Nodes {
			return LoadedDataset{}, fmt.Errorf("default query outside node range")
		}
		source, target = core.NodeID(doc.DefaultQuery.Source), core.NodeID(doc.DefaultQuery.Target)
	}
	sum := sha256.Sum256(payload)
	return LoadedDataset{Graph: g, Source: source, Target: target, Metadata: DatasetMetadata{ID: doc.ID, Source: doc.Source, License: doc.License, Path: clean, SHA256: hex.EncodeToString(sum[:]), Preprocessing: doc.Preprocessing}}, nil
}
