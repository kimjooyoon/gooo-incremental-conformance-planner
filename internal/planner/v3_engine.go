package planner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type V3SuiteOptions struct {
	MetaPath           string
	ContractPath       string
	CasesRoot          string
	OutputDir          string
	ActionsReceiptPath string
}

func RunV3Suite(options V3SuiteOptions) (V3SuiteReport, error) {
	if !filepath.IsAbs(options.OutputDir) {
		return V3SuiteReport{}, errors.New("v3 suite output directory must be an absolute caller-owned path")
	}
	source, sourceDigest, err := ParseV3Source(options.MetaPath)
	if err != nil {
		return V3SuiteReport{}, fmt.Errorf("load v3 .gooo: %w", err)
	}
	ir, err := BuildV3SemanticIR(source, options.MetaPath, sourceDigest)
	if err != nil {
		return V3SuiteReport{}, fmt.Errorf("build v3 semantic IR: %w", err)
	}
	contract, contractDigest, err := LoadV3Contract(options.ContractPath)
	if err != nil {
		return V3SuiteReport{}, err
	}
	if !sameV3Activities(source.Activities, contract.Activities) {
		return V3SuiteReport{}, errors.New("v3 contract activities do not match the authoritative .gooo source")
	}
	var actions *V3ActionsReceipt
	if options.ActionsReceiptPath != "" {
		receipt, err := LoadV3ActionsReceipt(options.ActionsReceiptPath)
		if err != nil {
			return V3SuiteReport{}, err
		}
		if receipt.SemanticIRDigest == "" {
			receipt.SemanticIRDigest = ir.Digest
		}
		actions = &receipt
	}
	if err := os.MkdirAll(options.OutputDir, 0o755); err != nil {
		return V3SuiteReport{}, err
	}
	if err := writeJSON(filepath.Join(options.OutputDir, "semantic-ir.json"), ir); err != nil {
		return V3SuiteReport{}, err
	}
	evaluator := V3EvaluatorArtifact{
		Schema: "gooo/incremental-conformance-planner/evaluator/v3", SemanticIRDigest: ir.Digest,
		Digest: ir.EvaluatorDigest, IdentityFields: ir.IdentityFields, PriorRunFields: ir.PriorRunFields,
		ObservationFields: ir.ObservationFields, UnknownFields: ir.UnknownFields, Policies: ir.Policies, Activities: ir.Activities,
	}
	if err := writeJSON(filepath.Join(options.OutputDir, "generated", "evaluator.json"), evaluator); err != nil {
		return V3SuiteReport{}, err
	}
	if actions != nil {
		if err := writeJSON(filepath.Join(options.OutputDir, "actions-receipt.json"), actions); err != nil {
			return V3SuiteReport{}, err
		}
	}

	caseResults := make([]V3CaseResult, 0, len(contract.Scenarios))
	legacyExactReuseClosed := true
	for _, spec := range sortedV3Cases(contract.Scenarios) {
		fixture, err := LoadV3Fixture(pathFrom(options.CasesRoot, spec.Source))
		if err != nil {
			return V3SuiteReport{}, fmt.Errorf("case %s: %w", spec.ID, err)
		}
		projection, err := ProjectV3(ir, contract, fixture)
		if err != nil {
			return V3SuiteReport{}, fmt.Errorf("project case %s: %w", spec.ID, err)
		}
		report := EvaluateV3(ir, contract, fixture, projection, contractDigest, actions)
		caseDir := filepath.Join(options.OutputDir, "cases", spec.ID)
		if err := writeV3Report(caseDir, report); err != nil {
			return V3SuiteReport{}, err
		}
		match := report.Decision == spec.Expected && report.Decision == fixture.Expected.Decision
		plans := map[string]string{}
		for _, activity := range report.Activities {
			plans[activity.ActivityID] = activity.Action
		}
		for activityID, expectedAction := range fixture.Expected.Activities {
			if plans[activityID] != expectedAction {
				match = false
				break
			}
		}
		if spec.ID == "exact-reuse" {
			if report.Decision != V3DecisionClosed || len(report.Activities) != 12 {
				legacyExactReuseClosed = false
			}
			for _, activity := range report.Activities {
				if activity.Action != V3ActionReusedClosed || !activity.ReusableEvidence || activity.MustExecute {
					legacyExactReuseClosed = false
				}
			}
		}
		caseResults = append(caseResults, V3CaseResult{ID: spec.ID, Expected: spec.Expected, Actual: report.Decision, Match: match, Report: filepath.ToSlash(filepath.Join("cases", spec.ID, "human-report.md"))})
	}

	decision := V3DecisionClosed
	for _, result := range caseResults {
		if !result.Match {
			decision = V3DecisionRefuted
			break
		}
	}
	if !legacyExactReuseClosed {
		decision = V3DecisionRefuted
	}
	actionsMetricState := "OBSERVED"
	missingActionsMetrics := []string{}
	if actions != nil {
		missingActionsMetrics = missingV3ActionsMetrics(*actions)
		if len(missingActionsMetrics) > 0 && decision != V3DecisionRefuted {
			actionsMetricState = V3DecisionUnknown
			decision = V3DecisionUnknown
		}
	}
	suite := V3SuiteReport{
		Schema: "gooo/incremental-conformance-planner/v3-suite-report/v1", Decision: decision,
		Contract: contract.ID, ContractDigest: contractDigest, TotalActivities: contract.TargetActivities,
		ProofTotals: contract.ProofTotals, IndicatorTotals: contract.IndicatorTotals, Cases: caseResults,
		LegacyExactReuseClosed: legacyExactReuseClosed, ActionsReceipt: actions, ActionsMetricState: actionsMetricState,
		MissingActionsMetrics: missingActionsMetrics,
		Operational: V3Operational{RepositoryWrites: 0, LocalTestExecutions: 0, CrossProjectRequiredGates: 0, FailedRunsPreserved: true, OutputLocation: options.OutputDir, VerificationAuthority: "GitHub Actions"},
	}
	if err := writeJSON(filepath.Join(options.OutputDir, "suite-report.json"), suite); err != nil {
		return V3SuiteReport{}, err
	}
	if err := writeText(filepath.Join(options.OutputDir, "human-report.md"), RenderV3SuiteReport(suite)); err != nil {
		return V3SuiteReport{}, err
	}
	return suite, nil
}

