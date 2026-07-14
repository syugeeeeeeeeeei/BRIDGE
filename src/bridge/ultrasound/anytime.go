package ultrasound

import (
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
	"math"
	"sort"
)

type AnytimePoint struct {
	Work  uint64   `json:"work"`
	Upper *float64 `json:"upper,omitempty"`
	Lower *float64 `json:"lower,omitempty"`
	Ratio *float64 `json:"ratio,omitempty"`
}
type ReuseMetrics struct {
	StateReuseRatio    float64 `json:"state_reuse_ratio"`
	DuplicateWorkRatio float64 `json:"duplicate_work_ratio"`
	Reused             uint64  `json:"reused"`
	Duplicate          uint64  `json:"duplicate"`
	Total              uint64  `json:"total"`
}

func AnytimeCurve(events []bearing.Event) []AnytimePoint {
	upper := math.Inf(1)
	lower := 0.0
	points := []AnytimePoint{}
	ordered := append([]bearing.Event(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].WorkAfter != ordered[j].WorkAfter {
			return ordered[i].WorkAfter < ordered[j].WorkAfter
		}
		return ordered[i].Kind < ordered[j].Kind
	})
	for _, e := range ordered {
		changed := false
		if v, ok := number(e.Attributes["upper_bound"]); ok && v < upper {
			upper = v
			changed = true
		}
		if v, ok := number(e.Attributes["lower_bound"]); ok && v > lower {
			lower = v
			changed = true
		}
		if !changed {
			continue
		}
		p := AnytimePoint{Work: e.WorkAfter}
		if !math.IsInf(upper, 1) {
			u := upper
			p.Upper = &u
		}
		if lower > 0 {
			l := lower
			p.Lower = &l
			if p.Upper != nil {
				r := upper / lower
				p.Ratio = &r
			}
		}
		points = append(points, p)
	}
	return points
}
func ComputeReuse(events []bearing.Event) ReuseMetrics {
	var m ReuseMetrics
	for _, e := range events {
		if e.Kind != "action" {
			continue
		}
		m.Total++
		if b, _ := e.Attributes["reused"].(bool); b {
			m.Reused++
		}
		if b, _ := e.Attributes["duplicate"].(bool); b {
			m.Duplicate++
		}
	}
	if m.Total > 0 {
		m.StateReuseRatio = float64(m.Reused) / float64(m.Total)
		m.DuplicateWorkRatio = float64(m.Duplicate) / float64(m.Total)
	}
	return m
}
func number(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case uint64:
		return float64(x), true
	case int64:
		return float64(x), true
	}
	return 0, false
}
