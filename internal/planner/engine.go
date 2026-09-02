package planner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type RunOptions struct {
	MetaPath     string
	ContractPath string
	FixturePath  string
	OutputDir    string
}

type SuiteOptions struct {
	MetaPath     string
	ContractPath string
	CasesRoot    string
	OutputDir    string
}

func Run(options RunOptions) (Report, error) {
	if !filepath.IsAbs(options.OutputDir) {
		return Report{}, errors.New("output directory must be an absolute caller-owned path")
	}
	meta, err := ParseMeta(options.MetaPath)
	if err != nil {
		return Report{}, fmt.Errorf("load meta: %w", err)
	}
	contractData, err := os.ReadFile(options.ContractPath)
	if err != nil {
		return Report{}, fmt.Errorf("load contract: %w", err)
	}
	var denominator Denominator
	if err := readJSON(options.ContractPath, &denominator); err != nil {
		return Report{}, err
	}
	if err := validateDenominator(denominator); err != nil {
		return Report{}, err
	}
	fixture, err := LoadFixture(options.FixturePath)
	if err != nil {
		return Report{}, err
	}
	report, err := Plan(meta, fixture, DigestBytes(contractData))
	if err != nil {
		return Report{}, err
	}
	if err := writeReport(options.OutputDir, report); err != nil {
		return Report{}, err
	}
	return report, nil
}

func Plan(meta Meta, fixture Fixture, contractDigest string) (Report, error) {
	if err := validateFixture(meta, fixture); err != nil {
		return Report{}, err
	}
	graphImpact, graphRefutations := computeImpact(fixture.Before, fixture.After)
	fixedPoint, fixedPointRefutations := evaluateFixedPoint(meta, fixture)
	refutations := append([]Evidence(nil), graphRefutations...)
	refutations = append(refutations, fixedPointRefutations...)
	units := sortedUnits(fixture.Units)
	plans := make([]UnitPlan, 0, len(units))
	unknowns := []Evidence{}
	for _, unit := range units {
		plan := planUnit(unit, graphImpact, graphRefutations)
		plans = append(plans, plan)
		if plan.Unknown {
			unknowns = append(unknowns, Evidence{
				Stage: "PLAN_VALIDATION_UNIT", Step: "BIND_CACHE_IDENTITY",
				Class: fixture.UnknownClass, Reason: plan.Reason, Next: plan.Next, BlockedBy: []string{unit.ID},
			})
		}
		if plan.Refuted {
			refutations = append(refutations, Evidence{
				Stage: "PLAN_VALIDATION_UNIT", Step: "PRESERVE_PROOF_STATE",
				Reason: plan.Reason, Next: plan.Next, BlockedBy: []string{unit.ID},
			})
		}
	}
	indicators := indicatorVector(fixture.Indicators, units)
	for _, indicator := range indicators {
		if indicator.State == IndicatorUnknown {
			unknowns = append(unknowns, Evidence{
				Stage: "OBSERVE_MATCHED_IDENTITY", Step: "COMPARE_BEFORE_AFTER",
				Class: fixture.UnknownClass, Reason: indicator.Reason, Next: "MEASURE_BEFORE_AND_AFTER_WITH_THE_SAME_IDENTITY",
				BlockedBy: []string{indicator.Metric},
			})
		}
	}
	decision := DecisionClosed
	if len(refutations) > 0 {
		decision = DecisionRefuted
	} else if len(unknowns) > 0 {
		decision = DecisionUnknown
	}
	fixedPoint = finalizeFixedPoint(fixedPoint, decision)
	optional, err := consumeOptionalSlicer(meta.OptionalInput, fixture.OptionalSlicer)
	if err != nil {
		return Report{}, err
	}
	operational := OperationalAudit{
		State:            OperationalState(decision, plans),
		RepositoryWrites: 0, LocalTestExecutions: 0, CrossProjectRequiredGates: 0,
		FailedRunsPreserved: true,
		FailureMutation:     "PRESERVE_FAILED_OR_COUNTEREXAMPLE; NEVER_PROMOTE_TO_CLOSED",
	}
	return Report{
		Schema: "gooo/incremental-conformance-planner/report/v1",
		CaseID: fixture.CaseID, Decision: decision, Precedence: append([]string(nil), Precedence...),
		Impact: graphImpact, Units: plans, Indicators: indicators, UnknownClass: fixture.UnknownClass, FixedPoint: fixedPoint,
		Unknowns: unknowns, Refutations: refutations, OptionalSlicer: optional,
		Operational: operational, SourceDigest: DigestBytes([]byte(fixture.CaseID)),
		ContractDigest: contractDigest, MetaDigest: meta.SourceDigest,
	}, nil
}