type V3EvaluatorArtifact struct {
	Schema            string                 `json:"schema"`
	SemanticIRDigest  string                 `json:"semantic_ir_digest"`
	Digest            string                 `json:"digest"`
	IdentityFields    []string               `json:"identity_fields"`
	PriorRunFields    []string               `json:"prior_run_fields"`
	ObservationFields []string               `json:"observation_fields"`
	UnknownFields     []string               `json:"unknown_fields"`
	Policies          []string               `json:"policies"`
	Activities        []V2ActivityDescriptor `json:"activities"`
}

func missingV3ActionsMetrics(receipt V3ActionsReceipt) []string {
	missing := []string{}
	if receipt.RunIdentity.RunID == "" {
		missing = append(missing, "run_identity.run_id")
	}
	if receipt.RunIdentity.CommitSHA == "" {
		missing = append(missing, "run_identity.commit_sha")
	}
	if receipt.RunIdentity.ReceiptDigest == "" {
		missing = append(missing, "run_identity.receipt_digest")
	}
	if receipt.TestIdentity == "" {
		missing = append(missing, "test_identity")
	}
	if receipt.ConformanceIdentity == "" {
		missing = append(missing, "conformance_identity")
	}
	if receipt.InputDigest == "" {
		missing = append(missing, "input_digest")
	}
	if receipt.ToolchainDigest == "" {
		missing = append(missing, "toolchain_digest")
	}
	if receipt.SemanticIRDigest == "" {
		missing = append(missing, "semantic_ir_digest")
	}
	if receipt.BuildMS == nil {
		missing = append(missing, "build_ms")
	}
	if receipt.TestMS == nil {
		missing = append(missing, "test_ms")
	}
	if receipt.WallMS == nil {
		missing = append(missing, "wall_ms")
	}
	if receipt.PeakRSSKiB == nil {
		missing = append(missing, "peak_rss_kib")
	}
	if receipt.CacheHit == nil {
		missing = append(missing, "cache_hit")
	}
	if receipt.CacheMiss == nil {
		missing = append(missing, "cache_miss")
	}
	if receipt.CacheHits == nil {
		missing = append(missing, "cache_hits")
	}
	if receipt.CacheMisses == nil {
		missing = append(missing, "cache_misses")
	}
	for _, activity := range receipt.Activities {
		if activity.DurationMS == nil {
			missing = append(missing, activity.Status+".duration_ms")
		}
	}
	return uniqueStrings(missing)
}

