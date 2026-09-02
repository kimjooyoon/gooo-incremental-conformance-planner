package planner

const (
	V3DecisionClosed  = "CLOSED"
	V3DecisionUnknown = "UNKNOWN"
	V3DecisionRefuted = "REFUTED"

	V3ActionReusedClosed = "REUSED_CLOSED"
	V3ActionRequiredRun  = "REQUIRED_RUN"
	V3ActionUnknown      = "UNKNOWN"
	V3ActionRefuted      = "REFUTED"

	V3ProofFoundation = "FOUNDATION"
	V3ProofCoherence  = "COHERENCE"
	V3ProofRegression = "REGRESSION"

	V3IndicatorDriver    = "DRIVER"
	V3IndicatorOutcome   = "OUTCOME"
	V3IndicatorGuardrail = "GUARDRAIL"

	V3UnknownMissingIdentity   = "MISSING_IDENTITY"
	V3UnknownMissingProvenance = "MISSING_PROVENANCE"
	V3UnknownMissingReceipt    = "MISSING_IMMUTABLE_RECEIPT"
	V3UnknownMissingMetrics    = "MISSING_ACTIONS_METRIC"
	V3UnknownMissingPriorRun   = "MISSING_PRIOR_RUN_IDENTITY"

	V3RefutedForgedReceipt         = "FORGED_OR_STALE_RECEIPT"
	V3RefutedAffectedSkip          = "AFFECTED_ACTIVITY_SKIPPED"
	V3RefutedEvaluatorSelfApproval = "EVALUATOR_SELF_APPROVAL"
	V3RefutedGraph                 = "PROOF_GRAPH_CONTRADICTION"
	V3RefutedLegacyReuse           = "LEGACY_EXACT_REUSE_WEAKENED"

	V3FixedPointExplicit = "EXPLICIT_FIXED_POINT"
	V3FixedPointRefuted  = "REFUTED"
	V3FixedPointNone     = "NOT_APPLICABLE"

	V3IndicatorObserved = "OBSERVED"
	V3IndicatorUnknown  = "UNKNOWN"
	V3IndicatorRefuted  = "REFUTED"
)

var V3IdentityFields = []string{
	"test_identity",
	"conformance_identity",
	"scenario_identity",
	"input_digest",
	"toolchain_digest",
	"semantic_ir_digest",
	"source_digest",
	"fixture_digest",
	"contract_digest",
	"evaluator_digest",
}

var V3PriorRunFields = []string{"run_id", "commit_sha", "receipt_digest"}

var V3ObservationFields = []string{
	"build_ms",
	"test_ms",
	"wall_ms",
	"peak_rss_kib",
	"cache_hit",
	"cache_miss",
}

var V3UnknownFields = []string{"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by"}

var V3Policies = []string{
	"EXACT_TEST_CONFORMANCE_IDENTITY_ONLY",
	"REUSE_ONLY_INDEPENDENT_IMMUTABLE_PASS",
	"IMPACT_OR_IDENTITY_CHANGE_REQUIRES_RUN",
	"MISSING_METRICS_UNKNOWN_REUSE_AND_MEASUREMENT_CLAIM",
	"EXACT_MATCHED_PAIR_ONLY_PER_INDICATOR_DELTA",
	"NO_TOTAL_SCORE_OR_WEIGHTED_SCORE_OR_ESTIMATE",
	"LEGACY_EXACT_REUSE_MUST_REMAIN_CLOSED",
	"FIXED_POINT_ONLY_IF_EXPLICIT",
	"UNKNOWN_TOP_DECISION_FAIL_CLOSED",
	"MALFORMED_OR_IMPLICIT_FIXED_POINT_REFUTED",
}

type V3Source struct {
	Schema            string
	Program           string
	Namespace         string
	Precedence        []string
	Authorities       []string
	IdentityFields    []string
	PriorRunFields    []string
	ObservationFields []string
	UnknownFields     []string
	UnknownClasses    []string
	Policies          []string
	ToolchainDigest   string
	ProofChoices      []string
	IndicatorClasses  []string
	FixedPointRules   []string
	FixedPointCases   []string
	Activities        []V2ActivityDescriptor
}

