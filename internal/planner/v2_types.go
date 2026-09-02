package planner

const (
	V2DecisionClosed  = "CLOSED"
	V2DecisionUnknown = "UNKNOWN"
	V2DecisionRefuted = "REFUTED"

	V2ActionReusedClosed = "REUSED_CLOSED"
	V2ActionRequiredRun  = "REQUIRED_RUN"
	V2ActionUnknown      = "UNKNOWN"
	V2ActionRefuted      = "REFUTED"

	V2ProofFoundation = "FOUNDATION"
	V2ProofCoherence  = "COHERENCE"
	V2ProofRegression = "REGRESSION"

	V2IndicatorDriver    = "DRIVER"
	V2IndicatorOutcome   = "OUTCOME"
	V2IndicatorGuardrail = "GUARDRAIL"

	V2UnknownMissingIdentity   = "MISSING_IDENTITY"
	V2UnknownMissingProvenance = "MISSING_PROVENANCE"
	V2UnknownMissingReceipt   = "MISSING_IMMUTABLE_RECEIPT"
	V2UnknownMissingMetrics   = "MISSING_ACTIONS_METRIC"

	V2RefutedForgedReceipt       = "FORGED_OR_STALE_RECEIPT"
	V2RefutedAffectedSkip        = "AFFECTED_ACTIVITY_SKIPPED"
	V2RefutedEvaluatorSelfApproval = "EVALUATOR_SELF_APPROVAL"
	V2RefutedGraph               = "PROOF_GRAPH_CONTRADICTION"
)

var V2IdentityDigestFields = []string{
	"source_digest",
	"semantic_ir_digest",
	"fixture_digest",
	"contract_digest",
	"evaluator_digest",
	"go_toolchain_digest",
}

var V2ActivityIdentityFields = []string{
	"scenario_digest",
	"build_activity_id",
	"test_activity_id",
}

type V2Source struct {
	Schema          string
	Program         string
	Namespace       string
	Precedence      []string
	Authorities     []string
	IdentityFields  []string
	UnknownClasses  []string
	ToolchainDigest string
	ProofChoices    []string
	IndicatorClasses []string
	Activities      []V2ActivityDescriptor
}

type V2ActivityDescriptor struct {
	Ordinal        int      `json:"ordinal"`
	ID             string   `json:"id"`
	Activity       string   `json:"activity"`
	Stage          string   `json:"stage"`
	Step           string   `json:"step"`
	ProofChoice    string   `json:"proof_choice"`
	IndicatorClass string   `json:"indicator_class"`
	DependsOn      []string `json:"depends_on"`
}

type V2SemanticIR struct {
	Schema          string                 `json:"schema"`
	Program         string                 `json:"program"`
	SourcePath      string                 `json:"source_path"`
	SourceDigest    string                 `json:"source_digest"`
	ToolchainDigest string                 `json:"toolchain_digest"`
	IdentityFields  []string               `json:"identity_fields"`
	Activities      []V2ActivityDescriptor `json:"activities"`
	EvaluatorDigest string                 `json:"evaluator_digest"`
	Digest          string                 `json:"digest"`
}

type V2CacheIdentity struct {
	SourceDigest      string `json:"source_digest"`
	SemanticIRDigest  string `json:"semantic_ir_digest"`
	FixtureDigest     string `json:"fixture_digest"`
	ContractDigest    string `json:"contract_digest"`
	EvaluatorDigest   string `json:"evaluator_digest"`
	GoToolchainDigest string `json:"go_toolchain_digest"`
}

func (identity V2CacheIdentity) Missing() []string {
	missing := make([]string, 0, len(V2IdentityDigestFields))
	values := []string{identity.SourceDigest, identity.SemanticIRDigest, identity.FixtureDigest, identity.ContractDigest, identity.EvaluatorDigest, identity.GoToolchainDigest}
	for index, value := range values {
		if value == "" {
			missing = append(missing, V2IdentityDigestFields[index])
		}
	}
	return missing
}