func sameV3Activities(source, contract []V2ActivityDescriptor) bool {
	if len(source) != len(contract) {
		return false
	}
	byID := map[string]V2ActivityDescriptor{}
	for _, activity := range source {
		byID[activity.ID] = activity
	}
	for _, activity := range contract {
		other, ok := byID[activity.ID]
		if !ok || other.Ordinal != activity.Ordinal || other.Activity != activity.Activity || other.Stage != activity.Stage || other.Step != activity.Step || other.ProofChoice != activity.ProofChoice || other.IndicatorClass != activity.IndicatorClass || !sameStringSet(other.DependsOn, activity.DependsOn) {
			return false
		}
	}
	return true
}

func ProjectV3(ir V3SemanticIR, contract V3Contract, fixture V3Fixture) (V3Projection, error) {
	if fixture.Expected.Decision == "" {
		return V3Projection{}, fmt.Errorf("fixture %s has no expected decision", fixture.CaseID)
	}
	impactNodes, impactEdges, graphReasons := computeV3Impact(fixture.Before, fixture.After, fixture.SemanticChange)
	inputByID := map[string]V3ActivityInput{}
	for _, input := range fixture.Activities {
		if input.ActivityID == "" || inputByID[input.ActivityID].ActivityID != "" {
			return V3Projection{}, fmt.Errorf("fixture %s has a duplicate or empty activity input", fixture.CaseID)
		}
		if input.CurrentIdentity == nil {
			identity := fixture.IdentityDefaults
			input.CurrentIdentity = &identity
		}
		if input.PriorEvidence == nil && fixture.PriorEvidenceDefaults != nil {
			prior := *fixture.PriorEvidenceDefaults
			input.PriorEvidence = &prior
		}
		if input.Observation == nil {
			observation := fixture.ObservationDefaults
			input.Observation = &observation
		}
		inputByID[input.ActivityID] = input
	}
	projected := make([]V3ProjectedActivity, 0, len(contract.Activities))
	for _, descriptor := range contract.Activities {
		input, ok := inputByID[descriptor.ID]
		if !ok {
			return V3Projection{}, fmt.Errorf("fixture %s is missing activity %s", fixture.CaseID, descriptor.ID)
		}
		observation := V3Observation{}
		if input.Observation != nil {
			observation = *input.Observation
		}
		identity := V3VerificationIdentity{}
		if input.CurrentIdentity != nil {
			identity = *input.CurrentIdentity
		}
		projected = append(projected, V3ProjectedActivity{Descriptor: descriptor, Input: input, Identity: identity, Prior: input.PriorEvidence, Observation: observation})
	}
	if len(inputByID) != len(contract.Activities) {
		return V3Projection{}, fmt.Errorf("fixture %s has activity inputs outside the v3 contract", fixture.CaseID)
	}
	return V3Projection{CaseID: fixture.CaseID, Change: fixture.SemanticChange, ImpactNodes: impactNodes, ImpactEdges: impactEdges, GraphReasons: graphReasons, Activities: projected}, nil
}