type V3SemanticIR struct {
	Schema            string                 `json:"schema"`
	Program           string                 `json:"program"`
	SourcePath        string                 `json:"source_path"`
	SourceDigest      string                 `json:"source_digest"`
	ToolchainDigest   string                 `json:"toolchain_digest"`
	IdentityFields    []string               `json:"identity_fields"`
	PriorRunFields    []string               `json:"prior_run_fields"`
	ObservationFields []string               `json:"observation_fields"`
	UnknownFields     []string               `json:"unknown_fields"`
	Policies          []string               `json:"policies"`
	Activities        []V2ActivityDescriptor `json:"activities"`
	EvaluatorDigest   string                 `json:"evaluator_digest"`
	Digest            string                 `json:"digest"`
}

type V3VerificationIdentity struct {
	TestIdentity        string `json:"test_identity"`
	ConformanceIdentity string `json:"conformance_identity"`
	ScenarioIdentity    string `json:"scenario_identity"`
	InputDigest         string `json:"input_digest"`
	ToolchainDigest     string `json:"toolchain_digest"`
	SemanticIRDigest    string `json:"semantic_ir_digest"`
	SourceDigest        string `json:"source_digest"`
	FixtureDigest       string `json:"fixture_digest"`
	ContractDigest      string `json:"contract_digest"`
	EvaluatorDigest     string `json:"evaluator_digest"`
}

func (identity V3VerificationIdentity) Missing() []string {
	values := []string{
		identity.TestIdentity, identity.ConformanceIdentity, identity.ScenarioIdentity,
		identity.InputDigest, identity.ToolchainDigest, identity.SemanticIRDigest,
		identity.SourceDigest, identity.FixtureDigest, identity.ContractDigest, identity.EvaluatorDigest,
	}
	missing := make([]string, 0, len(V3IdentityFields))
	for index, value := range values {
		if value == "" {
			missing = append(missing, V3IdentityFields[index])
		}
	}
	return missing
}

func (identity V3VerificationIdentity) Equal(other V3VerificationIdentity) bool {
	return identity.TestIdentity == other.TestIdentity &&
		identity.ConformanceIdentity == other.ConformanceIdentity &&
		identity.ScenarioIdentity == other.ScenarioIdentity &&
		identity.InputDigest == other.InputDigest &&
		identity.ToolchainDigest == other.ToolchainDigest &&
		identity.SemanticIRDigest == other.SemanticIRDigest &&
		identity.SourceDigest == other.SourceDigest &&
		identity.FixtureDigest == other.FixtureDigest &&
		identity.ContractDigest == other.ContractDigest &&
		identity.EvaluatorDigest == other.EvaluatorDigest
}

type V3PriorRunIdentity struct {
	RunID         string `json:"run_id"`
	CommitSHA     string `json:"commit_sha"`
	ReceiptDigest string `json:"receipt_digest"`
}

func (identity V3PriorRunIdentity) Missing() []string {
	values := []string{identity.RunID, identity.CommitSHA, identity.ReceiptDigest}
	missing := make([]string, 0, len(V3PriorRunFields))
	for index, value := range values {
		if value == "" {
			missing = append(missing, V3PriorRunFields[index])
		}
	}
	return missing
}

type V3Observation struct {
	ActivityID string `json:"activity_id,omitempty"`
	Status     string `json:"status"`
	BuildMS    *int64 `json:"build_ms"`
	TestMS     *int64 `json:"test_ms"`
	WallMS     *int64 `json:"wall_ms"`
	PeakRSSKiB *int64 `json:"peak_rss_kib"`
	CacheHit   *bool  `json:"cache_hit"`
	CacheMiss  *bool  `json:"cache_miss"`
	DurationMS *int64 `json:"duration_ms,omitempty"`
}

