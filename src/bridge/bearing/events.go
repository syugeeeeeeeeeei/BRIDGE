package bearing

// EventKind is the canonical BEARING event vocabulary. Producers may add
// attributes, but MUST preserve the semantic meaning of these kinds.
type EventKind string

const (
	KindLifecycle             EventKind = "lifecycle"
	KindSearchStarted         EventKind = "search_started"
	KindSearchFinished        EventKind = "search_finished"
	KindComponentStarted      EventKind = "component_started"
	KindComponentFinished     EventKind = "component_finished"
	KindFrontierEnqueued      EventKind = "frontier_enqueued"
	KindFrontierSelected      EventKind = "frontier_selected"
	KindNodeExpanded          EventKind = "node_expanded"
	KindEdgeEvaluated         EventKind = "edge_evaluated"
	KindRelaxation            EventKind = "relaxation"
	KindBudgetExtended        EventKind = "budget_extended"
	KindCandidateSubmitted    EventKind = "candidate_submitted"
	KindIncumbentUpdated      EventKind = "incumbent_updated"
	KindConnectorStarted      EventKind = "connector_started"
	KindConnectorSucceeded    EventKind = "connector_succeeded"
	KindConnectorFailed       EventKind = "connector_failed"
	KindProgressReported      EventKind = "progress_reported"
	KindEmergencyReported     EventKind = "emergency_reported"
	KindDirectiveIssued       EventKind = "directive_issued"
	KindFallbackStarted       EventKind = "fallback_started"
	KindFallbackFinished      EventKind = "fallback_finished"
	KindCertificationStarted  EventKind = "certification_started"
	KindCertificationFinished EventKind = "certification_finished"
	KindStateReuseStarted     EventKind = "state_reuse_started"
	KindStateReuseApplied     EventKind = "state_reuse_applied"
	KindStateReuseRejected    EventKind = "state_reuse_rejected"
	KindStateReuseFinished    EventKind = "state_reuse_finished"
	KindAction                EventKind = "action"
)

const (
	ClassLifecycle = "lifecycle"
	ClassControl   = "control"
	ClassCandidate = "candidate"
	ClassDetail    = "detail"
	ClassProfile   = "profile"
)

// ClassifyEvent defines which events are retained by each ULTRASOUND mode.
func ClassifyEvent(kind string) string {
	switch EventKind(kind) {
	case KindLifecycle, KindSearchStarted, KindSearchFinished, KindComponentStarted, KindComponentFinished:
		return ClassLifecycle
	case KindBudgetExtended, KindProgressReported, KindEmergencyReported, KindDirectiveIssued,
		KindFallbackStarted, KindFallbackFinished, KindCertificationStarted, KindCertificationFinished,
		KindStateReuseStarted, KindStateReuseApplied, KindStateReuseRejected, KindStateReuseFinished:
		return ClassControl
	case KindCandidateSubmitted, KindIncumbentUpdated, KindConnectorSucceeded, KindConnectorFailed:
		return ClassCandidate
	case KindAction:
		return ClassProfile
	default:
		return ClassDetail
	}
}