func EvaluateV3(ir V3SemanticIR, contract V3Contract, fixture V3Fixture, projection V3Projection, contractDigest string, actions *V3ActionsReceipt) V3Report {
	fixtureDigest := DigestBytes([]byte(fixture.FixtureAnchor))
	plans := make([]V3ActivityPlan, 0, len(projection.Activities))
	unknowns := []V3UnknownRecord{}
	refutedReasons := append([]string(nil), projection.GraphReasons...)
	for _, activity := range projection.Activities {
		plan := evaluateV3Activity(ir, contractDigest, fixtureDigest, projection, activity)
		plans = append(plans, plan)
		if plan.Unknown && plan.UnknownRecord != nil {
			unknowns = append(unknowns, *plan.UnknownRecord)
		}
		if plan.Refuted {
			refutedReasons = append(refutedReasons, plan.ActivityID+":"+plan.Reason)
		}
	}
	indicators := evaluateV3Indicators(fixture.Indicators)
	for _, indicator := range indicators {
		if indicator.State == V3IndicatorUnknown && indicator.UnknownRecord != nil {
			unknowns = append(unknowns, *indicator.UnknownRecord)
		}
		if indicator.State == V3IndicatorRefuted {
			refutedReasons = append(refutedReasons, indicator.Metric+":"+indicator.Reason)
		}
	}
	fixedPointState := evaluateV3FixedPoint(fixture)
	if fixedPointState == V3FixedPointRefuted {
		refutedReasons = append(refutedReasons, "FIXED_POINT:MALFORMED_OR_IMPLICIT_FIXED_POINT_COUNTEREXAMPLE")
	}
	decision := V3DecisionClosed
	if len(refutedReasons) > 0 {
		decision = V3DecisionRefuted
	} else if len(unknowns) > 0 {
		decision = V3DecisionUnknown
	}
	return V3Report{
		Schema: "gooo/incremental-conformance-planner/v3-report/v1", CaseID: fixture.CaseID, Decision: decision,
		Precedence: []string{V3DecisionRefuted, V3DecisionUnknown, V3DecisionClosed}, SemanticIR: ir,
		SemanticChange: fixture.SemanticChange, ImpactNodes: projection.ImpactNodes, ImpactEdges: projection.ImpactEdges,
		Activities: plans, Summary: summarizeV3(plans), Indicators: indicators, Unknowns: unknowns, RefutedReasons: refutedReasons,
		ContractDigest: contractDigest, FixtureDigest: fixtureDigest, EvaluatorDigest: ir.EvaluatorDigest, ActionsReceipt: actions,
		FixedPointState: fixedPointState,
		Operational: V3Operational{RepositoryWrites: 0, LocalTestExecutions: 0, CrossProjectRequiredGates: 0, FailedRunsPreserved: true, OutputLocation: "caller-owned", VerificationAuthority: "GitHub Actions"},
	}
}