type V3MetricVector struct {
	BuildMS    *int64 `json:"build_ms"`
	TestMS     *int64 `json:"test_ms"`
	WallMS     *int64 `json:"wall_ms"`
	PeakRSSKiB *int64 `json:"peak_rss_kib"`
}

type V3MetricPair struct {
	ScenarioIdentity string                 `json:"scenario_identity"`
	Before           V3MetricVector         `json:"before"`
	After            V3MetricVector         `json:"after"`
	BeforeIdentity   V3VerificationIdentity `json:"before_identity"`
	AfterIdentity    V3VerificationIdentity `json:"after_identity"`
	BeforePriorRun   V3PriorRunIdentity     `json:"before_prior_run_identity"`
	AfterPriorRun    V3PriorRunIdentity     `json:"after_prior_run_identity"`
	BeforeKnown      bool                   `json:"before_known"`
	AfterKnown       bool                   `json:"after_known"`
	RunState         string                 `json:"run_state"`
}

type V3GraphNode struct {
	ID     string `json:"id"`
	Digest string `json:"digest"`
}

type V3GraphEdge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Relation   string `json:"relation"`
	FromDigest string `json:"from_digest,omitempty"`
	ToDigest   string `json:"to_digest,omitempty"`
}

type V3ProofGraph struct {
	Nodes []V3GraphNode `json:"nodes"`
	Edges []V3GraphEdge `json:"edges"`
}

type V3ChangeSet struct {
	BeforeDigest string   `json:"before_digest"`
	AfterDigest  string   `json:"after_digest"`
	ChangedNodes []string `json:"changed_nodes"`
}

type V3ActivityInput struct {
	ActivityID      string                  `json:"activity_id"`
	SemanticNodes   []string                `json:"semantic_nodes"`
	CurrentIdentity *V3VerificationIdentity `json:"current_identity,omitempty"`
	PriorEvidence   *V3PriorEvidence        `json:"prior_evidence,omitempty"`
	Observation     *V3Observation          `json:"observation,omitempty"`
}

type V3PriorEvidence struct {
	State         string                 `json:"state"`
	Immutable     bool                   `json:"immutable"`
	Identity      V3VerificationIdentity `json:"identity"`
	PriorRun      V3PriorRunIdentity     `json:"prior_run_identity"`
	ResultDigest  string                 `json:"result_digest"`
	ReceiptDigest string                 `json:"receipt_digest"`
	ContentDigest string                 `json:"content_digest"`
	ProofSource   string                 `json:"proof_source"`
}

type V3FixedPointEvidence struct {
	Declared bool   `json:"declared"`
	Kind     string `json:"kind"`
	Witness  string `json:"witness"`
}

type V3Fixture struct {
	Schema                string                 `json:"schema"`
	CaseID                string                 `json:"case_id"`
	Description           string                 `json:"description"`
	Kind                  string                 `json:"kind"`
	FixtureAnchor         string                 `json:"fixture_anchor"`
	FixedPoint            *V3FixedPointEvidence  `json:"fixed_point,omitempty"`
	SemanticChange        V3ChangeSet            `json:"semantic_change"`
	Before                V3ProofGraph           `json:"before"`
	After                 V3ProofGraph           `json:"after"`
	IdentityDefaults      V3VerificationIdentity `json:"identity_defaults"`
	PriorEvidenceDefaults *V3PriorEvidence       `json:"prior_evidence_defaults,omitempty"`
	ObservationDefaults   V3Observation          `json:"observation_defaults"`
	Activities            []V3ActivityInput      `json:"activities"`
	Indicators            V3MetricPair           `json:"indicators"`
	Expected              V3Expected             `json:"expected"`
}

type V3Expected struct {
	Decision   string            `json:"decision"`
	Activities map[string]string `json:"activities"`
}

