package planner

import "sort"

const (
	DecisionClosed  = "CLOSED"
	DecisionUnknown = "UNKNOWN"
	DecisionRefuted = "REFUTED"

	ActionExecute = "EXECUTE"
	ActionReuse   = "REUSE"
	ActionUnknown = "UNKNOWN"
	ActionRefuted = "REFUTED"

	ProofPass           = "PASS"
	ProofFailed         = "FAILED"
	ProofCounterexample = "COUNTEREXAMPLE"

	OperationalRefuted = "OPERATIONAL_REFUTED"
	ToolchainVersion   = "go1.27.0"

	IndicatorObserved = "OBSERVED"
	IndicatorUnknown  = "UNKNOWN"
	IndicatorRefuted  = "REFUTED"
)

var Precedence = []string{DecisionRefuted, DecisionUnknown, DecisionClosed}

var RequiredActivities = []string{
	"parse-gooo",
	"resolve-validation-units",
	"bind-semantic-dependencies",
	"compute-impact-closure",
	"bind-cache-identity",
	"inspect-proof-state",
	"apply-reuse-rule",
	"classify-execution-action",
	"materialize-unit-vector",
	"observe-matched-identity",
	"preserve-operational-refutation",
	"render-conformance-evidence",
}

var RequiredAuthorities = []string{
	"ValidationUnit",
	"SemanticDependency",
	"CacheIdentity",
	"ReuseRule",
	"ExecutionPlan",
}

type Meta struct {
	Schema           string
	Program          string
	Namespace        string
	Precedence       []string
	Authorities      []string
	ValidationUnit   string
	Dependency       string
	CacheIdentity    string
	ReuseRules       []string
	ExecutionPlan    string
	Activities       []string
	ProofCells       []Cell
	IndicatorCells   []Cell
	OptionalInput    OptionalInput
	ForbiddenEffects []string
	SourcePath       string
	SourceDigest     string
}

type Cell struct {
	State string `json:"state"`
	ID    string `json:"id"`
}

type OptionalInput struct {
	Name             string `json:"name"`
	Release          string `json:"release"`
	DigestPinned     bool   `json:"digest_pinned"`
	Required         bool   `json:"required"`
	CrossProjectGate bool   `json:"cross_project_gate"`
	Copied           bool   `json:"copied"`
}

type CacheIdentity struct {
	SourceDigest            string `json:"source_digest"`
	SemanticIRDigest        string `json:"semantic_ir_digest"`
	FixtureDigest           string `json:"fixture_digest"`
	ContractDigest          string `json:"contract_digest"`
	GoToolchainDigest       string `json:"go_toolchain_digest"`
	CommandDescriptorDigest string `json:"command_descriptor_digest"`
}

func (c CacheIdentity) Missing() []string {
	missing := make([]string, 0, 6)
	if c.SourceDigest == "" {
		missing = append(missing, "source_digest")
	}
	if c.SemanticIRDigest == "" {
		missing = append(missing, "semantic_ir_digest")
	}
	if c.FixtureDigest == "" {
		missing = append(missing, "fixture_digest")
	}
	if c.ContractDigest == "" {
		missing = append(missing, "contract_digest")
	}
	if c.GoToolchainDigest == "" {
		missing = append(missing, "go_toolchain_digest")
	}
	if c.CommandDescriptorDigest == "" {
		missing = append(missing, "command_descriptor_digest")
	}
	return missing
}

func (c CacheIdentity) Equal(other CacheIdentity) bool {
	return c.SourceDigest == other.SourceDigest &&
		c.SemanticIRDigest == other.SemanticIRDigest &&
		c.FixtureDigest == other.FixtureDigest &&
		c.ContractDigest == other.ContractDigest &&
		c.GoToolchainDigest == other.GoToolchainDigest &&
		c.CommandDescriptorDigest == other.CommandDescriptorDigest
}

type ValidationUnit struct {
	ID              string        `json:"id"`
	SourceRef       string        `json:"source_ref"`
	Command         string        `json:"command"`
	SemanticNodes   []string      `json:"semantic_nodes"`
	CurrentIdentity CacheIdentity `json:"current_identity"`
	Cache           *CacheReceipt `json:"cache,omitempty"`
	Metrics         MetricVector  `json:"metrics"`
}

type CacheReceipt struct {
	Schema       string        `json:"schema"`
	State        string        `json:"state"`
	Immutable    bool          `json:"immutable"`
	Identity     CacheIdentity `json:"identity"`
	ResultDigest string        `json:"result_digest"`
	RunID        string        `json:"run_id"`
}

type SemanticNode struct {
	ID     string `json:"id"`
	Digest string `json:"digest"`
}

type SemanticDependency struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Relation   string `json:"relation"`
	FromDigest string `json:"from_digest,omitempty"`
	ToDigest   string `json:"to_digest,omitempty"`
}

type SemanticGraph struct {
	Nodes []SemanticNode       `json:"nodes"`
	Edges []SemanticDependency `json:"edges"`
}

type MetricVector struct {
	BuildMS    *int64 `json:"build_ms"`
	TestMS     *int64 `json:"test_ms"`
	WallMS     *int64 `json:"wall_ms"`
	PeakRSSKiB *int64 `json:"peak_rss_kib"`
}

