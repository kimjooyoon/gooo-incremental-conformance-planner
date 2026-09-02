package planner

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ParseV3Source(path string) (V3Source, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return V3Source{}, "", err
	}
	source := V3Source{
		Precedence: []string{}, Authorities: []string{}, IdentityFields: []string{}, PriorRunFields: []string{},
		ObservationFields: []string{}, UnknownFields: []string{}, UnknownClasses: []string{}, Policies: []string{},
		ProofChoices: []string{}, IndicatorClasses: []string{}, FixedPointRules: []string{}, FixedPointCases: []string{}, Activities: []V2ActivityDescriptor{},
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return V3Source{}, "", fmt.Errorf("line %d: declaration has no value", lineNumber)
		}
		switch fields[0] {
		case "schema":
			source.Schema = fields[1]
		case "program":
			source.Program = fields[1]
		case "namespace":
			source.Namespace = fields[1]
		case "precedence":
			source.Precedence = strings.Fields(strings.ReplaceAll(strings.Join(fields[1:], " "), ">", " "))
		case "authority":
			source.Authorities = append(source.Authorities, fields[1])
		case "identity_fields":
			source.IdentityFields = strings.Fields(strings.Join(fields[1:], " "))
		case "prior_run_fields":
			source.PriorRunFields = strings.Fields(strings.Join(fields[1:], " "))
		case "observation_fields":
			source.ObservationFields = strings.Fields(strings.Join(fields[1:], " "))
		case "unknown_fields":
			source.UnknownFields = strings.Fields(strings.Join(fields[1:], " "))
		case "unknown_class":
			if len(fields) < 3 || fields[1] != "enum" {
				return V3Source{}, "", fmt.Errorf("line %d: unknown_class requires enum values", lineNumber)
			}
			source.UnknownClasses = append(source.UnknownClasses, fields[2:]...)
		case "policy":
			source.Policies = append(source.Policies, fields[1])
		case "toolchain_digest":
			source.ToolchainDigest = fields[1]
		case "proof_choice":
			source.ProofChoices = append(source.ProofChoices, fields[1])
		case "indicator_class":
			source.IndicatorClasses = append(source.IndicatorClasses, fields[1])
		case "fixed_point_rule":
			source.FixedPointRules = append(source.FixedPointRules, fields[1])
		case "fixed_point_case":
			if len(fields) < 3 {
				return V3Source{}, "", fmt.Errorf("line %d: fixed_point_case requires id and mode", lineNumber)
			}
			source.FixedPointCases = append(source.FixedPointCases, fields[1]+"="+fields[2])
		case "semantic_change_set", "proof_graph", "test_identity", "conformance_identity", "input_digest", "activity_identity", "prior_run_identity", "observation", "semantic_ir_digest", "semantic_ir", "evaluator", "selection_policy":
			// These declarations name authority-owned structures and policies.
		case "activity":
			if len(fields) < 9 {
				return V3Source{}, "", fmt.Errorf("line %d: activity requires ordinal, id, stage, step, proof, indicator, and dependencies", lineNumber)
			}
			ordinal, err := strconv.Atoi(fields[1])
			if err != nil {
				return V3Source{}, "", fmt.Errorf("line %d: activity ordinal: %w", lineNumber, err)
			}
			dependencies := []string{}
			if fields[8] != "-" {
				dependencies = strings.Split(fields[8], ",")
			}
			source.Activities = append(source.Activities, V2ActivityDescriptor{
				Ordinal: ordinal, ID: fields[2], Activity: fields[3], Stage: fields[4], Step: fields[5],
				ProofChoice: fields[6], IndicatorClass: fields[7], DependsOn: dependencies,
			})
		default:
			return V3Source{}, "", fmt.Errorf("line %d: unknown v3 .gooo declaration %q", lineNumber, fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return V3Source{}, "", err
	}
	if err := validateV3Source(source); err != nil {
		return V3Source{}, "", err
	}
	return source, DigestBytes(data), nil
}

