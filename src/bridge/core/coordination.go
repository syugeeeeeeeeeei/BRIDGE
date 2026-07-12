package core

// Component identifies one of BRIDGE's parallel major components.
type Component string

const (
	ComponentAnchor Component = "ANCHOR"
	ComponentBolts  Component = "BOLTS"
)

type ProgressState string

const (
	ProgressAdvancing ProgressState = "ADVANCING"
	ProgressStalled   ProgressState = "STALLED"
	ProgressExhausted ProgressState = "EXHAUSTED"
	ProgressCompleted ProgressState = "COMPLETED"
)

type EmergencyKind string

const (
	EmergencyStallDetected             EmergencyKind = "STALL_DETECTED"
	EmergencyDetourRequired            EmergencyKind = "DETOUR_REQUIRED"
	EmergencyFrontierExhausted         EmergencyKind = "FRONTIER_EXHAUSTED"
	EmergencyHeuristicUnreliable       EmergencyKind = "HEURISTIC_UNRELIABLE"
	EmergencyBudgetExhaustedProgress   EmergencyKind = "BUDGET_EXHAUSTED_WITH_PROGRESS"
	EmergencyBudgetExhaustedNoProgress EmergencyKind = "BUDGET_EXHAUSTED_NO_PROGRESS"
	EmergencyRepeatedRegion            EmergencyKind = "REPEATED_REGION"
	EmergencyHighDuplicateExpansion    EmergencyKind = "HIGH_DUPLICATE_EXPANSION"
	EmergencyEdgeScanExplosion         EmergencyKind = "EDGE_SCAN_EXPLOSION"
	EmergencyReachabilityUncertain     EmergencyKind = "REACHABILITY_UNCERTAIN"
	EmergencyIncumbentQualityPoor      EmergencyKind = "INCUMBENT_QUALITY_POOR"
)

type ProgressReport struct {
	Component             Component
	Phase                 string
	State                 ProgressState
	Work                  WorkMetrics
	UniqueInvestigated    uint64
	DuplicateInvestigated uint64
	InvestigatedEdges     uint64
	NodeRatio             float64
	EdgeRatio             float64
	Candidate             *RouteResult
}

type EmergencyReport struct {
	Component   Component
	Kind        EmergencyKind
	Evidence    map[string]any
	Recoverable bool
}

type DirectiveKind string

const (
	DirectiveContinue   DirectiveKind = "CONTINUE"
	DirectiveExtend     DirectiveKind = "EXTEND_BUDGET"
	DirectivePause      DirectiveKind = "PAUSE"
	DirectiveResume     DirectiveKind = "RESUME"
	DirectiveYield      DirectiveKind = "YIELD"
	DirectiveTerminate  DirectiveKind = "TERMINATE"
	DirectiveStartBolts DirectiveKind = "START_BOLTS"
)

type Directive struct {
	Kind       DirectiveKind
	Capability string
	Reason     string
	Budget     WorkBudget
}