func validateFixture(meta Meta, fixture Fixture) error {
	if fixture.Schema != "gooo/incremental-conformance-planner/fixture/v1" || fixture.CaseID == "" {
		return errors.New("fixture must use the incremental planner fixture schema and a case_id")
	}
	if fixture.Kind != "CLOSED" && fixture.Kind != "UNKNOWN" && fixture.Kind != "REFUTED" {
		return fmt.Errorf("fixture %s has invalid kind %q", fixture.CaseID, fixture.Kind)
	}
	if len(fixture.Units) == 0 {
		return fmt.Errorf("fixture %s has no validation units", fixture.CaseID)
	}
	if fixture.Expected.Decision != fixture.Kind {
		return fmt.Errorf("fixture %s expected decision must match kind", fixture.CaseID)
	}
	if fixture.Kind == DecisionUnknown {
		if !contains(meta.UnknownClasses, fixture.UnknownClass) {
			return fmt.Errorf("fixture %s must declare a .gooo-owned UNKNOWN class", fixture.CaseID)
		}
		if fixture.Expected.UnknownClass != fixture.UnknownClass {
			return fmt.Errorf("fixture %s expected UNKNOWN class must match fixture class", fixture.CaseID)
		}
	} else if fixture.UnknownClass != "" || fixture.Expected.UnknownClass != "" {
		return fmt.Errorf("fixture %s may declare unknown_class only for UNKNOWN cases", fixture.CaseID)
	}
	if _, ok := fixedPointCaseMode(meta, fixture.CaseID); !ok && fixture.FixedPoint != nil {
		return fmt.Errorf("fixture %s has an unauthorized fixed-point declaration", fixture.CaseID)
	}
	seen := map[string]bool{}
	for _, unit := range fixture.Units {
		if unit.ID == "" || seen[unit.ID] {
			return fmt.Errorf("fixture %s has duplicate or empty validation unit", fixture.CaseID)
		}
		seen[unit.ID] = true
		if unit.Command == "" || len(unit.SemanticNodes) == 0 {
			return fmt.Errorf("validation unit %s must define command and semantic nodes", unit.ID)
		}
	}
	return nil
}

func fixedPointCaseMode(meta Meta, caseID string) (string, bool) {
	for _, item := range meta.FixedPointCases {
		if item.ID == caseID {
			return item.Mode, true
		}
	}
	return "", false
}

