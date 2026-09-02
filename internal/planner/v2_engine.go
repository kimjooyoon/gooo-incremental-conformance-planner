package planner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type V2SuiteOptions struct {
	MetaPath          string
	ContractPath      string
	CasesRoot         string
	OutputDir         string
	ActionsReceiptPath string
}

func RunV2Suite(options V2SuiteOptions) (V2SuiteReport, error) {
	if !filepath.IsAbs(options.OutputDir) {
		return V2SuiteReport{}, errors.New("v2 suite output directory must be an absolute caller-owned path")
	}
	source, sourceDigest, err := ParseV2Source(options.MetaPath)
	if err != nil {
		return V2SuiteReport{}, fmt.Errorf("load v2 .gooo: %w", err)
	}
	ir, err := BuildV2SemanticIR(source, options.MetaPath, sourceDigest)
	if err != nil {
		return V2SuiteReport{}, fmt.Errorf("build semantic IR: %w", err)
	}
	contract, contractDigest, err := LoadV2Contract(options.ContractPath)
	if err != nil {
		return V2SuiteReport{}, err
	}
	if !sameV2Activities(source.Activities, contract.Activities) {
		return V2SuiteReport{}, errors.New("v2 contract activities do not match the authoritative .gooo source")
	}
	var actions *V2ActionsReceipt
	if options.ActionsReceiptPath != "" {
		receipt, err := LoadV2ActionsReceipt(options.ActionsReceiptPath)
		if err != nil {
			return V2SuiteReport{}, err
		}
		actions = &receipt
	}

	if err := os.MkdirAll(options.OutputDir, 0o755); err != nil {
		return V2SuiteReport{}, err
	}
	if err := writeJSON(filepath.Join(options.OutputDir, "semantic-ir.json"), ir); err != nil {
		return V2SuiteReport{}, err
	}
	evaluator := V2EvaluatorArtifact{Schema: "gooo/incremental-conformance-planner/evaluator/v2", SemanticIRDigest: ir.Digest, Digest: ir.EvaluatorDigest, Activities: ir.Activities}
	if err := writeJSON(filepath.Join(options.OutputDir, "generated", "evaluator.json"), evaluator); err != nil {
		return V2SuiteReport{}, err
	}
	if actions != nil {
		if err := writeJSON(filepath.Join(options.OutputDir, "actions-receipt.json"), actions); err != nil {
			return V2SuiteReport{}, err
		}
	}

	caseResults := make([]V2CaseResult, 0, len(contract.Scenarios))
	for _, spec := range sortedV2Cases(contract.Scenarios) {
		fixture, err := LoadV2Fixture(pathFrom(options.CasesRoot, spec.Source))
		if err != nil {
			return V2SuiteReport{}, fmt.Errorf("case %s: %w", spec.ID, err)
		}
		projection, err := ProjectV2(ir, contract, fixture)
		if err != nil {
			return V2SuiteReport{}, fmt.Errorf("project case %s: %w", spec.ID, err)
		}
		report := EvaluateV2(ir, contract, fixture, projection, contractDigest, actions)
		caseDir := filepath.Join(options.OutputDir, "cases", spec.ID)
		if err := writeV2Report(caseDir, report); err != nil {
			return V2SuiteReport{}, err
		}
		match := report.Decision == spec.Expected && report.Decision == fixture.Expected.Decision
		if match {
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
		}
		caseResults = append(caseResults, V2CaseResult{ID: spec.ID, Expected: spec.Expected, Actual: report.Decision, Match: match, Report: filepath.ToSlash(filepath.Join("cases", spec.ID, "human-report.md"))})
	}

	decision := V2DecisionClosed
	for _, result := range caseResults {
		if !result.Match {
			decision = V2DecisionRefuted
			break
		}
	}
	actionsMetricState := "OBSERVED"
	missingActionsMetrics := []string{}
	if actions != nil {
		missingActionsMetrics = missingV2ActionsMetrics(*actions)
		if len(missingActionsMetrics) > 0 {
			actionsMetricState = V2DecisionUnknown
			decision = V2DecisionUnknown
		}
	}
	suite := V2SuiteReport{
		Schema: "gooo/incremental-conformance-planner/v2-suite-report/v1", Decision: decision,
		Contract: contract.ID, ContractDigest: contractDigest, TotalActivities: contract.TargetActivities,
		Cases: caseResults, ActionsReceipt: actions, ActionsMetricState: actionsMetricState, MissingActionsMetrics: missingActionsMetrics,
		Operational: V2Operational{RepositoryWrites: 0, LocalTestExecutions: 0, CrossProjectRequiredGates: 0, FailedRunsPreserved: true, OutputLocation: options.OutputDir, VerificationAuthority: "GitHub Actions"},
	}
	if err := writeJSON(filepath.Join(options.OutputDir, "suite-report.json"), suite); err != nil {
		return V2SuiteReport{}, err
	}
	if err := writeText(filepath.Join(options.OutputDir, "human-report.md"), RenderV2SuiteReport(suite)); err != nil {
		return V2SuiteReport{}, err
	}
	return suite, nil
}

