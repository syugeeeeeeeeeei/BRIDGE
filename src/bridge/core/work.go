package core

import (
	"fmt"
)

type WorkAction string

const (
	WorkSelect           WorkAction = "select"
	WorkExpand           WorkAction = "expand"
	WorkEvaluate         WorkAction = "evaluate"
	WorkRelax            WorkAction = "relax"
	WorkEnqueue          WorkAction = "enqueue"
	WorkReject           WorkAction = "reject"
	WorkBacktrack        WorkAction = "backtrack"
	WorkConnect          WorkAction = "connect"
	WorkCandidate        WorkAction = "candidate"
	WorkRepair           WorkAction = "repair"
	WorkBound            WorkAction = "bound"
	WorkTerminate        WorkAction = "terminate"
	WorkHypothesisAction WorkAction = "hypothesis"
	WorkEvidenceAction   WorkAction = "evidence"
	WorkHandoffAction    WorkAction = "handoff"
	WorkScheduleAction   WorkAction = "schedule"
)

func IsWorkAction(kind string) bool {
	switch WorkAction(kind) {
	case WorkSelect, WorkExpand, WorkEvaluate, WorkRelax, WorkEnqueue, WorkReject, WorkBacktrack, WorkConnect, WorkCandidate, WorkRepair, WorkBound, WorkTerminate, WorkHypothesisAction, WorkEvidenceAction, WorkHandoffAction, WorkScheduleAction:
		return true
	default:
		return false
	}
}

type WorkMetrics struct {
	TotalActions      uint64 `json:"total_actions"`
	SelectActions     uint64 `json:"select_actions"`
	ExpandActions     uint64 `json:"expand_actions"`
	EvaluateActions   uint64 `json:"evaluate_actions"`
	RelaxActions      uint64 `json:"relax_actions"`
	EnqueueActions    uint64 `json:"enqueue_actions"`
	RejectActions     uint64 `json:"reject_actions"`
	BacktrackActions  uint64 `json:"backtrack_actions"`
	ConnectActions    uint64 `json:"connect_actions"`
	CandidateActions  uint64 `json:"candidate_actions"`
	RepairActions     uint64 `json:"repair_actions"`
	BoundActions      uint64 `json:"bound_actions"`
	TerminateActions  uint64 `json:"terminate_actions"`
	HypothesisActions uint64 `json:"hypothesis_actions"`
	EvidenceActions   uint64 `json:"evidence_actions"`
	HandoffActions    uint64 `json:"handoff_actions"`
	ScheduleActions   uint64 `json:"schedule_actions"`
	LogicalSteps      uint64 `json:"logical_steps"`
	ScheduledSteps    uint64 `json:"scheduled_steps"`
	WorkerCount       uint32 `json:"logical_worker_count"`
}

func (w *WorkMetrics) AddAction(kind string) {
	if !IsWorkAction(kind) {
		panic("unknown Work action: " + kind)
	}
	w.TotalActions++
	switch kind {
	case "select":
		w.SelectActions++
	case "expand":
		w.ExpandActions++
	case "evaluate":
		w.EvaluateActions++
	case "relax":
		w.RelaxActions++
	case "enqueue":
		w.EnqueueActions++
	case "reject":
		w.RejectActions++
	case "backtrack":
		w.BacktrackActions++
	case "connect":
		w.ConnectActions++
	case "candidate":
		w.CandidateActions++
	case "repair":
		w.RepairActions++
	case "bound":
		w.BoundActions++
	case "terminate":
		w.TerminateActions++
	case "hypothesis":
		w.HypothesisActions++
	case "evidence":
		w.EvidenceActions++
	case "handoff":
		w.HandoffActions++
	case "schedule":
		w.ScheduleActions++
	}
}

func (w *WorkMetrics) Add(other WorkMetrics) {
	w.TotalActions += other.TotalActions
	w.SelectActions += other.SelectActions
	w.ExpandActions += other.ExpandActions
	w.EvaluateActions += other.EvaluateActions
	w.RelaxActions += other.RelaxActions
	w.EnqueueActions += other.EnqueueActions
	w.RejectActions += other.RejectActions
	w.BacktrackActions += other.BacktrackActions
	w.ConnectActions += other.ConnectActions
	w.CandidateActions += other.CandidateActions
	w.RepairActions += other.RepairActions
	w.BoundActions += other.BoundActions
	w.TerminateActions += other.TerminateActions
	w.HypothesisActions += other.HypothesisActions
	w.EvidenceActions += other.EvidenceActions
	w.HandoffActions += other.HandoffActions
	w.ScheduleActions += other.ScheduleActions
	w.LogicalSteps += other.LogicalSteps
	w.ScheduledSteps += other.ScheduledSteps
	if other.WorkerCount > w.WorkerCount {
		w.WorkerCount = other.WorkerCount
	}
}

func (w WorkMetrics) CountedActions() uint64 {
	return w.SelectActions + w.ExpandActions + w.EvaluateActions + w.RelaxActions + w.EnqueueActions + w.RejectActions + w.BacktrackActions + w.ConnectActions + w.CandidateActions + w.RepairActions + w.BoundActions + w.TerminateActions + w.HypothesisActions + w.EvidenceActions + w.HandoffActions + w.ScheduleActions
}

func (w WorkMetrics) ValidationErrors() []string {
	errs := []string{}
	if w.TotalActions != w.CountedActions() {
		errs = append(errs, fmt.Sprintf("total_actions=%d differs from action sum=%d", w.TotalActions, w.CountedActions()))
	}
	if w.LogicalSteps > w.ScheduledSteps {
		errs = append(errs, fmt.Sprintf("logical_steps=%d exceeds scheduled_steps=%d", w.LogicalSteps, w.ScheduledSteps))
	}
	if w.ScheduledSteps > w.TotalActions {
		errs = append(errs, fmt.Sprintf("scheduled_steps=%d exceeds total_actions=%d", w.ScheduledSteps, w.TotalActions))
	}
	return errs
}

func (w WorkMetrics) Valid() bool {
	return w.TotalActions == w.CountedActions() && w.LogicalSteps <= w.ScheduledSteps && w.ScheduledSteps <= w.TotalActions
}

// TimeBreakdown records measured execution phases. These durations are not Work.