func evaluateFixedPoint(meta Meta, fixture Fixture) (FixedPointAssessment, []Evidence) {
	mode, authorized := fixedPointCaseMode(meta, fixture.CaseID)
	assessment := FixedPointAssessment{
		State: FixedPointNotApplicable, Declared: fixture.FixedPoint != nil && fixture.FixedPoint.Declared,
		CaseMode: mode, Rule: "FIXED_POINT_ONLY_IF_EXPLICIT", Reason: "NO_FIXED_POINT_CASE_DECLARED_BY_GOOO_AUTHORITY",
	}
	if !contains(meta.FixedPointRules, "FIXED_POINT_ONLY_IF_EXPLICIT") {
		return malformedFixedPointAssessment(assessment, "UNDECLARED_FIXED_POINT_RULE"), []Evidence{{
			Stage: "PARSE_GOOO", Step: "VALIDATE_FIXED_POINT_RULE",
			Reason: "MALFORMED_OR_IMPLICIT_FIXED_POINT_COUNTEREXAMPLE", Next: "RELEASE_CORRECTED_GOOO_AUTHORITY",
			BlockedBy: []string{"FIXED_POINT_ONLY_IF_EXPLICIT"},
		}}
	}
	if !authorized {
		if fixture.FixedPoint == nil {
			return assessment, nil
		}
		return malformedFixedPointAssessment(assessment, "FIXED_POINT_CASE_NOT_AUTHORIZED_BY_GOOO"), []Evidence{{
			Stage: "INSPECT_PROOF_STATE", Step: "VALIDATE_FIXED_POINT_DECLARATION",
			Reason: "MALFORMED_OR_IMPLICIT_FIXED_POINT_COUNTEREXAMPLE", Next: "RELEASE_CORRECTED_EXPLICIT_FIXED_POINT_CASE",
			BlockedBy: []string{fixture.CaseID},
		}}
	}
	assessment.Rule = modeRule(meta, mode)
	if assessment.Rule == "" {
		return malformedFixedPointAssessment(assessment, "UNDECLARED_FIXED_POINT_RULE"), []Evidence{{
			Stage: "PARSE_GOOO", Step: "VALIDATE_FIXED_POINT_RULE",
			Reason: "MALFORMED_OR_IMPLICIT_FIXED_POINT_COUNTEREXAMPLE", Next: "RELEASE_CORRECTED_GOOO_AUTHORITY",
			BlockedBy: []string{mode},
		}}
	}
	switch mode {
	case "EXPLICIT_FIXED_POINT", "FAIL_CLOSED_UNKNOWN":
		if fixture.FixedPoint == nil || !fixture.FixedPoint.Declared || fixture.FixedPoint.Kind != FixedPointExplicitKind || fixture.FixedPoint.Witness == "" {
			return malformedFixedPointAssessment(assessment, "MALFORMED_OR_IMPLICIT_FIXED_POINT_COUNTEREXAMPLE"), []Evidence{{
				Stage: "INSPECT_PROOF_STATE", Step: "VALIDATE_FIXED_POINT_DECLARATION",
				Reason: "MALFORMED_OR_IMPLICIT_FIXED_POINT_COUNTEREXAMPLE", Next: "RELEASE_CORRECTED_EXPLICIT_FIXED_POINT_CASE",
				BlockedBy: []string{fixture.CaseID},
			}}
		}
		assessment.Declared = true
		assessment.Reason = "EXPLICIT_FIXED_POINT_DECLARATION_ACCEPTED_FOR_EVALUATION"
		return assessment, nil
	case "MALFORMED_OR_IMPLICIT_COUNTEREXAMPLE":
		if fixture.FixedPoint == nil || (fixture.FixedPoint.Declared && fixture.FixedPoint.Kind == FixedPointExplicitKind) || (fixture.FixedPoint.Kind != FixedPointImplicitKind && fixture.FixedPoint.Kind != FixedPointMalformedKind) {
			return malformedFixedPointAssessment(assessment, "MALFORMED_OR_IMPLICIT_FIXED_POINT_COUNTEREXAMPLE"), []Evidence{{
				Stage: "INSPECT_PROOF_STATE", Step: "VALIDATE_FIXED_POINT_DECLARATION",
				Reason: "MALFORMED_OR_IMPLICIT_FIXED_POINT_COUNTEREXAMPLE", Next: "RELEASE_CORRECTED_EXPLICIT_FIXED_POINT_CASE",
				BlockedBy: []string{fixture.CaseID},
			}}
		}
		assessment.Declared = fixture.FixedPoint.Declared
		assessment.State = FixedPointRefuted
		assessment.Reason = "MALFORMED_OR_IMPLICIT_FIXED_POINT_IS_REFUTED"
		return assessment, []Evidence{{
			Stage: "INSPECT_PROOF_STATE", Step: "PRESERVE_FIXED_POINT_COUNTEREXAMPLE",
			Reason: "MALFORMED_OR_IMPLICIT_FIXED_POINT_COUNTEREXAMPLE", Next: "RELEASE_CORRECTED_EXPLICIT_FIXED_POINT_CASE",
			BlockedBy: []string{fixture.CaseID},
		}}
	default:
		return malformedFixedPointAssessment(assessment, "UNKNOWN_FIXED_POINT_CASE_MODE"), []Evidence{{
			Stage: "INSPECT_PROOF_STATE", Step: "VALIDATE_FIXED_POINT_DECLARATION",
			Reason: "MALFORMED_OR_IMPLICIT_FIXED_POINT_COUNTEREXAMPLE", Next: "RELEASE_CORRECTED_EXPLICIT_FIXED_POINT_CASE",
			BlockedBy: []string{fixture.CaseID},
		}}
	}
}

func malformedFixedPointAssessment(assessment FixedPointAssessment, reason string) FixedPointAssessment {
	assessment.State = FixedPointRefuted
	assessment.Reason = reason
	return assessment
}