type V3Contract struct {
	Schema            string                 `json:"schema"`
	ID                string                 `json:"id"`
	Version           string                 `json:"version"`
	AppendOnlyFrom    string                 `json:"append_only_from"`
	TargetActivities  int                    `json:"target_activities"`
	Precedence        []string               `json:"precedence"`
	IdentityFields    []string               `json:"identity_fields"`
	PriorRunFields    []string               `json:"prior_run_fields"`
	ObservationFields []string               `json:"observation_fields"`
	UnknownFields     []string               `json:"unknown_fields"`
	ForbiddenOutputs  []string               `json:"forbidden_outputs"`
	ProofTotals       map[string]int         `json:"proof_totals"`
	IndicatorTotals   map[string]int         `json:"indicator_totals"`
	Activities        []V2ActivityDescriptor `json:"activities"`
	Scenarios         []V3CaseSpec           `json:"scenarios"`
}

type V3CaseSpec struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Expected string `json:"expected"`
}

type V3Projection struct {
	CaseID       string
	Change       V3ChangeSet
	ImpactNodes  []string
	ImpactEdges  []string
	GraphReasons []string
	Activities   []V3ProjectedActivity
}

type V3ProjectedActivity struct {
	Descriptor  V2ActivityDescriptor
	Input       V3ActivityInput
	Identity    V3VerificationIdentity
	Prior       *V3PriorEvidence
	Observation V3Observation
}

type V3UnknownRecord struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type V3ActivityPlan struct {
	ActivityID          string              `json:"activity_id"`
	Activity            string              `json:"activity"`
	ProofChoice         string              `json:"proof_choice"`
	IndicatorClass      string              `json:"indicator_class"`
	Action              string              `json:"action"`
	ReusableEvidence    bool                `json:"reusable_evidence"`
	MustExecute         bool                `json:"must_execute"`
	AlreadyVerified     bool                `json:"already_verified"`
	RequiredRun         bool                `json:"required_run"`
	Executed            bool                `json:"executed"`
	SkippedWithProof    bool                `json:"skipped_with_proof"`
	Unknown             bool                `json:"unknown"`
	Refuted             bool                `json:"refuted"`
	ReuseClaimState     string              `json:"reuse_claim_state"`
	MeasurementState    string              `json:"measurement_state"`
	TestIdentity        string              `json:"test_identity"`
	ConformanceIdentity string              `json:"conformance_identity"`
	InputDigest         string              `json:"input_digest"`
	ToolchainDigest     string              `json:"toolchain_digest"`
	SemanticIRDigest    string              `json:"semantic_ir_digest"`
	PriorRunIdentity    *V3PriorRunIdentity `json:"prior_run_identity,omitempty"`
	BuildMS             *int64              `json:"build_ms"`
	TestMS              *int64              `json:"test_ms"`
	WallMS              *int64              `json:"wall_ms"`
	PeakRSSKiB          *int64              `json:"peak_rss_kib"`
	CacheHit            *bool               `json:"cache_hit"`
	CacheMiss           *bool               `json:"cache_miss"`
	MismatchedFields    []string            `json:"mismatched_fields,omitempty"`
	UnknownRecord       *V3UnknownRecord    `json:"unknown_record,omitempty"`
	Reason              string              `json:"reason"`
	NextOperation       string              `json:"next_operation"`
}

type V3IndicatorObservation struct {
	Metric        string           `json:"metric"`
	Before        *int64           `json:"before"`
	After         *int64           `json:"after"`
	Delta         *int64           `json:"delta"`
	SignedDelta   *int64           `json:"signed_delta"`
	Improvement   *int64           `json:"improvement"`
	State         string           `json:"state"`
	SameIdentity  bool             `json:"same_identity"`
	UnknownRecord *V3UnknownRecord `json:"unknown_record,omitempty"`
	Reason        string           `json:"reason"`
}

type V3DossierSummary struct {
	TotalActivities  int `json:"total_activities"`
	ReusableEvidence int `json:"reusable_evidence"`
	RequiredRuns     int `json:"required_runs"`
	Unknown          int `json:"unknown"`
	Refuted          int `json:"refuted"`
	Executed         int `json:"executed"`
	SkippedWithProof int `json:"skipped_with_proof"`
}