func evaluateV3Activity(ir V3SemanticIR, contractDigest, fixtureDigest string, projection V3Projection, activity V3ProjectedActivity) V3ActivityPlan {
	identity := activity.Identity
	plan := V3ActivityPlan{
		ActivityID: activity.Descriptor.ID, Activity: activity.Descriptor.Activity, ProofChoice: activity.Descriptor.ProofChoice,
		IndicatorClass: activity.Descriptor.IndicatorClass, Action: V3ActionUnknown, MustExecute: false,
		TestIdentity: identity.TestIdentity, ConformanceIdentity: identity.ConformanceIdentity, InputDigest: identity.InputDigest,
		ToolchainDigest: identity.ToolchainDigest, SemanticIRDigest: identity.SemanticIRDigest, BuildMS: activity.Observation.BuildMS,
		TestMS: activity.Observation.TestMS, WallMS: activity.Observation.WallMS, PeakRSSKiB: activity.Observation.PeakRSSKiB,
		CacheHit: activity.Observation.CacheHit, CacheMiss: activity.Observation.CacheMiss, ReuseClaimState: V3DecisionUnknown,
		MeasurementState: measurementState(activity.Observation),
	}
	if activity.Prior != nil {
		prior := activity.Prior.PriorRun
		plan.PriorRunIdentity = &prior
	}
	if len(projection.GraphReasons) > 0 {
		return v3RefutedPlan(plan, V3RefutedGraph, "RELEASE_CORRECTED_SEMANTIC_OR_PROOF_GRAPH")
	}
	if missing := identity.Missing(); len(missing) > 0 {
		return v3UnknownPlan(plan, V3UnknownRecord{Stage: "BINDING", Step: "BIND_TEST_CONFORMANCE_IDENTITY", Reason: "REQUIRED_TEST_CONFORMANCE_INPUT_IDENTITY_MISSING", UnknownClass: V3UnknownMissingIdentity, NextOperation: "COLLECT_EXACT_TEST_CONFORMANCE_INPUT_TOOLCHAIN_SEMANTIC_IR_IDENTITY", BlockedBy: missing})
	}
	if identity.ToolchainDigest != ir.ToolchainDigest || identity.SemanticIRDigest != ir.Digest || identity.ContractDigest != contractDigest || identity.EvaluatorDigest != ir.EvaluatorDigest || identity.FixtureDigest != fixtureDigest {
		plan.MismatchedFields = v3AuthorityMismatches(identity, ir, contractDigest, fixtureDigest)
		return v3RequiredOrUnknown(plan, activity, "CURRENT_IDENTITY_CHANGED_REQUIRES_CURRENT_REVERIFICATION")
	}
	prior := activity.Prior
	if prior == nil {
		return v3UnknownPlan(plan, V3UnknownRecord{Stage: "RECEIPT", Step: "BIND_PRIOR_RUN_IDENTITY", Reason: "PRIOR_RUN_IDENTITY_NOT_AVAILABLE", UnknownClass: V3UnknownMissingPriorRun, NextOperation: "OBTAIN_AN_IMMUTABLE_PRIOR_PASS_RECEIPT", BlockedBy: []string{activity.Descriptor.ID}})
	}
	if prior.ProofSource == "EVALUATOR_SELF_APPROVAL" {
		return v3RefutedPlan(plan, V3RefutedEvaluatorSelfApproval, "OBTAIN_INDEPENDENT_CI_PROOF")
	}
	if prior.ReceiptDigest != "" && prior.ContentDigest != "" && prior.ReceiptDigest != prior.ContentDigest {
		return v3RefutedPlan(plan, V3RefutedForgedReceipt, "PRESERVE_THE_FORGED_RECEIPT_AND_RUN_CURRENT_ACTIVITY")
	}
	if missing := prior.Identity.Missing(); len(missing) > 0 {
		return v3UnknownPlan(plan, V3UnknownRecord{Stage: "RECEIPT", Step: "VERIFY_PRIOR_RECEIPT_PROVENANCE", Reason: "PRIOR_RECEIPT_IDENTITY_INCOMPLETE", UnknownClass: V3UnknownMissingProvenance, NextOperation: "RESTORE_COMPLETE_PRIOR_RECEIPT_PROVENANCE", BlockedBy: missing})
	}
	if missing := prior.PriorRun.Missing(); len(missing) > 0 {
		return v3UnknownPlan(plan, V3UnknownRecord{Stage: "RECEIPT", Step: "VERIFY_PRIOR_RUN_IDENTITY", Reason: "PRIOR_RUN_IDENTITY_INCOMPLETE", UnknownClass: V3UnknownMissingPriorRun, NextOperation: "RESTORE_PRIOR_RUN_COMMIT_AND_RECEIPT_IDENTITY", BlockedBy: missing})
	}
	if prior.ReceiptDigest == "" || prior.ContentDigest == "" || prior.ResultDigest == "" {
		return v3UnknownPlan(plan, V3UnknownRecord{Stage: "RECEIPT", Step: "VERIFY_PRIOR_IMMUTABLE_RECEIPT", Reason: "PRIOR_IMMUTABLE_RECEIPT_PROVENANCE_INCOMPLETE", UnknownClass: V3UnknownMissingReceipt, NextOperation: "EMIT_AN_IMMUTABLE_PASS_RECEIPT_WITH_PRIOR_RUN_IDENTITY", BlockedBy: []string{activity.Descriptor.ID}})
	}
	if prior.ReceiptDigest != prior.ContentDigest || !prior.Immutable {
		return v3RefutedPlan(plan, V3RefutedForgedReceipt, "PRESERVE_THE_FORGED_RECEIPT_AND_RUN_CURRENT_ACTIVITY")
	}
	if prior.State != ProofPass {
		return v3RefutedPlan(plan, "PRIOR_PROOF_REFUTED", "PRESERVE_PRIOR_FAILURE_AND_EXECUTE_CURRENT_ACTIVITY")
	}
	if !prior.Identity.Equal(identity) {
		plan.MismatchedFields = v3IdentityMismatches(identity, prior.Identity)
		return v3RequiredOrUnknown(plan, activity, "PRIOR_IDENTITY_CHANGED_REQUIRES_CURRENT_REVERIFICATION")
	}
	impacted := intersects(activity.Input.SemanticNodes, projection.ImpactNodes)
	if (impacted || len(plan.MismatchedFields) > 0) && activity.Observation.Status == "SKIPPED_WITH_PROOF" {
		return v3RefutedPlan(plan, V3RefutedAffectedSkip, "EXECUTE_EVERY_IMPACTED_OR_IDENTITY_CHANGED_ACTIVITY")
	}
	if impacted {
		return v3RequiredOrUnknown(plan, activity, "SEMANTIC_IMPACT_REQUIRES_CURRENT_REVERIFICATION")
	}
	if activity.Observation.Status == "SKIPPED_WITH_PROOF" {
		plan.Action = V3ActionReusedClosed
		plan.ReusableEvidence = true
		plan.AlreadyVerified = true
		plan.SkippedWithProof = true
		plan.ReuseClaimState = V3DecisionClosed
		plan.Reason = "EXACT_TEST_CONFORMANCE_INPUT_TOOLCHAIN_SEMANTIC_IR_IDENTITY_AND_INDEPENDENT_IMMUTABLE_PASS_RECEIPT"
		plan.NextOperation = "NONE"
		return plan
	}
	return v3UnknownPlan(plan, V3UnknownRecord{Stage: "OBSERVATION", Step: "CONFIRM_CURRENT_ACTIVITY_RESULT", Reason: "CURRENT_ACTIVITY_RESULT_NOT_OBSERVED", UnknownClass: V3UnknownMissingMetrics, NextOperation: "EXECUTE_AND_RECORD_THE_SELECTED_ACTIVITY", BlockedBy: []string{activity.Descriptor.ID}})
}

