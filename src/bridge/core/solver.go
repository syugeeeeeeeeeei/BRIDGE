package core

import (
	"math"
)

type WorkBudget struct {
	MaxWork   *uint64
	MaxExpand *uint64
}
type SolverProgress struct {
	TaskID          string
	WorkUsed        uint64
	ElapsedMS       float64
	Found           bool
	BestDistance    *float64
	LowerBound      *float64
	CandidateCount  int
	StagnationScore float64
	Finished        bool
	FailureReason   string
}
type SolverTask struct {
	ID, SolverKind, Purpose string
	Budget                  WorkBudget
	Workers                 int
	QualityTarget           float64
	Parameters              map[string]string
}
type TaskTrace struct {
	TaskID, Solver, Purpose, Reason string
	Allocation                      float64
	Budget                          *uint64
	Found                           bool
	Distance                        float64
	WorkUsed                        uint64
}

func Euclidean(a, b Point) float64 { return math.Hypot(a.X-b.X, a.Y-b.Y) }
func PathDistance(g Graph, path []NodeID) float64 {
	if len(path) == 0 {
		return math.Inf(1)
	}
	total := 0.0
	for i := 0; i+1 < len(path); i++ {
		ok := false
		for _, e := range g.EdgesFrom(path[i]) {
			if e.To == path[i+1] {
				total += e.Weight
				ok = true
				break
			}
		}
		if !ok {
			return math.Inf(1)
		}
	}
	return total
}