func (identity V2CacheIdentity) Equal(other V2CacheIdentity) bool {
	return identity.SourceDigest == other.SourceDigest &&
		identity.SemanticIRDigest == other.SemanticIRDigest &&
		identity.FixtureDigest == other.FixtureDigest &&
		identity.ContractDigest == other.ContractDigest &&
		identity.EvaluatorDigest == other.EvaluatorDigest &&
		identity.GoToolchainDigest == other.GoToolchainDigest
}

type V2ActivityIdentity struct {
	ScenarioDigest   string `json:"scenario_digest"`
	BuildActivityID  string `json:"build_activity_id"`
	TestActivityID   string `json:"test_activity_id"`
}

func (identity V2ActivityIdentity) Missing() []string {
	missing := make([]string, 0, len(V2ActivityIdentityFields))
	values := []string{identity.ScenarioDigest, identity.BuildActivityID, identity.TestActivityID}
	for index, value := range values {
		if value == "" {
			missing = append(missing, V2ActivityIdentityFields[index])
		}
	}
	return missing
}

func (identity V2ActivityIdentity) Equal(other V2ActivityIdentity) bool {
	return identity.ScenarioDigest == other.ScenarioDigest &&
		identity.BuildActivityID == other.BuildActivityID &&
		identity.TestActivityID == other.TestActivityID
}

type V2IdentityBundle struct {
	Digests  V2CacheIdentity   `json:"digests"`
	Activity V2ActivityIdentity `json:"activity"`
}

func (identity V2IdentityBundle) Missing() []string {
	missing := identity.Digests.Missing()
	for _, field := range identity.Activity.Missing() {
		missing = append(missing, field)
	}
	return missing
}

func (identity V2IdentityBundle) Equal(other V2IdentityBundle) bool {
	return identity.Digests.Equal(other.Digests) && identity.Activity.Equal(other.Activity)
}

type V2PriorReceipt struct {
	State         string           `json:"state"`
	Immutable     bool             `json:"immutable"`
	Identity      V2IdentityBundle `json:"identity"`
	ResultDigest  string           `json:"result_digest"`
	ReceiptDigest string           `json:"receipt_digest"`
	ContentDigest string           `json:"content_digest"`
	SourceRunID   string           `json:"source_run_id"`
	SourceCommit  string           `json:"source_commit"`
	ProofSource   string           `json:"proof_source"`
}

type V2ActivityObservation struct {
	ActivityID   string `json:"activity_id,omitempty"`
	Status       string `json:"status"`
	DurationMS   *int64 `json:"duration_ms"`
	BuildMS      *int64 `json:"build_ms"`
	TestMS       *int64 `json:"test_ms"`
	WallMS       *int64 `json:"wall_ms"`
	PeakRSSKiB   *int64 `json:"peak_rss_kib"`
	CacheHit     *bool  `json:"cache_hit"`
	CacheMiss    *bool  `json:"cache_miss"`
}

type V2MetricVector struct {
	BuildMS    *int64 `json:"build_ms"`
	TestMS     *int64 `json:"test_ms"`
	WallMS     *int64 `json:"wall_ms"`
	PeakRSSKiB *int64 `json:"peak_rss_kib"`
}

type V2MetricPair struct {
	ScenarioDigest string           `json:"scenario_digest"`
	Before         V2MetricVector   `json:"before"`
	After          V2MetricVector   `json:"after"`
	BeforeIdentity V2IdentityBundle `json:"before_identity"`
	AfterIdentity  V2IdentityBundle `json:"after_identity"`
	BeforeKnown    bool             `json:"before_known"`
	AfterKnown     bool             `json:"after_known"`
	RunState       string           `json:"run_state"`
}

type V2GraphNode struct {
	ID     string `json:"id"`
	Digest string `json:"digest"`
}

type V2GraphEdge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Relation   string `json:"relation"`
	FromDigest string `json:"from_digest,omitempty"`
	ToDigest   string `json:"to_digest,omitempty"`
}