func modeRule(meta Meta, mode string) string {
	rule := "FIXED_POINT_ONLY_IF_EXPLICIT"
	switch mode {
	case "FAIL_CLOSED_UNKNOWN":
		rule = "UNKNOWN_TOP_DECISION_FAIL_CLOSED"
	case "MALFORMED_OR_IMPLICIT_COUNTEREXAMPLE":
		rule = "MALFORMED_OR_IMPLICIT_FIXED_POINT_REFUTED"
	}
	if contains(meta.FixedPointRules, rule) {
		return rule
	}
	return ""
}

func finalizeFixedPoint(assessment FixedPointAssessment, decision string) FixedPointAssessment {
	if assessment.State == FixedPointRefuted || assessment.CaseMode == "" {
		return assessment
	}
	if decision == DecisionUnknown {
		assessment.State = FixedPointUnknown
		assessment.Reason = "UNKNOWN_TOP_DECISION_IS_NOT_FIXED_POINT; FAIL_CLOSED_UNKNOWN"
		return assessment
	}
	if decision == DecisionRefuted {
		assessment.State = FixedPointRefuted
		assessment.Reason = "REFUTED_TOP_DECISION_IS_NOT_FIXED_POINT"
		return assessment
	}
	if assessment.CaseMode == "EXPLICIT_FIXED_POINT" {
		assessment.State = FixedPointState
		assessment.Reason = "EXPLICIT_FIXED_POINT_DECLARATION_ACCEPTED"
		return assessment
	}
	assessment.State = FixedPointNotApplicable
	assessment.Reason = "FIXED_POINT_ONLY_IF_EXPLICIT; TOP_DECISION_NOT_AN_EXPLICIT_FIXED_POINT_CASE"
	return assessment
}

func validateDenominator(denominator Denominator) error {
	if !denominator.Fixed || denominator.Cells != 12 || denominator.Version != "v1" || len(denominator.Cases) != 12 {
		return errors.New("denominator must be the fixed 12-cell v1 contract")
	}
	for _, state := range []string{DecisionClosed, DecisionUnknown, DecisionRefuted} {
		if denominator.ProofDenominator[state] != 4 {
			return errors.New("proof denominator must be 4/4/4")
		}
	}
	for _, state := range []string{IndicatorObserved, IndicatorUnknown, IndicatorRefuted} {
		if denominator.IndicatorDenominator[state] != 4 {
			return errors.New("indicator denominator must be 4/4/4")
		}
	}
	return nil
}

func sortedUnits(units []ValidationUnit) []ValidationUnit {
	copyUnits := append([]ValidationUnit(nil), units...)
	sort.Slice(copyUnits, func(i, j int) bool { return copyUnits[i].ID < copyUnits[j].ID })
	return copyUnits
}

func planUnit(unit ValidationUnit, impact ImpactClosure, graphRefutations []Evidence) UnitPlan {
	plan := UnitPlan{UnitID: unit.ID, Planned: true, Identity: unit.CurrentIdentity, Metrics: unit.Metrics}
	if len(graphRefutations) > 0 {
		plan.Action = ActionRefuted
		plan.Refuted = true
		plan.Reason = graphRefutations[0].Reason
		plan.Next = "RELEASE_CORRECTED_SEMANTIC_GRAPH"
		return plan
	}
	missing := unit.CurrentIdentity.Missing()
	if len(missing) > 0 {
		plan.Action = ActionUnknown
		plan.Unknown = true
		plan.Missing = missing
		plan.Reason = "MISSING_CACHE_IDENTITY_COMPONENT"
		plan.Next = "COLLECT_ALL_SIX_IDENTITY_DIGESTS_THEN_EXECUTE"
		return plan
	}
	if unit.Cache == nil {
		plan.Action = ActionUnknown
		plan.Unknown = true
		plan.Reason = "MISSING_CACHE_RECEIPT"
		plan.Next = "EXECUTE_WITH_FRESH_IMMUTABLE_RECEIPT"
		return plan
	}
	plan.PriorState = unit.Cache.State
	if unit.Cache.State == ProofFailed || unit.Cache.State == ProofCounterexample {
		plan.Action = ActionRefuted
		plan.Refuted = true
		plan.Reason = "FAILED_OR_COUNTEREXAMPLE_IS_NOT_REUSABLE"
		plan.Next = "EXECUTE_REMEDIATION_AND_PRESERVE_OPERATIONAL_REFUTATION"
		return plan
	}
	if unit.Cache.State != ProofPass || !unit.Cache.Immutable || unit.Cache.ResultDigest == "" {
		plan.Action = ActionUnknown
		plan.Unknown = true
		plan.Reason = "CACHE_PROOF_NOT_IMMUTABLE_PASS"
		plan.Next = "EXECUTE_AND_EMIT_IMMUTABLE_PASS_RECEIPT"
		return plan
	}
	cacheMissing := unit.Cache.Identity.Missing()
	if len(cacheMissing) > 0 {
		plan.Action = ActionUnknown
		plan.Unknown = true
		plan.Missing = cacheMissing
		plan.Reason = "CACHED_IDENTITY_INCOMPLETE"
		plan.Next = "REFRESH_ALL_SIX_IDENTITY_DIGESTS"
		return plan
	}
	plan.Mismatched = identityMismatches(unit.CurrentIdentity, unit.Cache.Identity)
	if intersects(unit.SemanticNodes, impact.ImpactedNodes) {
		plan.Action = ActionExecute
		plan.Executed = true
		plan.Reason = "SEMANTIC_IMPACT_CLOSURE_REQUIRES_EXECUTION"
		plan.Next = "EXECUTE_AND_BIND_CURRENT_IMPACTED_PROOF"
		return plan
	}
	if len(plan.Mismatched) > 0 {
		plan.Action = ActionExecute
		plan.Executed = true
		plan.Reason = "CACHE_IDENTITY_MISMATCH_REUSE_FORBIDDEN"
		plan.Next = "EXECUTE_WITH_CURRENT_IDENTITY"
		return plan
	}
	plan.Action = ActionReuse
	plan.Reused = true
	plan.Reason = "EXACT_CACHE_IDENTITY_AND_IMMUTABLE_PASS"
	plan.Next = "NONE"
	return plan
}

