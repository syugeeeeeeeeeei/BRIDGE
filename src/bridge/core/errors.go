package core

type ErrorCode string

const (
	ErrInvalidRequest   ErrorCode = "INVALID_REQUEST"
	ErrNoPath           ErrorCode = "NO_PATH"
	ErrBudgetExhausted  ErrorCode = "BUDGET_EXHAUSTED"
	ErrDeadlineExceeded ErrorCode = "DEADLINE_EXCEEDED"
	ErrCancelled        ErrorCode = "CANCELLED"
)