type V2ProofGraph struct {
	Nodes []V2GraphNode `json:"nodes"`
	Edges []V2GraphEdge `json:"edges"`
}

type V2ChangeSet struct {
	BeforeDigest string   `json:"before_digest"`
	AfterDigest  string   `json:"after_digest"`
	ChangedNodes []string `json:"changed_nodes"`
}

type V2ActivityInput struct {
	ActivityID      string                `json:"activity_id"`
	SemanticNodes   []string              `json:"semantic_nodes"`
	CurrentIdentity *V2IdentityBundle     `json:"current_identity"`
	PriorReceipt    *V2PriorReceipt       `json:"prior_receipt"`
	Observation     *V2ActivityObservation `json:"observation"`
}

type V2Fixture struct {
	Schema              string                   `json:"schema"`
	CaseID              string                   `json:"case_id"`
	Description         string                   `json:"description"`
	Kind                string                   `json:"kind"`
	FixtureAnchor       string                   `json:"fixture_anchor"`
	SemanticChange      V2ChangeSet             `json:"semantic_change"`
	Before              V2ProofGraph            `json:"before"`
	After               V2ProofGraph             `json:"after"`
	IdentityDefaults    V2IdentityBundle        `json:"identity_defaults"`
	ReceiptDefaults     *V2PriorReceipt         `json:"receipt_defaults"`
	ObservationDefaults V2ActivityObservation  `json:"observation_defaults"`
	Activities          []V2ActivityInput       `json:"activities"`
	Indicators          V2MetricPair            `json:"indicators"`
	Expected            V2Expected              `json:"expected"`
}

type V2Expected struct {
	Decision   string            `json:"decision"`
	Activities map[string]string `json:"activities"`
}

type V2Contract struct {
	Schema          string                 `json:"schema"`
	ID              string                 `json:"id"`
	Version         string                 `json:"version"`
	AppendOnlyFrom  string                 `json:"append_only_from"`
	TargetActivities int                   `json:"target_activities"`
	Precedence      []string               `json:"precedence"`
	ProofTotals     map[string]int         `json:"proof_totals"`
	IndicatorTotals map[string]int         `json:"indicator_totals"`
	Activities      []V2ActivityDescriptor `json:"activities"`
	Scenarios       []V2CaseSpec           `json:"scenarios"`
}

type V2CaseSpec struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Expected string `json:"expected"`
}

type V2Projection struct {
	CaseID       string
	Change       V2ChangeSet
	ImpactNodes  []string
	ImpactEdges  []string
	GraphReasons []string
	Activities   []V2ProjectedActivity
}

type V2ProjectedActivity struct {
	Descriptor  V2ActivityDescriptor
	Input       V2ActivityInput
	Identity    V2IdentityBundle
	Receipt     *V2PriorReceipt
	Observation V2ActivityObservation
}

type V2ActivityPlan struct {
	ActivityID       string                `json:"activity_id"`
	Activity         string                `json:"activity"`
	ProofChoice      string                `json:"proof_choice"`
	IndicatorClass   string                `json:"indicator_class"`
	Action           string                `json:"action"`
	AlreadyVerified  bool                  `json:"already_verified"`
	RequiredRun      bool                  `json:"required_run"`
	Executed         bool                  `json:"executed"`
	ReusedClosed     bool                  `json:"reused_closed"`
	SkippedWithProof bool                  `json:"skipped_with_proof"`
	Unknown          bool                  `json:"unknown"`
	Refuted          bool                  `json:"refuted"`
	CacheHit         *bool                 `json:"cache_hit"`
	CacheMiss        *bool                 `json:"cache_miss"`
	DurationMS       *int64                `json:"duration_ms"`
	PriorState       string                `json:"prior_state,omitempty"`
	MissingFields    []string              `json:"missing_fields,omitempty"`
	MismatchedFields []string              `json:"mismatched_fields,omitempty"`
	Reason           string                `json:"reason"`
	NextOperation    string                `json:"next_operation"`
}