func identityMismatches(current, cached CacheIdentity) []string {
	fields := []struct {
		name string
		a    string
		b    string
	}{
		{"source_digest", current.SourceDigest, cached.SourceDigest},
		{"semantic_ir_digest", current.SemanticIRDigest, cached.SemanticIRDigest},
		{"fixture_digest", current.FixtureDigest, cached.FixtureDigest},
		{"contract_digest", current.ContractDigest, cached.ContractDigest},
		{"go_toolchain_digest", current.GoToolchainDigest, cached.GoToolchainDigest},
		{"command_descriptor_digest", current.CommandDescriptorDigest, cached.CommandDescriptorDigest},
	}
	mismatched := []string{}
	for _, field := range fields {
		if field.a != field.b {
			mismatched = append(mismatched, field.name)
		}
	}
	return mismatched
}

func intersects(left, right []string) bool {
	set := map[string]bool{}
	for _, value := range right {
		set[value] = true
	}
	for _, value := range left {
		if set[value] {
			return true
		}
	}
	return false
}

func computeImpact(before, after SemanticGraph) (ImpactClosure, []Evidence) {
	beforeNodes := map[string]SemanticNode{}
	afterNodes := map[string]SemanticNode{}
	for _, node := range before.Nodes {
		beforeNodes[node.ID] = node
	}
	for _, node := range after.Nodes {
		afterNodes[node.ID] = node
	}
	changed := map[string]bool{}
	for id, oldNode := range beforeNodes {
		newNode, ok := afterNodes[id]
		if !ok || oldNode.Digest != newNode.Digest {
			changed[id] = true
		}
	}
	for id := range afterNodes {
		if _, ok := beforeNodes[id]; !ok {
			changed[id] = true
		}
	}
	beforeEdges := edgeSet(before.Edges)
	afterEdges := edgeSet(after.Edges)
	for key, edge := range beforeEdges {
		if _, ok := afterEdges[key]; !ok {
			changed[edge.From] = true
			changed[edge.To] = true
		}
	}
	for key, edge := range afterEdges {
		if _, ok := beforeEdges[key]; !ok {
			changed[edge.From] = true
			changed[edge.To] = true
		}
	}
	changedNodes := sortedMapKeys(changed)
	impacted := map[string]bool{}
	for _, id := range changedNodes {
		impacted[id] = true
	}
	queue := append([]string(nil), changedNodes...)
	closureEdges := map[string]bool{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, edge := range after.Edges {
			if edge.From != current {
				continue
			}
			closureEdges[edge.From+"->"+edge.To] = true
			if !impacted[edge.To] {
				impacted[edge.To] = true
				queue = append(queue, edge.To)
			}
		}
	}
	refutations := validateGraph(after)
	edges := sortedMapKeys(closureEdges)
	return ImpactClosure{ChangedNodes: changedNodes, ImpactedNodes: sortedMapKeys(impacted), Edges: edges}, refutations
}

