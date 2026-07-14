package core

type HandoffPurpose string

const (
	ConnectCheckpoints HandoffPurpose = "CONNECT_CHECKPOINTS"
	EscapeRegion       HandoffPurpose = "ESCAPE_REGION"
	RepairSegment      HandoffPurpose = "REPAIR_SEGMENT"
	ProveUnreachable   HandoffPurpose = "PROVE_UNREACHABLE"
	TightenBound       HandoffPurpose = "TIGHTEN_BOUND"
	CertifyCandidate   HandoffPurpose = "CERTIFY_CANDIDATE"
)

type HandoffRequest struct {
	ID           string
	Purpose      HandoffPurpose
	Region       Region
	Inputs       []Checkpoint
	Expected     string
	Evidence     []Evidence
	Budget       WorkBudget
	HypothesisID string
}
type HandoffResult struct {
	RequestID         string
	Path              []NodeID
	Distance          float64
	Found             bool
	Work              WorkMetrics
	Evidence          []Evidence
	ResumeCheckpoints []Checkpoint
}