type V2IndicatorObservation struct {
	Metric        string `json:"metric"`
	Before        *int64 `json:"before"`
	After         *int64 `json:"after"`
	SignedDelta   *int64 `json:"signed_delta"`
	Improvement   *int64 `json:"improvement"`
	State         string `json:"state"`
	SameIdentity  bool   `json:"same_identity"`
	Reason        string `json:"reason"`
}

type V2DossierSummary struct {
	TotalActivities int `json:"total_activities"`
	RequiredRuns    int `json:"required_runs"`
	ReusedClosed    int `json:"reused_closed"`
	Unknown         int `json:"unknown"`
	Refuted         int `json:"refuted"`
	Executed        int `json:"executed"`
	SkippedWithProof int `json:"skipped_with_proof"`
}

type V2ActionsReceipt struct {
	Schema            string                    `json:"schema"`
	RunID             string                    `json:"run_id"`
	BuildMS           *int64                    `json:"build_ms"`
	TestMS            *int64                    `json:"test_ms"`
	WallMS            *int64                    `json:"wall_ms"`
	PeakRSSKiB        *int64                    `json:"peak_rss_kib"`
	CacheHits         *int64                    `json:"cache_hits"`
	CacheMisses       *int64                    `json:"cache_misses"`
	Activities        []V2ActivityObservation  `json:"activities"`
	OperationalState  string                    `json:"operational_state"`
	RepositoryWrites  int                       `json:"repository_writes"`
	LocalTestExecutions int                     `json:"local_test_executions"`
	CrossProjectRequiredGates int               `json:"cross_project_required_gates"`
}

type V2Report struct {
	Schema          string                    `json:"schema"`
	CaseID          string                    `json:"case_id"`
	Decision        string                    `json:"decision"`
	Precedence      []string                  `json:"precedence"`
	SemanticIR      V2SemanticIR              `json:"semantic_ir"`
	SemanticChange  V2ChangeSet               `json:"semantic_change"`
	ImpactNodes     []string                  `json:"impacted_nodes"`
	ImpactEdges     []string                  `json:"impact_edges"`
	Activities      []V2ActivityPlan          `json:"activities"`
	Summary         V2DossierSummary          `json:"dossier_summary"`
	Indicators      []V2IndicatorObservation  `json:"indicators"`
	UnknownReasons  []string                  `json:"unknown_reasons,omitempty"`
	RefutedReasons  []string                  `json:"refuted_reasons,omitempty"`
	ContractDigest  string                    `json:"contract_digest"`
	FixtureDigest   string                    `json:"fixture_digest"`
	EvaluatorDigest string                    `json:"evaluator_digest"`
	ActionsReceipt  *V2ActionsReceipt         `json:"actions_receipt,omitempty"`
	Operational     V2Operational             `json:"operational"`
}

type V2Operational struct {
	RepositoryWrites          int    `json:"repository_writes"`
	LocalTestExecutions       int    `json:"local_test_executions"`
	CrossProjectRequiredGates int    `json:"cross_project_required_gates"`
	FailedRunsPreserved       bool   `json:"failed_runs_preserved"`
	OutputLocation            string `json:"output_location"`
	VerificationAuthority     string `json:"verification_authority"`
}

type V2CaseResult struct {
	ID       string `json:"id"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Match    bool   `json:"match"`
	Report   string `json:"report"`
}

type V2SuiteReport struct {
	Schema         string             `json:"schema"`
	Decision       string             `json:"decision"`
	Contract       string             `json:"contract"`
	ContractDigest string             `json:"contract_digest"`
	TotalActivities int               `json:"total_activities"`
	Cases          []V2CaseResult     `json:"cases"`
	ActionsReceipt *V2ActionsReceipt  `json:"actions_receipt,omitempty"`
	ActionsMetricState string         `json:"actions_metric_state"`
	MissingActionsMetrics []string    `json:"missing_actions_metrics,omitempty"`
	Operational    V2Operational      `json:"operational"`
}