func edgeSet(edges []SemanticDependency) map[string]SemanticDependency {
	result := map[string]SemanticDependency{}
	for _, edge := range edges {
		result[edge.From+"->"+edge.To+"#"+edge.Relation] = edge
	}
	return result
}

func validateGraph(graph SemanticGraph) []Evidence {
	nodes := map[string]SemanticNode{}
	for _, node := range graph.Nodes {
		nodes[node.ID] = node
	}
	refutations := []Evidence{}
	for _, edge := range graph.Edges {
		from, fromOK := nodes[edge.From]
		to, toOK := nodes[edge.To]
		if !fromOK || !toOK || (edge.FromDigest != "" && edge.FromDigest != from.Digest) || (edge.ToDigest != "" && edge.ToDigest != to.Digest) {
			refutations = append(refutations, Evidence{
				Stage: "BIND_SEMANTIC_DEPENDENCIES", Step: "VERIFY_DEPENDENCY_DIGEST",
				Reason: "SEMANTIC_DEPENDENCY_CONTRADICTION", Next: "RELEASE_CORRECTED_SEMANTIC_GRAPH",
				BlockedBy: []string{edge.From + "->" + edge.To},
			})
		}
	}
	return refutations
}

func sortedMapKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func indicatorVector(pair MetricPair, units []ValidationUnit) []IndicatorObservation {
	unitID := "matched-run"
	if len(units) > 0 {
		unitID = units[0].ID
	}
	identityPair := pair.BeforeKnown && pair.AfterKnown && len(pair.BeforeID.Missing()) == 0 && len(pair.AfterID.Missing()) == 0 && pair.BeforeID.Equal(pair.AfterID)
	metrics := []struct {
		name   string
		before *int64
		after  *int64
	}{
		{"build_ms", pair.Before.BuildMS, pair.After.BuildMS},
		{"test_ms", pair.Before.TestMS, pair.After.TestMS},
		{"wall_ms", pair.Before.WallMS, pair.After.WallMS},
		{"peak_rss_kib", pair.Before.PeakRSSKiB, pair.After.PeakRSSKiB},
	}
	result := make([]IndicatorObservation, 0, len(metrics))
	for _, metric := range metrics {
		observation := IndicatorObservation{UnitID: unitID, Metric: metric.name, Before: metric.before, After: metric.after, IdentityPair: identityPair}
		if pair.RunState == ProofFailed || pair.RunState == ProofCounterexample {
			observation.State = IndicatorRefuted
			observation.Reason = "FAILED_RUN_IS_OPERATIONAL_REFUTATION"
		} else if !identityPair || metric.before == nil || metric.after == nil {
			observation.State = IndicatorUnknown
			observation.Reason = "NO_MATCHED_BEFORE_AFTER_IDENTITY_PAIR"
		} else {
			delta := *metric.after - *metric.before
			observation.SignedDelta = &delta
			observation.State = IndicatorObserved
			observation.Reason = "EXACT_SAME_IDENTITY_BEFORE_AFTER_PAIR"
		}
		result = append(result, observation)
	}
	return result
}

func consumeOptionalSlicer(meta OptionalInput, supplied *OptionalSlicerInput) (OptionalSlicerStatus, error) {
	status := OptionalSlicerStatus{Required: meta.Required, CrossProjectGate: meta.CrossProjectGate}
	if supplied == nil {
		status.Reason = "OPTIONAL_INPUT_NOT_SUPPLIED; LOCAL_PLANNER_REMAINS_SELF_CONTAINED"
		return status, nil
	}
	if supplied.Release != meta.Release || !strings.HasPrefix(supplied.Digest, "sha256:") || len(supplied.Digest) < len("sha256:")+8 {
		return OptionalSlicerStatus{}, fmt.Errorf("optional slicer input must be release %s with a digest-pinned sha256 value", meta.Release)
	}
	status.Consumed = true
	status.Release = supplied.Release
	status.Digest = supplied.Digest
	status.Reason = "DIGEST_PINNED_OPTIONAL_INPUT_CONSUMED; NOT_COPIED; NOT_A_REQUIRED_CROSS_PROJECT_GATE"
	return status, nil
}