func v3RequiredOrUnknown(plan V3ActivityPlan, activity V3ProjectedActivity, reason string) V3ActivityPlan {
	plan.RequiredRun = true
	plan.MustExecute = true
	plan.ReuseClaimState = V3DecisionUnknown
	plan.Reason = reason
	plan.NextOperation = "EXECUTE_AND_RECORD_THE_SELECTED_ACTIVITY"
	if activity.Observation.Status == "EXECUTED" {
		plan.Action = V3ActionRequiredRun
		plan.Executed = true
		return plan
	}
	return v3UnknownPlan(plan, V3UnknownRecord{Stage: "PLAN", Step: "SELECT_NEXT_CI_VALIDATION_SET", Reason: "REQUIRED_ACTIVITY_NOT_CURRENTLY_EXECUTED", UnknownClass: V3UnknownMissingMetrics, NextOperation: "EXECUTE_AND_RECORD_THE_SELECTED_ACTIVITY", BlockedBy: []string{plan.ActivityID}})
}

func v3UnknownPlan(plan V3ActivityPlan, record V3UnknownRecord) V3ActivityPlan {
	plan.Action = V3ActionUnknown
	plan.Unknown = true
	plan.ReuseClaimState = V3DecisionUnknown
	plan.UnknownRecord = &record
	plan.Reason = record.Reason
	plan.NextOperation = record.NextOperation
	return plan
}

func v3RefutedPlan(plan V3ActivityPlan, reason, next string) V3ActivityPlan {
	plan.Action = V3ActionRefuted
	plan.Refuted = true
	plan.ReuseClaimState = V3DecisionRefuted
	plan.Reason = reason
	plan.NextOperation = next
	return plan
}