func missingV2ActionsMetrics(receipt V2ActionsReceipt) []string {
	missing := []string{}
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
	if receipt.CacheHits == nil {
		missing = append(missing, "cache_hits")
	}
	if receipt.CacheMisses == nil {
		missing = append(missing, "cache_misses")
	}
	for _, activity := range receipt.Activities {
		if activity.ActivityID == "BUILD_REPOSITORY" && activity.DurationMS == nil {
			missing = append(missing, "BUILD_REPOSITORY.duration_ms")
		}
		if activity.ActivityID == "TEST_REPOSITORY" && activity.DurationMS == nil {
			missing = append(missing, "TEST_REPOSITORY.duration_ms")
		}
	}
	return uniqueStrings(missing)
}

func sameV2Activities(source, contract []V2ActivityDescriptor) bool {
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

type V2EvaluatorArtifact struct {
	Schema          string                 `json:"schema"`
	SemanticIRDigest string                `json:"semantic_ir_digest"`
	Digest          string                 `json:"digest"`
	Activities      []V2ActivityDescriptor `json:"activities"`
}

func ProjectV2(ir V2SemanticIR, contract V2Contract, fixture V2Fixture) (V2Projection, error) {
	if fixture.Kind != "NORMAL" && fixture.Kind != V2DecisionUnknown && fixture.Kind != V2DecisionRefuted {
		return V2Projection{}, fmt.Errorf("fixture %s has invalid kind %q", fixture.CaseID, fixture.Kind)
	}
	if fixture.Expected.Decision == "" {
		return V2Projection{}, fmt.Errorf("fixture %s has no expected decision", fixture.CaseID)
	}
	impactNodes, impactEdges, graphReasons := computeV2Impact(fixture.Before, fixture.After, fixture.SemanticChange)
	inputByID := map[string]V2ActivityInput{}
	for _, input := range fixture.Activities {
		if input.ActivityID == "" || inputByID[input.ActivityID].ActivityID != "" {
			return V2Projection{}, fmt.Errorf("fixture %s has a duplicate or empty activity input", fixture.CaseID)
		}
		inputByID[input.ActivityID] = input
	}
	projected := make([]V2ProjectedActivity, 0, len(contract.Activities))
	for _, descriptor := range contract.Activities {
		input, ok := inputByID[descriptor.ID]
		if !ok {
			return V2Projection{}, fmt.Errorf("fixture %s is missing activity %s", fixture.CaseID, descriptor.ID)
		}
		if input.CurrentIdentity == nil {
			identity := fixture.IdentityDefaults
			input.CurrentIdentity = &identity
		}
		if input.PriorReceipt == nil && fixture.ReceiptDefaults != nil {
			receipt := *fixture.ReceiptDefaults
			input.PriorReceipt = &receipt
		}
		if input.Observation == nil {
			observation := fixture.ObservationDefaults
			input.Observation = &observation
		}
		projected = append(projected, V2ProjectedActivity{Descriptor: descriptor, Input: input, Identity: *input.CurrentIdentity, Receipt: input.PriorReceipt, Observation: *input.Observation})
	}
	if len(inputByID) != len(contract.Activities) {
		return V2Projection{}, fmt.Errorf("fixture %s has activity inputs outside the v2 contract", fixture.CaseID)
	}
	return V2Projection{CaseID: fixture.CaseID, Change: fixture.SemanticChange, ImpactNodes: impactNodes, ImpactEdges: impactEdges, GraphReasons: graphReasons, Activities: projected}, nil
}

func EvaluateV2(ir V2SemanticIR, contract V2Contract, fixture V2Fixture, projection V2Projection, contractDigest string, actions *V2ActionsReceipt) V2Report {
	plans := make([]V2ActivityPlan, 0, len(projection.Activities))
	unknownReasons := []string{}
	refutedReasons := append([]string(nil), projection.GraphReasons...)
	fixtureDigest := DigestBytes([]byte(fixture.FixtureAnchor))
	for _, activity := range projection.Activities {
		plan := evaluateV2Activity(ir, contractDigest, fixtureDigest, fixture.Indicators.ScenarioDigest, projection, activity)
		plans = append(plans, plan)
		if plan.Unknown {
			unknownReasons = append(unknownReasons, plan.ActivityID+":"+plan.Reason)
		}
		if plan.Refuted {
			refutedReasons = append(refutedReasons, plan.ActivityID+":"+plan.Reason)
		}
	}
	indicators := evaluateV2Indicators(fixture.Indicators)
	for _, indicator := range indicators {
		if indicator.State == IndicatorRefuted {
			refutedReasons = append(refutedReasons, indicator.Metric+":"+indicator.Reason)
		}
		if indicator.State == IndicatorUnknown {
			unknownReasons = append(unknownReasons, indicator.Metric+":"+indicator.Reason)
		}
	}
	decision := V2DecisionClosed
	if len(refutedReasons) > 0 {
		decision = V2DecisionRefuted
	} else if len(unknownReasons) > 0 {
		decision = V2DecisionUnknown
	}
	summary := summarizeV2(plans)
	return V2Report{
		Schema: "gooo/incremental-conformance-planner/v2-report/v1", CaseID: fixture.CaseID, Decision: decision,
		Precedence: []string{V2DecisionRefuted, V2DecisionUnknown, V2DecisionClosed}, SemanticIR: ir,
		SemanticChange: fixture.SemanticChange, ImpactNodes: projection.ImpactNodes, ImpactEdges: projection.ImpactEdges,
		Activities: plans, Summary: summary, Indicators: indicators, UnknownReasons: unknownReasons, RefutedReasons: refutedReasons,
		ContractDigest: contractDigest, FixtureDigest: fixtureDigest, EvaluatorDigest: ir.EvaluatorDigest, ActionsReceipt: actions,
		Operational: V2Operational{RepositoryWrites: 0, LocalTestExecutions: 0, CrossProjectRequiredGates: 0, FailedRunsPreserved: true, OutputLocation: "caller-owned", VerificationAuthority: "GitHub Actions"},
	}
}

func evaluateV2Activity(ir V2SemanticIR, contractDigest, fixtureDigest, scenarioDigest string, projection V2Projection, activity V2ProjectedActivity) V2ActivityPlan {
	plan := V2ActivityPlan{ActivityID: activity.Descriptor.ID, Activity: activity.Descriptor.Activity, ProofChoice: activity.Descriptor.ProofChoice, IndicatorClass: activity.Descriptor.IndicatorClass, DurationMS: activity.Observation.DurationMS, CacheHit: activity.Observation.CacheHit, CacheMiss: activity.Observation.CacheMiss, PriorState: priorState(activity.Receipt)}
	if len(projection.GraphReasons) > 0 {
		return v2RefutedPlan(plan, V2RefutedGraph, "RELEASE_CORRECTED_SEMANTIC_OR_PROOF_GRAPH")
	}
	identity := activity.Identity
	if missing := identity.Digests.Missing(); len(missing) > 0 {
		plan.MissingFields = missing
		return v2UnknownPlan(plan, V2UnknownMissingIdentity, "COLLECT_THE_SIX_REQUIRED_IDENTITY_DIGESTS")
	}
	if missing := identity.Activity.Missing(); len(missing) > 0 {
		plan.MissingFields = missing
		return v2UnknownPlan(plan, V2UnknownMissingProvenance, "DECLARE_BUILD_AND_TEST_ACTIVITY_PROVENANCE")
	}
	if identity.Digests.SourceDigest != ir.SourceDigest || identity.Digests.SemanticIRDigest != ir.Digest || identity.Digests.FixtureDigest != fixtureDigest || identity.Digests.ContractDigest != contractDigest || identity.Digests.EvaluatorDigest != ir.EvaluatorDigest || identity.Digests.GoToolchainDigest != ir.ToolchainDigest || identity.Activity.ScenarioDigest != scenarioDigest {
		return v2RefutedPlan(plan, "IDENTITY_AUTHORITY_MISMATCH", "RELEASE_CURRENT_GOOO_IR_CONTRACT_AND_EVALUATOR_BINDINGS")
	}
	receipt := activity.Receipt
	if receipt == nil {
		return v2UnknownPlan(plan, V2UnknownMissingReceipt, "EXECUTE_AND_EMIT_AN_IMMUTABLE_RECEIPT")
	}
	if receipt.ProofSource == "EVALUATOR_SELF_APPROVAL" {
		return v2RefutedPlan(plan, V2RefutedEvaluatorSelfApproval, "OBTAIN_INDEPENDENT_CI_PROOF")
	}
	if receipt.ReceiptDigest != "" && receipt.ContentDigest != "" && receipt.ReceiptDigest != receipt.ContentDigest {
		return v2RefutedPlan(plan, V2RefutedForgedReceipt, "PRESERVE_THE_FORGED_RECEIPT_AND_RUN_CURRENT_ACTIVITY")
	}
	if missing := receipt.Identity.Missing(); len(missing) > 0 {
		plan.MissingFields = missing
		return v2UnknownPlan(plan, V2UnknownMissingProvenance, "RESTORE_COMPLETE_PRIOR_RECEIPT_PROVENANCE")
	}
	if receipt.ReceiptDigest == "" || receipt.ContentDigest == "" || receipt.SourceRunID == "" || receipt.SourceCommit == "" {
		return v2UnknownPlan(plan, V2UnknownMissingReceipt, "RESTORE_SIGNED_IMMUTABLE_RECEIPT_PROVENANCE")
	}
	if receipt.ReceiptDigest != receipt.ContentDigest || !receipt.Immutable {
		return v2RefutedPlan(plan, V2RefutedForgedReceipt, "PRESERVE_THE_FORGED_RECEIPT_AND_RUN_CURRENT_ACTIVITY")
	}
	if receipt.State == ProofFailed || receipt.State == ProofCounterexample || receipt.State != ProofPass || receipt.ResultDigest == "" {
		return v2RefutedPlan(plan, "PRIOR_PROOF_REFUTED", "PRESERVE_PRIOR_FAILURE_AND_EXECUTE_CURRENT_ACTIVITY")
	}
	plan.MismatchedFields = v2IdentityMismatches(identity, receipt.Identity)
	impacted := intersects(activity.Input.SemanticNodes, projection.ImpactNodes)
	if impacted && activity.Observation.Status == "SKIPPED_WITH_PROOF" {
		return v2RefutedPlan(plan, V2RefutedAffectedSkip, "EXECUTE_EVERY_IMPACTED_BUILD_AND_TEST_ACTIVITY")
	}
	if impacted {
		plan.Action = V2ActionRequiredRun
		plan.RequiredRun = true
		plan.Reason = "SEMANTIC_IMPACT_REQUIRES_CURRENT_REVERIFICATION"
		plan.NextOperation = "EXECUTE_AND_RECORD_BUILD_TEST_ACTIVITY"
		return requireExecutedV2Plan(plan, activity.Observation)
	}
	if len(plan.MismatchedFields) > 0 {
		plan.Action = V2ActionRequiredRun
		plan.RequiredRun = true
		plan.Reason = "IDENTITY_CHANGED_REQUIRES_CURRENT_REVERIFICATION"
		plan.NextOperation = "EXECUTE_WITH_CURRENT_IDENTITY"
		return requireExecutedV2Plan(plan, activity.Observation)
	}
	if activity.Observation.Status == "SKIPPED_WITH_PROOF" {
		plan.Action = V2ActionReusedClosed
		plan.AlreadyVerified = true
		plan.ReusedClosed = true
		plan.SkippedWithProof = true
		plan.Reason = "EXACT_IDENTITY_AND_INDEPENDENT_IMMUTABLE_PASS_RECEIPT"
		plan.NextOperation = "NONE"
		return plan
	}
	if activity.Observation.CacheHit != nil && *activity.Observation.CacheHit {
		return v2UnknownPlan(plan, V2UnknownMissingReceipt, "CACHE_HIT_IS_A_FACT_NOT_A_REUSE_PROOF")
	}
	return v2UnknownPlan(plan, V2UnknownMissingMetrics, "REUSE_PROOF_REQUIRES_A_CURRENT_ACTIONS_OBSERVATION")
}

func requireExecutedV2Plan(plan V2ActivityPlan, observation V2ActivityObservation) V2ActivityPlan {
	if observation.Status == "SKIPPED_WITH_PROOF" {
		return v2RefutedPlan(plan, V2RefutedAffectedSkip, "EXECUTE_EVERY_REQUIRED_ACTIVITY")
	}
	if observation.Status != "EXECUTED" {
		return v2UnknownPlan(plan, V2UnknownMissingMetrics, "EXECUTE_REQUIRED_ACTIVITY_AND_RECORD_ACTIONS_RECEIPT")
	}
	plan.Executed = true
	return plan
}

func v2UnknownPlan(plan V2ActivityPlan, class, next string) V2ActivityPlan {
	plan.Action = V2ActionUnknown
	plan.Unknown = true
	plan.Reason = class
	plan.NextOperation = next
	return plan
}

func v2RefutedPlan(plan V2ActivityPlan, reason, next string) V2ActivityPlan {
	plan.Action = V2ActionRefuted
	plan.Refuted = true
	plan.Reason = reason
	plan.NextOperation = next
	return plan
}

func priorState(receipt *V2PriorReceipt) string {
	if receipt == nil {
		return ""
	}
	return receipt.State
}

func v2IdentityMismatches(current, prior V2IdentityBundle) []string {
	result := []string{}
	if current.Digests.SourceDigest != prior.Digests.SourceDigest { result = append(result, "source_digest") }
	if current.Digests.SemanticIRDigest != prior.Digests.SemanticIRDigest { result = append(result, "semantic_ir_digest") }
	if current.Digests.FixtureDigest != prior.Digests.FixtureDigest { result = append(result, "fixture_digest") }
	if current.Digests.ContractDigest != prior.Digests.ContractDigest { result = append(result, "contract_digest") }
	if current.Digests.GoToolchainDigest != prior.Digests.GoToolchainDigest { result = append(result, "go_toolchain_digest") }
	if !current.Activity.Equal(prior.Activity) { result = append(result, "activity_identity") }
	return result
}

func summarizeV2(plans []V2ActivityPlan) V2DossierSummary {
	summary := V2DossierSummary{TotalActivities: len(plans)}
	for _, plan := range plans {
		switch plan.Action {
		case V2ActionRequiredRun:
			summary.RequiredRuns++
		case V2ActionReusedClosed:
			summary.ReusedClosed++
		case V2ActionUnknown:
			summary.Unknown++
		case V2ActionRefuted:
			summary.Refuted++
		}
		if plan.Executed { summary.Executed++ }
		if plan.SkippedWithProof { summary.SkippedWithProof++ }
	}
	return summary
}

func computeV2Impact(before, after V2ProofGraph, declared V2ChangeSet) ([]string, []string, []string) {
	beforeNodes, afterNodes := map[string]V2GraphNode{}, map[string]V2GraphNode{}
	for _, node := range before.Nodes { beforeNodes[node.ID] = node }
	for _, node := range after.Nodes { afterNodes[node.ID] = node }
	changed := map[string]bool{}
	for id, oldNode := range beforeNodes {
		newNode, ok := afterNodes[id]
		if !ok || oldNode.Digest != newNode.Digest { changed[id] = true }
	}
	for id := range afterNodes { if _, ok := beforeNodes[id]; !ok { changed[id] = true } }
	beforeEdges, afterEdges := v2EdgeSet(before.Edges), v2EdgeSet(after.Edges)
	for key, edge := range beforeEdges { if _, ok := afterEdges[key]; !ok { changed[edge.From] = true; changed[edge.To] = true } }
	for key, edge := range afterEdges { if _, ok := beforeEdges[key]; !ok { changed[edge.From] = true; changed[edge.To] = true } }
	changedIDs := sortedMapKeys(changed)
	reasons := []string{}
	if declared.BeforeDigest == "" || declared.AfterDigest == "" || len(declared.ChangedNodes) == 0 && len(changedIDs) > 0 || !sameStringSet(declared.ChangedNodes, changedIDs) {
		reasons = append(reasons, "SEMANTIC_CHANGE_SET_CONTRADICTION")
	}
	for _, edge := range after.Edges {
		from, fromOK := afterNodes[edge.From]
		to, toOK := afterNodes[edge.To]
		if !fromOK || !toOK || (edge.FromDigest != "" && edge.FromDigest != from.Digest) || (edge.ToDigest != "" && edge.ToDigest != to.Digest) {
			reasons = append(reasons, V2RefutedGraph)
		}
	}
	impacted := map[string]bool{}
	for _, id := range changedIDs { impacted[id] = true }
	queue := append([]string(nil), changedIDs...)
	closureEdges := map[string]bool{}
	for len(queue) > 0 {
		current := queue[0]; queue = queue[1:]
		for _, edge := range after.Edges {
			if edge.From != current { continue }
			closureEdges[edge.From+"->"+edge.To] = true
			if !impacted[edge.To] { impacted[edge.To] = true; queue = append(queue, edge.To) }
		}
	}
	return sortedMapKeys(impacted), sortedMapKeys(closureEdges), uniqueStrings(reasons)
}

func v2EdgeSet(edges []V2GraphEdge) map[string]V2GraphEdge {
	result := map[string]V2GraphEdge{}
	for _, edge := range edges { result[edge.From+"->"+edge.To+"#"+edge.Relation] = edge }
	return result
}

func sameStringSet(left, right []string) bool {
	left = sortedStrings(left); right = sortedStrings(right)
	if len(left) != len(right) { return false }
	for index := range left { if left[index] != right[index] { return false } }
	return true
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values { if !seen[value] { seen[value] = true; result = append(result, value) } }
	sort.Strings(result)
	return result
}

func evaluateV2Indicators(pair V2MetricPair) []V2IndicatorObservation {
	identityPair := pair.BeforeKnown && pair.AfterKnown && pair.ScenarioDigest != "" && len(pair.BeforeIdentity.Missing()) == 0 && len(pair.AfterIdentity.Missing()) == 0 && pair.BeforeIdentity.Activity.ScenarioDigest == pair.ScenarioDigest && pair.AfterIdentity.Activity.ScenarioDigest == pair.ScenarioDigest && pair.BeforeIdentity.Equal(pair.AfterIdentity)
	metrics := []struct { name string; before, after *int64 }{
		{name: "build_ms", before: pair.Before.BuildMS, after: pair.After.BuildMS},
		{name: "test_ms", before: pair.Before.TestMS, after: pair.After.TestMS},
		{name: "wall_ms", before: pair.Before.WallMS, after: pair.After.WallMS},
		{name: "peak_rss_kib", before: pair.Before.PeakRSSKiB, after: pair.After.PeakRSSKiB},
	}
	result := make([]V2IndicatorObservation, 0, len(metrics))
	for _, metric := range metrics {
		observation := V2IndicatorObservation{Metric: metric.name, Before: metric.before, After: metric.after, SameIdentity: identityPair}
		if pair.RunState == ProofFailed || pair.RunState == ProofCounterexample {
			observation.State = IndicatorRefuted
			observation.Reason = "FAILED_ACTIONS_RECEIPT_IS_OPERATIONAL_REFUTATION"
		} else if !identityPair || metric.before == nil || metric.after == nil {
			observation.State = IndicatorUnknown
			observation.Reason = "EXACT_SCENARIO_AND_IDENTITY_PAIR_REQUIRED"
		} else {
			delta := *metric.after - *metric.before
			observation.SignedDelta = &delta
			observation.Improvement = &delta
			observation.State = IndicatorObserved
			observation.Reason = "EXACT_SAME_SCENARIO_SOURCE_IR_FIXTURE_CONTRACT_EVALUATOR_TOOLCHAIN_PAIR"
		}
		result = append(result, observation)
	}
	return result
}

func sortedV2Cases(cases []V2CaseSpec) []V2CaseSpec {
	result := append([]V2CaseSpec(nil), cases...)
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func writeV2Report(outputDir string, report V2Report) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil { return err }
	if err := writeJSON(filepath.Join(outputDir, "report.json"), report); err != nil { return err }
	if err := writeJSON(filepath.Join(outputDir, "activity-vector.json"), report.Activities); err != nil { return err }
	if err := writeJSON(filepath.Join(outputDir, "indicator-vector.json"), report.Indicators); err != nil { return err }
	return writeText(filepath.Join(outputDir, "human-report.md"), RenderV2Report(report))
}