func validateV3Source(source V3Source) error {
	if source.Schema != "gooo/incremental-conformance-planner/meta/v3" || source.Program == "" || source.Namespace == "" {
		return errors.New("v3 .gooo must define the planner schema, program, and namespace")
	}
	if strings.Join(source.Precedence, ">") != "REFUTED>UNKNOWN>CLOSED" {
		return errors.New("v3 .gooo precedence must be REFUTED > UNKNOWN > CLOSED")
	}
	for _, required := range []string{"SemanticChangeSet", "ProofGraph", "TestIdentity", "ConformanceIdentity", "InputDigest", "ToolchainDigest", "SemanticIRDigest", "PriorRunIdentity", "Observation", "EvidenceSelection", "ImprovementPair", "Dossier"} {
		if !contains(source.Authorities, required) {
			return fmt.Errorf("v3 .gooo is missing authority %q", required)
		}
	}
	if strings.Join(source.IdentityFields, ",") != strings.Join(V3IdentityFields, ",") {
		return errors.New("v3 .gooo must declare the exact verification identity fields")
	}
	if strings.Join(source.PriorRunFields, ",") != strings.Join(V3PriorRunFields, ",") {
		return errors.New("v3 .gooo must declare the exact prior run identity fields")
	}
	if strings.Join(source.ObservationFields, ",") != strings.Join(V3ObservationFields, ",") {
		return errors.New("v3 .gooo must declare build/test/wall/RSS and cache observation fields")
	}
	if strings.Join(source.UnknownFields, ",") != strings.Join(V3UnknownFields, ",") {
		return errors.New("v3 .gooo must declare the fixed six-field UNKNOWN record")
	}
	for _, required := range V3Policies {
		if !contains(source.Policies, required) {
			return fmt.Errorf("v3 .gooo is missing policy %q", required)
		}
	}
	if len(source.UnknownClasses) != 5 || !contains(source.UnknownClasses, V3UnknownMissingIdentity) || !contains(source.UnknownClasses, V3UnknownMissingProvenance) || !contains(source.UnknownClasses, V3UnknownMissingReceipt) || !contains(source.UnknownClasses, V3UnknownMissingMetrics) || !contains(source.UnknownClasses, V3UnknownMissingPriorRun) {
		return errors.New("v3 .gooo must declare the five UNKNOWN classes")
	}
	if source.ToolchainDigest == "" {
		return errors.New("v3 .gooo must declare the Go toolchain digest")
	}
	if len(source.ProofChoices) != 3 || !contains(source.ProofChoices, V3ProofFoundation) || !contains(source.ProofChoices, V3ProofCoherence) || !contains(source.ProofChoices, V3ProofRegression) {
		return errors.New("v3 .gooo must declare FOUNDATION, COHERENCE, and REGRESSION")
	}
	if len(source.IndicatorClasses) != 3 || !contains(source.IndicatorClasses, V3IndicatorDriver) || !contains(source.IndicatorClasses, V3IndicatorOutcome) || !contains(source.IndicatorClasses, V3IndicatorGuardrail) {
		return errors.New("v3 .gooo must declare DRIVER, OUTCOME, and GUARDRAIL")
	}
	if !contains(source.FixedPointRules, "FIXED_POINT_ONLY_IF_EXPLICIT") || !contains(source.FixedPointRules, "UNKNOWN_TOP_DECISION_FAIL_CLOSED") || !contains(source.FixedPointRules, "MALFORMED_OR_IMPLICIT_FIXED_POINT_REFUTED") {
		return errors.New("v3 .gooo must declare explicit fixed-point rules")
	}
	if len(source.FixedPointCases) == 0 {
		return errors.New("v3 .gooo must declare at least one explicit fixed-point case")
	}
	if len(source.Activities) != 12 {
		return errors.New("v3 .gooo must declare exactly 12 activities")
	}
	seen := map[string]bool{}
	seenOrdinals := map[int]bool{}
	proofCounts := map[string]int{}
	indicatorCounts := map[string]int{}
	for _, activity := range source.Activities {
		if activity.Ordinal < 1 || activity.Ordinal > 12 || activity.ID == "" || activity.Activity == "" || activity.Stage == "" || activity.Step == "" || seen[activity.ID] || seenOrdinals[activity.Ordinal] {
			return fmt.Errorf("v3 .gooo has an invalid or duplicate activity %q", activity.ID)
		}
		seen[activity.ID] = true
		seenOrdinals[activity.Ordinal] = true
		proofCounts[activity.ProofChoice]++
		indicatorCounts[activity.IndicatorClass]++
	}
	for _, choice := range []string{V3ProofFoundation, V3ProofCoherence, V3ProofRegression} {
		if proofCounts[choice] != 4 {
			return fmt.Errorf("v3 .gooo proof choice %s must have four activities", choice)
		}
	}
	for _, class := range []string{V3IndicatorDriver, V3IndicatorOutcome, V3IndicatorGuardrail} {
		if indicatorCounts[class] != 4 {
			return fmt.Errorf("v3 .gooo indicator class %s must have four activities", class)
		}
	}
	return nil
}

