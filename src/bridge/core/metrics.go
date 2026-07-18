package core

type TimeBreakdown struct {
	// Nanosecond fields are primary measurements. Millisecond fields are derived compatibility views.
	TotalNS         int64   `json:"total_ns"`
	SolverNS        int64   `json:"solver_ns"`
	TrussNS         int64   `json:"truss_ns,omitempty"`
	GateNS          int64   `json:"gate_ns,omitempty"`
	AnchorNS        int64   `json:"anchor_ns,omitempty"`
	BoltsNS         int64   `json:"bolts_ns,omitempty"`
	FallbackNS      int64   `json:"fallback_ns,omitempty"`
	SupervisorNS    int64   `json:"supervisor_ns,omitempty"`
	ArbiterNS       int64   `json:"arbiter_ns,omitempty"`
	OrchestrationNS int64   `json:"orchestration_ns,omitempty"`
	TotalMS         float64 `json:"total_ms"`
	SolverMS        float64 `json:"solver_ms"`
	TrussMS         float64 `json:"truss_ms,omitempty"`
	AnchorMS        float64 `json:"anchor_ms,omitempty"`
	BoltsMS         float64 `json:"bolts_ms,omitempty"`
	FallbackMS      float64 `json:"fallback_ms,omitempty"`
	SupervisorMS    float64 `json:"supervisor_ms,omitempty"`
	ArbiterMS       float64 `json:"arbiter_ms,omitempty"`
	OrchestrationMS float64 `json:"orchestration_ms,omitempty"`
	GateMS          float64 `json:"gate_ms,omitempty"`
}

// SystemMetrics records benchmark-process deltas. They are observational overhead, not search Work.
type SystemMetrics struct {
	AllocBytes           uint64 `json:"alloc_bytes"`
	MallocCount          uint64 `json:"malloc_count"`
	GCCount              uint32 `json:"gc_count"`
	HeapAllocBefore      uint64 `json:"heap_alloc_before"`
	HeapAllocAfter       uint64 `json:"heap_alloc_after"`
	HeapAllocBoundaryMax uint64 `json:"heap_alloc_boundary_max"`
	HeapAllocSampledPeak uint64 `json:"heap_alloc_sampled_peak,omitempty"`
}

// BudgetLedgerEntry records one solver task allocation and consumption.
// It is accounting data, not search Work itself.
type BudgetLedgerEntry struct {
	TaskID    string    `json:"task_id"`
	Component Component `json:"component"`
	Purpose   string    `json:"purpose"`
	Granted   *uint64   `json:"granted,omitempty"`
	Used      uint64    `json:"used"`
}

// BudgetLedger is the public accounting snapshot for a TRUSS execution.
type BudgetLedger struct {
	Limit       *uint64              `json:"limit,omitempty"`
	Used        uint64               `json:"used"`
	Remaining   *uint64              `json:"remaining,omitempty"`
	ByComponent map[Component]uint64 `json:"by_component"`
	Entries     []BudgetLedgerEntry  `json:"entries"`
}

// HandoffRecord captures one ANCHOR-to-BOLTS transfer and its measurable cost.
type HandoffRecord struct {
	Sequence                          uint64  `json:"sequence"`
	Reason                            string  `json:"reason"`
	AnchorWorkAtHandoff               uint64  `json:"anchor_work_at_handoff"`
	BoltsWork                         uint64  `json:"bolts_work"`
	BoltsTimeNS                       int64   `json:"bolts_time_ns"`
	AvailableStateUnits               uint64  `json:"available_state_units"`
	TransferredStateUnits             uint64  `json:"transferred_state_units"`
	QueuedSeedStateUnits              uint64  `json:"queued_seed_state_units"`
	ExpandedSeedStateUnits            uint64  `json:"expanded_seed_state_units"`
	PathContributingSeedStateUnits    uint64  `json:"path_contributing_seed_state_units"`
	ReusedStateUnits                  uint64  `json:"reused_state_units"`
	PreHandoffWasteWork               uint64  `json:"pre_handoff_waste_work"`
	BoltsStandaloneWork               *uint64 `json:"bolts_standalone_work,omitempty"`
	AdditionalWorkVsBoltsStandalone   *int64  `json:"additional_work_vs_bolts_standalone,omitempty"`
	BoltsStandaloneTimeNS             *int64  `json:"bolts_standalone_time_ns,omitempty"`
	AdditionalTimeNSVsBoltsStandalone *int64  `json:"additional_time_ns_vs_bolts_standalone,omitempty"`
}

type HandoffMetrics struct {
	Count                               uint64          `json:"count"`
	Records                             []HandoffRecord `json:"records,omitempty"`
	TotalBoltsWork                      uint64          `json:"total_bolts_work"`
	TotalBoltsTimeNS                    int64           `json:"total_bolts_time_ns"`
	TotalAvailableStateUnits            uint64          `json:"total_available_state_units"`
	TotalTransferredStateUnits          uint64          `json:"total_transferred_state_units"`
	TotalQueuedSeedStateUnits           uint64          `json:"total_queued_seed_state_units"`
	TotalExpandedSeedStateUnits         uint64          `json:"total_expanded_seed_state_units"`
	TotalPathContributingSeedStateUnits uint64          `json:"total_path_contributing_seed_state_units"`
	TotalReusedStateUnits               uint64          `json:"total_reused_state_units"`
	TotalPreHandoffWasteWork            uint64          `json:"total_pre_handoff_waste_work"`
}

type ProgressSample struct {
	Work                      uint64  `json:"work"`
	CandidateFound            bool    `json:"candidate_found"`
	WorksSinceCandidateUpdate uint64  `json:"works_since_candidate_update"`
	FrontierSize              uint64  `json:"frontier_size"`
	RejectRate                float64 `json:"reject_rate"`
	BestHeuristic             float64 `json:"best_heuristic"`
	LowerBound                float64 `json:"lower_bound"`
}

type BottleneckProfile struct {
	AnchorWork                uint64           `json:"anchor_work"`
	BoltsWork                 uint64           `json:"bolts_work"`
	TrussWork                 uint64           `json:"truss_work"`
	AnchorTimeNS              int64            `json:"anchor_time_ns"`
	BoltsTimeNS               int64            `json:"bolts_time_ns"`
	OrchestrationTimeNS       int64            `json:"orchestration_time_ns"`
	EpochCount                uint64           `json:"epoch_count"`
	MaxFrontierSize           uint64           `json:"max_frontier_size"`
	CandidateUpdateCount      uint64           `json:"candidate_update_count"`
	WorksSinceCandidateUpdate uint64           `json:"works_since_candidate_update"`
	DominantWorkComponent     string           `json:"dominant_work_component"`
	DominantTimeComponent     string           `json:"dominant_time_component"`
	ProgressSamples           []ProgressSample `json:"progress_samples,omitempty"`
}
