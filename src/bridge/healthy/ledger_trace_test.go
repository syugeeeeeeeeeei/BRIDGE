package healthy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/core"
)

func TestValidateWorkWithLedger(t *testing.T) {
	var w core.WorkMetrics
	w.AddAction(string(core.WorkExpand))
	w.LogicalSteps, w.ScheduledSteps = 1, 1
	one := uint64(1)
	l := &core.BudgetLedger{Limit: &one, Used: 1, Remaining: func() *uint64 { v := uint64(0); return &v }(), ByComponent: map[core.Component]uint64{core.ComponentAnchor: 1}, Entries: []core.BudgetLedgerEntry{{TaskID: "a", Component: core.ComponentAnchor, Purpose: "first_path", Granted: &one, Used: 1}}}
	got := ValidateWorkWithLedger(w, nil, l)
	if !got.LedgerVerifiable || !got.LedgerValid || got.Status != StatusPass {
		t.Fatalf("unexpected validation: %+v", got)
	}
	l.Used = 2
	got = ValidateWorkWithLedger(w, nil, l)
	if got.Status != StatusInvalid {
		t.Fatalf("expected invalid ledger: %+v", got)
	}
}

func TestLoadTraceVerifiesDigestAndReconstructs(t *testing.T) {
	dir := t.TempDir()
	trace := filepath.Join(dir, "trace.jsonl")
	manifest := filepath.Join(dir, "manifest.json")
	e := bearing.Event{Kind: "action", Action: string(core.WorkExpand), LogicalStep: 1, ScheduledStep: 1}
	b, _ := json.Marshal(e)
	b = append(b, '\n')
	if err := os.WriteFile(trace, b, 0644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	m := map[string]any{"sample_rate": 1.0, "dropped_event_count": 0, "truncated": false, "trace_sha256": hex.EncodeToString(sum[:])}
	mb, _ := json.Marshal(m)
	if err := os.WriteFile(manifest, mb, 0644); err != nil {
		t.Fatal(err)
	}
	events, meta, err := loadTrace(manifest, trace)
	if err != nil {
		t.Fatal(err)
	}
	rec := ReconstructWork(events, meta.SampleRate, meta.Truncated, meta.Dropped)
	if !rec.Verifiable || rec.Work.ExpandActions != 1 || rec.Work.TotalActions != 1 {
		t.Fatalf("bad reconstruction: %+v", rec)
	}
}