func BuildV3SemanticIR(source V3Source, sourcePath, sourceDigest string) (V3SemanticIR, error) {
	ir := V3SemanticIR{
		Schema: "gooo/incremental-conformance-planner/semantic-ir/v3", Program: source.Program, SourcePath: sourcePath,
		SourceDigest: sourceDigest, ToolchainDigest: source.ToolchainDigest, IdentityFields: append([]string(nil), source.IdentityFields...),
		PriorRunFields: append([]string(nil), source.PriorRunFields...), ObservationFields: append([]string(nil), source.ObservationFields...),
		UnknownFields: append([]string(nil), source.UnknownFields...), Policies: append([]string(nil), source.Policies...), Activities: append([]V2ActivityDescriptor(nil), source.Activities...),
	}
	canonical := struct {
		Schema            string                 `json:"schema"`
		Program           string                 `json:"program"`
		SourceDigest      string                 `json:"source_digest"`
		ToolchainDigest   string                 `json:"toolchain_digest"`
		IdentityFields    []string               `json:"identity_fields"`
		PriorRunFields    []string               `json:"prior_run_fields"`
		ObservationFields []string               `json:"observation_fields"`
		UnknownFields     []string               `json:"unknown_fields"`
		Policies          []string               `json:"policies"`
		Activities        []V2ActivityDescriptor `json:"activities"`
	}{ir.Schema, ir.Program, ir.SourceDigest, ir.ToolchainDigest, ir.IdentityFields, ir.PriorRunFields, ir.ObservationFields, ir.UnknownFields, ir.Policies, ir.Activities}
	digest, err := DigestJSON(canonical)
	if err != nil {
		return V3SemanticIR{}, err
	}
	ir.Digest = digest
	ir.EvaluatorDigest = DigestBytes([]byte("gooo-incremental-conformance-planner/v3/evaluator/" + digest))
	return ir, nil
}

func LoadV3Contract(path string) (V3Contract, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return V3Contract{}, "", err
	}
	var contract V3Contract
	if err := readJSON(path, &contract); err != nil {
		return V3Contract{}, "", err
	}
	if err := validateV3Contract(contract); err != nil {
		return V3Contract{}, "", err
	}
	return contract, DigestBytes(data), nil
}