type V3ActionsReceipt struct {
	Schema                    string             `json:"schema"`
	RunIdentity               V3PriorRunIdentity `json:"run_identity"`
	TestIdentity              string             `json:"test_identity"`
	ConformanceIdentity       string             `json:"conformance_identity"`
	InputDigest               string             `json:"input_digest"`
	ToolchainDigest           string             `json:"toolchain_digest"`
	SemanticIRDigest          string             `json:"semantic_ir_digest"`
	BuildMS                   *int64             `json:"build_ms"`
	TestMS                    *int64             `json:"test_ms"`
	WallMS                    *int64             `json:"wall_ms"`
	PeakRSSKiB                *int64             `json:"peak_rss_kib"`
	CacheHit                  *bool              `json:"cache_hit"`
	CacheMiss                 *bool              `json:"cache_miss"`
	CacheHits                 *int64             `json:"cache_hits"`
	CacheMisses               *int64             `json:"cache_misses"`
	Activities                []V3Observation    `json:"activities"`
	OperationalState          string             `json:"operational_state"`
	RepositoryWrites          int                `json:"repository_writes"`
	LocalTestExecutions       int                `json:"local_test_executions"`
	CrossProjectRequiredGates int                `json:"cross_project_required_gates"`
}

type V3Report struct {
	Schema          string                   `json:"schema"`
	CaseID          string                   `json:"case_id"`
	Decision        string                   `json:"decision"`
	Precedence      []string                 `json:"precedence"`
	SemanticIR      V3SemanticIR             `json:"semantic_ir"`
	SemanticChange  V3ChangeSet              `json:"semantic_change"`
	ImpactNodes     []string                 `json:"impacted_nodes"`
	ImpactEdges     []string                 `json:"impact_edges"`
	Activities      []V3ActivityPlan         `json:"activities"`
	Summary         V3DossierSummary         `json:"dossier_summary"`
	Indicators      []V3IndicatorObservation `json:"indicators"`
	Unknowns        []V3UnknownRecord        `json:"unknowns,omitempty"`
	RefutedReasons  []string                 `json:"refuted_reasons,omitempty"`
	ContractDigest  string                   `json:"contract_digest"`
	FixtureDigest   string                   `json:"fixture_digest"`
	EvaluatorDigest string                   `json:"evaluator_digest"`
	ActionsReceipt  *V3ActionsReceipt        `json:"actions_receipt,omitempty"`
	FixedPointState string                   `json:"fixed_point_state"`
	Operational     V3Operational            `json:"operational"`
}

type V3Operational struct {
	RepositoryWrites          int    `json:"repository_writes"`
	LocalTestExecutions       int    `json:"local_test_executions"`
	CrossProjectRequiredGates int    `json:"cross_project_required_gates"`
	FailedRunsPreserved       bool   `json:"failed_runs_preserved"`
	OutputLocation            string `json:"output_location"`
	VerificationAuthority     string `json:"verification_authority"`
}

type V3CaseResult struct {
	ID       string `json:"id"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Match    bool   `json:"match"`
	Report   string `json:"report"`
}

type V3SuiteReport struct {
	Schema                 string            `json:"schema"`
	Decision               string            `json:"decision"`
	Contract               string            `json:"contract"`
	ContractDigest         string            `json:"contract_digest"`
	TotalActivities        int               `json:"total_activities"`
	ProofTotals            map[string]int    `json:"proof_totals"`
	IndicatorTotals        map[string]int    `json:"indicator_totals"`
	Cases                  []V3CaseResult    `json:"cases"`
	LegacyExactReuseClosed bool              `json:"legacy_exact_reuse_closed"`
	ActionsReceipt         *V3ActionsReceipt `json:"actions_receipt,omitempty"`
	ActionsMetricState     string            `json:"actions_metric_state"`
	MissingActionsMetrics  []string          `json:"missing_actions_metrics,omitempty"`
	Operational            V3Operational     `json:"operational"`
}