func OperationalState(decision string, plans []UnitPlan) string {
	if decision == DecisionRefuted {
		return OperationalRefuted
	}
	if decision == DecisionUnknown {
		return "PLANNER_UNKNOWN"
	}
	for _, plan := range plans {
		if plan.PriorState == ProofFailed || plan.PriorState == ProofCounterexample {
			return OperationalRefuted
		}
	}
	return "PLANNER_CLOSED"
}

func writeReport(outputDir string, report Report) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outputDir, "report.json"), report); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outputDir, "unit-vector.json"), report.Units); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outputDir, "indicator-vector.json"), report.Indicators); err != nil {
		return err
	}
	return writeText(filepath.Join(outputDir, "human-report.md"), RenderReport(report))
}

func RunSuite(options SuiteOptions) (SuiteReport, error) {
	if !filepath.IsAbs(options.OutputDir) {
		return SuiteReport{}, errors.New("suite output directory must be an absolute caller-owned path")
	}
	meta, err := ParseMeta(options.MetaPath)
	if err != nil {
		return SuiteReport{}, err
	}
	denominator, err := LoadDenominator(options.ContractPath)
	if err != nil {
		return SuiteReport{}, err
	}
	contractData, err := os.ReadFile(options.ContractPath)
	if err != nil {
		return SuiteReport{}, err
	}
	caseResults := make([]CaseResult, 0, len(denominator.Cases))
	for _, item := range sortedCaseSpecs(denominator.Cases) {
		fixturePath := pathFrom(options.CasesRoot, item.Source)
		fixture, err := LoadFixture(fixturePath)
		if err != nil {
			return SuiteReport{}, fmt.Errorf("case %s: %w", item.ID, err)
		}
		report, err := Plan(meta, fixture, DigestBytes(contractData))
		if err != nil {
			return SuiteReport{}, fmt.Errorf("case %s: %w", item.ID, err)
		}
		caseDir := filepath.Join(options.OutputDir, "cases", item.ID)
		if err := writeReport(caseDir, report); err != nil {
			return SuiteReport{}, err
		}
		match := report.Decision == item.Expected && fixture.Expected.Decision == item.Expected
		if match && report.UnknownClass != fixture.Expected.UnknownClass {
			match = false
		}
		if match {
			mode, authorized := fixedPointCaseMode(meta, fixture.CaseID)
			if authorized {
				switch mode {
				case "EXPLICIT_FIXED_POINT":
					match = report.FixedPoint.State == FixedPointState && report.Decision == DecisionClosed
				case "FAIL_CLOSED_UNKNOWN":
					match = report.FixedPoint.State == FixedPointUnknown && report.Decision == DecisionUnknown
				case "MALFORMED_OR_IMPLICIT_COUNTEREXAMPLE":
					match = report.FixedPoint.State == FixedPointRefuted && report.Decision == DecisionRefuted
				}
			}
		}
		if match {
			byID := map[string]string{}
			for _, unit := range report.Units {
				byID[unit.UnitID] = unit.Action
			}
			for unitID, expectedAction := range fixture.Expected.Actions {
				if byID[unitID] != expectedAction {
					match = false
					break
				}
			}
		}
		caseResults = append(caseResults, CaseResult{ID: item.ID, Expected: item.Expected, Actual: report.Decision, Match: match, Report: filepath.ToSlash(filepath.Join("cases", item.ID, "human-report.md"))})
	}
	decision := DecisionClosed
	for _, item := range caseResults {
		if !item.Match {
			decision = DecisionRefuted
			break
		}
	}
	suite := SuiteReport{
		Schema:   "gooo/incremental-conformance-planner/suite-report/v1",
		Decision: decision, Cases: caseResults,
		Operational: OperationalAudit{State: OperationalState(decision, nil), RepositoryWrites: 0, LocalTestExecutions: 0, CrossProjectRequiredGates: 0, FailedRunsPreserved: true, FailureMutation: "PRESERVE_FAILED_OR_COUNTEREXAMPLE; NEVER_PROMOTE_TO_CLOSED"},
	}
	if err := writeJSON(filepath.Join(options.OutputDir, "suite-report.json"), suite); err != nil {
		return SuiteReport{}, err
	}
	if err := writeText(filepath.Join(options.OutputDir, "human-report.md"), RenderSuiteReport(denominator, suite)); err != nil {
		return SuiteReport{}, err
	}
	return suite, nil
}
