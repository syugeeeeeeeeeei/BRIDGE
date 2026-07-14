package core

type TerminationStatus string

const (
	TerminationRunning       TerminationStatus = "RUNNING"
	TerminationFound         TerminationStatus = "FOUND"
	TerminationUnreachable   TerminationStatus = "UNREACHABLE"
	TerminationUnknownBudget TerminationStatus = "UNKNOWN_BUDGET"
	TerminationCancelled     TerminationStatus = "CANCELLED"
	TerminationDeadline      TerminationStatus = "DEADLINE_EXCEEDED"
	TerminationInvalid       TerminationStatus = "INVALID_REQUEST"
)