type MetricPair struct {
	Before      MetricVector  `json:"before"`
	After       MetricVector  `json:"after"`
	BeforeID    CacheIdentity `json:"before_identity"`
	AfterID     CacheIdentity `json:"after_identity"`
	BeforeKnown bool          `json:"before_known"`
	AfterKnown  bool          `json:"after_known"`
	RunState    string        `json:"run_state"`
}

type OptionalSlicerInput struct {
	Release string `json:"release"`
	Digest  string `json:"digest"`
}

type Fixture struct {
	Schema         string               `json:"schema"`
	CaseID         string               `json:"case_id"`
	Description    string               `json:"description"`
	Kind           string               `json:"kind"`
	Before         SemanticGraph        `json:"before"`
	After          SemanticGraph        `json:"after"`
	Units          []ValidationUnit     `json:"units"`
	Indicators     MetricPair           `json:"indicators"`
	OptionalSlicer *OptionalSlicerInput `json:"optional_slicer,omitempty"`
	Expected       Expected             `json:"expected"`
}

type Expected struct {
	Decision string            `json:"decision"`
	Actions  map[string]string `json:"actions"`
}

type ImpactClosure struct {
	ChangedNodes  []string `json:"changed_nodes"`
	ImpactedNodes []string `json:"impacted_nodes"`
	Edges         []string `json:"edges"`
}

type UnitPlan struct {
	UnitID     string        `json:"unit_id"`
	Action     string        `json:"action"`
	Planned    bool          `json:"planned"`
	Executed   bool          `json:"executed"`
	Reused     bool          `json:"reused"`
	Unknown    bool          `json:"unknown"`
	Refuted    bool          `json:"refuted"`
	Reason     string        `json:"reason"`
	Next       string        `json:"next_operation"`
	Identity   CacheIdentity `json:"identity"`
	Missing    []string      `json:"missing_identity_fields,omitempty"`
	Mismatched []string      `json:"mismatched_identity_fields,omitempty"`
	PriorState string        `json:"prior_state,omitempty"`
	Metrics    MetricVector  `json:"metrics"`
}

type Evidence struct {
	Stage     string   `json:"stage"`
	Step      string   `json:"step"`
	Reason    string   `json:"reason"`
	Next      string   `json:"next_operation"`
	BlockedBy []string `json:"blocked_by"`
}

type IndicatorObservation struct {
	UnitID       string `json:"unit_id"`
	Metric       string `json:"metric"`
	Before       *int64 `json:"before"`
	After        *int64 `json:"after"`
	SignedDelta  *int64 `json:"signed_delta"`
	State        string `json:"state"`
	Reason       string `json:"reason"`
	IdentityPair bool   `json:"same_identity"`
}

type OperationalAudit struct {
	State                     string `json:"state"`
	RepositoryWrites          int    `json:"repository_writes"`
	LocalTestExecutions       int    `json:"local_test_executions"`
	CrossProjectRequiredGates int    `json:"cross_project_required_gates"`
	FailedRunsPreserved       bool   `json:"failed_runs_preserved"`
	FailureMutation           string `json:"failure_mutation"`
}

type Report struct {
	Schema         string                 `json:"schema"`
	CaseID         string                 `json:"case_id"`
	Decision       string                 `json:"decision"`
	Precedence     []string               `json:"precedence"`
	Impact         ImpactClosure          `json:"semantic_impact"`
	Units          []UnitPlan             `json:"unit_vector"`
	Indicators     []IndicatorObservation `json:"indicator_vector"`
	Unknowns       []Evidence             `json:"unknown_evidence,omitempty"`
	Refutations    []Evidence             `json:"refutation_evidence,omitempty"`
	OptionalSlicer OptionalSlicerStatus   `json:"optional_slicer"`
	Operational    OperationalAudit       `json:"operational_audit"`
	SourceDigest   string                 `json:"source_digest"`
	ContractDigest string                 `json:"contract_digest"`
	MetaDigest     string                 `json:"meta_digest"`
}

type OptionalSlicerStatus struct {
	Consumed         bool   `json:"consumed"`
	Required         bool   `json:"required"`
	CrossProjectGate bool   `json:"cross_project_gate"`
	Release          string `json:"release,omitempty"`
	Digest           string `json:"digest,omitempty"`
	Reason           string `json:"reason"`
}

type CaseSpec struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Expected string `json:"expected"`
	Kind     string `json:"kind"`
}

type Denominator struct {
	Schema               string         `json:"schema"`
	ID                   string         `json:"id"`
	Version              string         `json:"version"`
	Fixed                bool           `json:"fixed"`
	Cells                int            `json:"cells"`
	Precedence           []string       `json:"precedence"`
	ProofDenominator     map[string]int `json:"proof_denominator"`
	IndicatorDenominator map[string]int `json:"indicator_denominator"`
	Cases                []CaseSpec     `json:"cases"`
}

type CaseResult struct {
	ID       string `json:"id"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Match    bool   `json:"match"`
	Report   string `json:"report"`
}

type SuiteReport struct {
	Schema      string           `json:"schema"`
	Decision    string           `json:"decision"`
	Cases       []CaseResult     `json:"cases"`
	Operational OperationalAudit `json:"operational_audit"`
}

func sortedStrings(values []string) []string {
	copyValues := append([]string(nil), values...)
	sort.Strings(copyValues)
	return copyValues
}