func v3AuthorityMismatches(identity V3VerificationIdentity, ir V3SemanticIR, contractDigest, fixtureDigest string) []string {
	result := []string{}
	if identity.ToolchainDigest != ir.ToolchainDigest {
		result = append(result, "toolchain_digest")
	}
	if identity.SemanticIRDigest != ir.Digest {
		result = append(result, "semantic_ir_digest")
	}
	if identity.ContractDigest != contractDigest {
		result = append(result, "contract_digest")
	}
	if identity.FixtureDigest != fixtureDigest {
		result = append(result, "fixture_digest")
	}
	if identity.EvaluatorDigest != ir.EvaluatorDigest {
		result = append(result, "evaluator_digest")
	}
	return result
}

func v3IdentityMismatches(current, prior V3VerificationIdentity) []string {
	result := []string{}
	values := []struct {
		name string
		left string
		right string
	}{
		{"test_identity", current.TestIdentity, prior.TestIdentity}, {"conformance_identity", current.ConformanceIdentity, prior.ConformanceIdentity},
		{"scenario_identity", current.ScenarioIdentity, prior.ScenarioIdentity}, {"input_digest", current.InputDigest, prior.InputDigest},
		{"toolchain_digest", current.ToolchainDigest, prior.ToolchainDigest}, {"semantic_ir_digest", current.SemanticIRDigest, prior.SemanticIRDigest},
		{"source_digest", current.SourceDigest, prior.SourceDigest}, {"fixture_digest", current.FixtureDigest, prior.FixtureDigest},
		{"contract_digest", current.ContractDigest, prior.ContractDigest}, {"evaluator_digest", current.EvaluatorDigest, prior.EvaluatorDigest},
	}
	for _, value := range values {
		if value.left != value.right {
			result = append(result, value.name)
		}
	}
	return result
}

func measurementState(observation V3Observation) string {
	if observation.BuildMS == nil || observation.TestMS == nil || observation.WallMS == nil || observation.PeakRSSKiB == nil || observation.CacheHit == nil || observation.CacheMiss == nil {
		return V3DecisionUnknown
	}
	return V3IndicatorObserved
}

func evaluateV3Indicators(pair V3MetricPair) []V3IndicatorObservation {
	sameIdentity := pair.BeforeKnown && pair.AfterKnown && pair.ScenarioIdentity != "" && pair.BeforeIdentity.Equal(pair.AfterIdentity) && pair.BeforeIdentity.ScenarioIdentity == pair.ScenarioIdentity && pair.AfterIdentity.ScenarioIdentity == pair.ScenarioIdentity && len(pair.BeforeIdentity.Missing()) == 0 && len(pair.AfterIdentity.Missing()) == 0 && len(pair.BeforePriorRun.Missing()) == 0 && len(pair.AfterPriorRun.Missing()) == 0
	metrics := []struct {
		name          string
		before, after *int64
	}{
		{name: "build_ms", before: pair.Before.BuildMS, after: pair.After.BuildMS},
		{name: "test_ms", before: pair.Before.TestMS, after: pair.After.TestMS},
		{name: "wall_ms", before: pair.Before.WallMS, after: pair.After.WallMS},
		{name: "peak_rss_kib", before: pair.Before.PeakRSSKiB, after: pair.After.PeakRSSKiB},
	}
	result := make([]V3IndicatorObservation, 0, len(metrics))
	for _, metric := range metrics {
		observation := V3IndicatorObservation{Metric: metric.name, Before: metric.before, After: metric.after, SameIdentity: sameIdentity}
		if pair.RunState == ProofFailed || pair.RunState == ProofCounterexample {
			observation.State = V3IndicatorRefuted
			observation.Reason = "FAILED_ACTIONS_RECEIPT_IS_OPERATIONAL_REFUTATION"
		} else if !sameIdentity || metric.before == nil || metric.after == nil {
			class := V3UnknownMissingIdentity
			if metric.before == nil || metric.after == nil {
				class = V3UnknownMissingMetrics
			}
			observation.State = V3IndicatorUnknown
			observation.Reason = "EXACT_MATCHED_BEFORE_AFTER_PAIR_REQUIRED"
			observation.UnknownRecord = &V3UnknownRecord{Stage: "OBSERVATION", Step: "COMPARE_EXACT_BEFORE_AFTER_PAIR", Reason: observation.Reason, UnknownClass: class, NextOperation: "RECORD_BOTH_SIDES_WITH_THE_SAME_IDENTITY", BlockedBy: []string{metric.name}}
		} else {
			delta := *metric.after - *metric.before
			observation.Delta = &delta
			observation.SignedDelta = &delta
			observation.Improvement = &delta
			observation.State = V3IndicatorObserved
			observation.Reason = "EXACT_TEST_CONFORMANCE_INPUT_TOOLCHAIN_SEMANTIC_IR_PAIR_WITH_PRIOR_RUN_PROVENANCE"
		}
		result = append(result, observation)
	}
	return result
}