func validateV3Contract(contract V3Contract) error {
	if contract.Schema != "gooo/incremental-conformance-planner/denominator/v3" || contract.Version != "v3" || contract.AppendOnlyFrom != "v2" || contract.TargetActivities != 12 || len(contract.Activities) != 12 {
		return errors.New("v3 contract must be an append-only 12-activity extension of v2")
	}
	if strings.Join(contract.Precedence, ">") != "REFUTED>UNKNOWN>CLOSED" {
		return errors.New("v3 contract precedence must be REFUTED > UNKNOWN > CLOSED")
	}
	if !sameStringSet(contract.IdentityFields, V3IdentityFields) || !sameStringSet(contract.PriorRunFields, V3PriorRunFields) || !sameStringSet(contract.ObservationFields, V3ObservationFields) || !sameStringSet(contract.UnknownFields, V3UnknownFields) {
		return errors.New("v3 contract identity, observation, and UNKNOWN fields are not exact")
	}
	for _, forbidden := range []string{"total_score", "weighted_score", "estimated_time", "estimated_savings"} {
		if !contains(contract.ForbiddenOutputs, forbidden) {
			return fmt.Errorf("v3 contract must forbid output %q", forbidden)
		}
	}
	for _, choice := range []string{V3ProofFoundation, V3ProofCoherence, V3ProofRegression} {
		if contract.ProofTotals[choice] != 4 {
			return fmt.Errorf("v3 contract proof total for %s must be four", choice)
		}
	}
	for _, class := range []string{V3IndicatorDriver, V3IndicatorOutcome, V3IndicatorGuardrail} {
		if contract.IndicatorTotals[class] != 4 {
			return fmt.Errorf("v3 contract indicator total for %s must be four", class)
		}
	}
	seen := map[string]bool{}
	seenOrdinals := map[int]bool{}
	for _, activity := range contract.Activities {
		if activity.Ordinal < 1 || activity.Ordinal > 12 || activity.ID == "" || seen[activity.ID] || seenOrdinals[activity.Ordinal] {
			return fmt.Errorf("v3 contract has an invalid or duplicate activity %q", activity.ID)
		}
		seen[activity.ID] = true
		seenOrdinals[activity.Ordinal] = true
	}
	if len(contract.Scenarios) < 3 {
		return errors.New("v3 contract must include normal, UNKNOWN, and REFUTED scenarios")
	}
	seenExpected := map[string]bool{}
	for _, scenario := range contract.Scenarios {
		if scenario.ID == "" || scenario.Source == "" || (scenario.Expected != V3DecisionClosed && scenario.Expected != V3DecisionUnknown && scenario.Expected != V3DecisionRefuted) {
			return errors.New("v3 contract has an invalid scenario")
		}
		seenExpected[scenario.Expected] = true
	}
	for _, expected := range []string{V3DecisionClosed, V3DecisionUnknown, V3DecisionRefuted} {
		if !seenExpected[expected] {
			return fmt.Errorf("v3 contract is missing a %s scenario", expected)
		}
	}
	return nil
}

func LoadV3Fixture(path string) (V3Fixture, error) {
	var fixture V3Fixture
	if err := readJSON(path, &fixture); err != nil {
		return V3Fixture{}, err
	}
	if fixture.Schema != "gooo/incremental-conformance-planner/fixture/v3" || fixture.CaseID == "" || fixture.FixtureAnchor == "" || len(fixture.Activities) == 0 {
		return V3Fixture{}, fmt.Errorf("fixture %s must define v3 schema, case_id, fixture_anchor, and activities", path)
	}
	if fixture.Kind != "NORMAL" && fixture.Kind != V3DecisionUnknown && fixture.Kind != V3DecisionRefuted {
		return V3Fixture{}, fmt.Errorf("fixture %s has invalid kind %q", path, fixture.Kind)
	}
	expectedKind := fixture.Kind
	if fixture.Kind == "NORMAL" {
		expectedKind = V3DecisionClosed
	}
	if fixture.Expected.Decision != expectedKind {
		return V3Fixture{}, fmt.Errorf("fixture %s expected decision must match kind", path)
	}
	seen := map[string]bool{}
	for _, activity := range fixture.Activities {
		if activity.ActivityID == "" || seen[activity.ActivityID] {
			return V3Fixture{}, fmt.Errorf("fixture %s has a duplicate or empty activity input", path)
		}
		seen[activity.ActivityID] = true
	}
	if fixture.FixedPoint != nil && (!fixture.FixedPoint.Declared || fixture.FixedPoint.Kind != V3FixedPointExplicit || fixture.FixedPoint.Witness == "") {
		return V3Fixture{}, fmt.Errorf("fixture %s has a malformed or implicit fixed-point declaration", path)
	}
	return fixture, nil
}

func LoadV3ActionsReceipt(path string) (V3ActionsReceipt, error) {
	var receipt V3ActionsReceipt
	if err := readJSON(path, &receipt); err != nil {
		return V3ActionsReceipt{}, err
	}
	if receipt.Schema != "gooo/incremental-conformance-planner/actions-receipt/v3" {
		return V3ActionsReceipt{}, fmt.Errorf("actions receipt %s has an invalid v3 schema", path)
	}
	return receipt, nil
}
