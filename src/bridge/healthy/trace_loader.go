package healthy

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
)

type traceManifest struct {
	SampleRate  float64 `json:"sample_rate"`
	Dropped     uint64  `json:"dropped_event_count"`
	Truncated   bool    `json:"truncated"`
	TraceSHA256 string  `json:"trace_sha256"`
}

func loadTrace(manifestPath, tracePath string) ([]bearing.Event, traceManifest, error) {
	var m traceManifest
	if manifestPath == "" || tracePath == "" {
		return nil, m, fmt.Errorf("trace paths are missing")
	}
	b, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, m, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, m, err
	}
	tb, err := os.ReadFile(tracePath)
	if err != nil {
		return nil, m, err
	}
	sum := sha256.Sum256(tb)
	if m.TraceSHA256 != "" && hex.EncodeToString(sum[:]) != m.TraceSHA256 {
		return nil, m, fmt.Errorf("trace sha256 mismatch")
	}
	f, err := os.Open(tracePath)
	if err != nil {
		return nil, m, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 8*1024*1024)
	var events []bearing.Event
	for sc.Scan() {
		var e bearing.Event
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			return nil, m, err
		}
		events = append(events, e)
	}
	if err := sc.Err(); err != nil {
		return nil, m, err
	}
	return events, m, nil
}