func summarizeV3(plans []V3ActivityPlan) V3DossierSummary {
	summary := V3DossierSummary{TotalActivities: len(plans)}
	for _, plan := range plans {
		if plan.ReusableEvidence {
			summary.ReusableEvidence++
		}
		if plan.RequiredRun {
			summary.RequiredRuns++
		}
		if plan.Unknown {
			summary.Unknown++
		}
		if plan.Refuted {
			summary.Refuted++
		}
		if plan.Executed {
			summary.Executed++
		}
		if plan.SkippedWithProof {
			summary.SkippedWithProof++
		}
	}
	return summary
}

func evaluateV3FixedPoint(fixture V3Fixture) string {
	if fixture.FixedPoint == nil {
		return V3FixedPointNone
	}
	if !fixture.FixedPoint.Declared || fixture.FixedPoint.Kind != V3FixedPointExplicit || fixture.FixedPoint.Witness == "" {
		return V3FixedPointRefuted
	}
	return V3FixedPointExplicit
}

func computeV3Impact(before, after V3ProofGraph, declared V3ChangeSet) ([]string, []string, []string) {
	beforeNodes, afterNodes := map[string]V3GraphNode{}, map[string]V3GraphNode{}
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
	beforeEdges, afterEdges := v3EdgeSet(before.Edges), v3EdgeSet(after.Edges)
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
	changedIDs := sortedMapKeys(changed)
	reasons := []string{}
	if declared.BeforeDigest == "" || declared.AfterDigest == "" || (len(declared.ChangedNodes) == 0 && len(changedIDs) > 0) || !sameStringSet(declared.ChangedNodes, changedIDs) {
		reasons = append(reasons, "SEMANTIC_CHANGE_SET_CONTRADICTION")
	}
	for _, edge := range after.Edges {
		from, fromOK := afterNodes[edge.From]
		to, toOK := afterNodes[edge.To]
		if !fromOK || !toOK || (edge.FromDigest != "" && edge.FromDigest != from.Digest) || (edge.ToDigest != "" && edge.ToDigest != to.Digest) {
			reasons = append(reasons, V3RefutedGraph)
		}
	}
	impacted := map[string]bool{}
	for _, id := range changedIDs {
		impacted[id] = true
	}
	queue := append([]string(nil), changedIDs...)
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
	return sortedMapKeys(impacted), sortedMapKeys(closureEdges), uniqueStrings(reasons)
}

func v3EdgeSet(edges []V3GraphEdge) map[string]V3GraphEdge {
	result := map[string]V3GraphEdge{}
	for _, edge := range edges {
		result[edge.From+"->"+edge.To+"#"+edge.Relation] = edge
	}
	return result
}

func sortedV3Cases(cases []V3CaseSpec) []V3CaseSpec {
	result := append([]V3CaseSpec(nil), cases...)
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func writeV3Report(outputDir string, report V3Report) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outputDir, "report.json"), report); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outputDir, "activity-vector.json"), report.Activities); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outputDir, "indicator-vector.json"), report.Indicators); err != nil {
		return err
	}
	return writeText(filepath.Join(outputDir, "human-report.md"), RenderV3Report(report))
}

