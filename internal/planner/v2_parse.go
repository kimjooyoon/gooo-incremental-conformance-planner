package planner

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ParseV2Source(path string) (V2Source, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return V2Source{}, "", err
	}
	source := V2Source{
		Precedence: []string{}, Authorities: []string{}, IdentityFields: []string{}, UnknownClasses: []string{},
		ProofChoices: []string{}, IndicatorClasses: []string{}, Activities: []V2ActivityDescriptor{},
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
			return V2Source{}, "", fmt.Errorf("line %d: declaration has no value", lineNumber)
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
		case "cache_identity":
			source.IdentityFields = strings.Fields(strings.Join(fields[1:], " "))
		case "unknown_class":
			if len(fields) < 3 || fields[1] != "enum" {
				return V2Source{}, "", fmt.Errorf("line %d: unknown_class requires enum values", lineNumber)
			}
			source.UnknownClasses = append(source.UnknownClasses, fields[2:]...)
		case "toolchain_digest":
			source.ToolchainDigest = fields[1]
		case "semantic_change_set", "proof_graph", "activity_identity", "prior_immutable_receipt", "semantic_ir", "evaluator":
			// These declarations are structural authority owned by the source.
		case "proof_choice":
			source.ProofChoices = append(source.ProofChoices, fields[1])
		case "indicator_class":
			source.IndicatorClasses = append(source.IndicatorClasses, fields[1])
		case "activity":
			if len(fields) < 9 {
				return V2Source{}, "", fmt.Errorf("line %d: activity requires ordinal, id, stage, step, proof, indicator, and dependencies", lineNumber)
			}
			ordinal, err := strconv.Atoi(fields[1])
			if err != nil {
				return V2Source{}, "", fmt.Errorf("line %d: activity ordinal: %w", lineNumber, err)
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
			return V2Source{}, "", fmt.Errorf("line %d: unknown v2 .gooo declaration %q", lineNumber, fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return V2Source{}, "", err
	}
	if err := validateV2Source(source); err != nil {
		return V2Source{}, "", err
	}
	return source, DigestBytes(data), nil
}

func validateV2Source(source V2Source) error {
	if source.Schema != "gooo/incremental-conformance-planner/meta/v2" || source.Program == "" || source.Namespace == "" {
		return errors.New("v2 .gooo must define the planner schema, program, and namespace")
	}
	if strings.Join(source.Precedence, ">") != "REFUTED>UNKNOWN>CLOSED" {
		return errors.New("v2 .gooo precedence must be REFUTED > UNKNOWN > CLOSED")
	}
	for _, required := range []string{"SemanticChangeSet", "ProofGraph", "ActivityIdentity", "CacheIdentity", "PriorImmutableReceipt", "SemanticIR", "Evaluator", "ActionsReceipt", "Dossier"} {
		if !contains(source.Authorities, required) {
			return fmt.Errorf("v2 .gooo is missing authority %q", required)
		}
	}
	if strings.Join(source.IdentityFields, ",") != strings.Join(V2IdentityDigestFields, ",") {
		return errors.New("v2 .gooo must declare exactly the six cache identity digests")
	}
	if len(source.UnknownClasses) != 4 || !contains(source.UnknownClasses, V2UnknownMissingIdentity) || !contains(source.UnknownClasses, V2UnknownMissingProvenance) || !contains(source.UnknownClasses, V2UnknownMissingReceipt) || !contains(source.UnknownClasses, V2UnknownMissingMetrics) {
		return errors.New("v2 .gooo must declare the four unknown classes")
	}
	if source.ToolchainDigest == "" {
		return errors.New("v2 .gooo must declare the Go toolchain digest")
	}
	if len(source.ProofChoices) != 3 || !contains(source.ProofChoices, V2ProofFoundation) || !contains(source.ProofChoices, V2ProofCoherence) || !contains(source.ProofChoices, V2ProofRegression) {
		return errors.New("v2 .gooo must declare FOUNDATION, COHERENCE, and REGRESSION")
	}
	if len(source.IndicatorClasses) != 3 || !contains(source.IndicatorClasses, V2IndicatorDriver) || !contains(source.IndicatorClasses, V2IndicatorOutcome) || !contains(source.IndicatorClasses, V2IndicatorGuardrail) {
		return errors.New("v2 .gooo must declare DRIVER, OUTCOME, and GUARDRAIL")
	}
	if len(source.Activities) != 12 {
		return errors.New("v2 .gooo must declare exactly 12 activities")
	}
	seen := map[string]bool{}
	seenOrdinals := map[int]bool{}
	proofCounts := map[string]int{}
	indicatorCounts := map[string]int{}
	for _, activity := range source.Activities {
		if activity.Ordinal < 1 || activity.Ordinal > 12 || activity.ID == "" || activity.Activity == "" || activity.Stage == "" || activity.Step == "" || seen[activity.ID] || seenOrdinals[activity.Ordinal] {
			return fmt.Errorf("v2 .gooo has an invalid or duplicate activity %q", activity.ID)
		}
		seen[activity.ID] = true
		seenOrdinals[activity.Ordinal] = true
		proofCounts[activity.ProofChoice]++
		indicatorCounts[activity.IndicatorClass]++
	}
	for _, choice := range []string{V2ProofFoundation, V2ProofCoherence, V2ProofRegression} {
		if proofCounts[choice] != 4 {
			return fmt.Errorf("v2 .gooo proof choice %s must have four activities", choice)
		}
	}
	for _, class := range []string{V2IndicatorDriver, V2IndicatorOutcome, V2IndicatorGuardrail} {
		if indicatorCounts[class] != 4 {
			return fmt.Errorf("v2 .gooo indicator class %s must have four activities", class)
		}
	}
	return nil
}

func BuildV2SemanticIR(source V2Source, sourcePath, sourceDigest string) (V2SemanticIR, error) {
	ir := V2SemanticIR{
		Schema:  "gooo/incremental-conformance-planner/semantic-ir/v2",
		Program: source.Program, SourcePath: sourcePath, SourceDigest: sourceDigest, ToolchainDigest: source.ToolchainDigest,
		IdentityFields: append([]string(nil), source.IdentityFields...),
		Activities:     append([]V2ActivityDescriptor(nil), source.Activities...),
	}
	canonical := struct {
		Schema          string                 `json:"schema"`
		Program         string                 `json:"program"`
		SourceDigest    string                 `json:"source_digest"`
		ToolchainDigest string                 `json:"toolchain_digest"`
		IdentityFields  []string               `json:"identity_fields"`
		Activities      []V2ActivityDescriptor `json:"activities"`
	}{ir.Schema, ir.Program, ir.SourceDigest, ir.ToolchainDigest, ir.IdentityFields, ir.Activities}
	digest, err := DigestJSON(canonical)
	if err != nil {
		return V2SemanticIR{}, err
	}
	ir.Digest = digest
	ir.EvaluatorDigest = DigestBytes([]byte("gooo-incremental-conformance-planner/v2/evaluator/" + digest))
	return ir, nil
}

func LoadV2Contract(path string) (V2Contract, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return V2Contract{}, "", err
	}
	var contract V2Contract
	if err := readJSON(path, &contract); err != nil {
		return V2Contract{}, "", err
	}
	if err := validateV2Contract(contract); err != nil {
		return V2Contract{}, "", err
	}
	return contract, DigestBytes(data), nil
}

func validateV2Contract(contract V2Contract) error {
	if contract.Schema != "gooo/incremental-conformance-planner/denominator/v2" || contract.Version != "v2" || contract.AppendOnlyFrom != "v1" || contract.TargetActivities != 12 || len(contract.Activities) != 12 {
		return errors.New("v2 contract must be an append-only 12-activity extension of v1")
	}
	if strings.Join(contract.Precedence, ">") != "REFUTED>UNKNOWN>CLOSED" {
		return errors.New("v2 contract precedence must be REFUTED > UNKNOWN > CLOSED")
	}
	for _, choice := range []string{V2ProofFoundation, V2ProofCoherence, V2ProofRegression} {
		if contract.ProofTotals[choice] != 4 {
			return fmt.Errorf("v2 contract proof total for %s must be four", choice)
		}
	}
	for _, class := range []string{V2IndicatorDriver, V2IndicatorOutcome, V2IndicatorGuardrail} {
		if contract.IndicatorTotals[class] != 4 {
			return fmt.Errorf("v2 contract indicator total for %s must be four", class)
		}
	}
	seen := map[string]bool{}
	seenOrdinals := map[int]bool{}
	for _, activity := range contract.Activities {
		if activity.Ordinal < 1 || activity.Ordinal > 12 || activity.ID == "" || seen[activity.ID] || seenOrdinals[activity.Ordinal] {
			return fmt.Errorf("v2 contract has an invalid or duplicate activity %q", activity.ID)
		}
		seen[activity.ID] = true
		seenOrdinals[activity.Ordinal] = true
	}
	if len(contract.Scenarios) < 3 {
		return errors.New("v2 contract must include normal, UNKNOWN, and REFUTED scenarios")
	}
	return nil
}

func LoadV2Fixture(path string) (V2Fixture, error) {
	var fixture V2Fixture
	if err := readJSON(path, &fixture); err != nil {
		return V2Fixture{}, err
	}
	if fixture.Schema != "gooo/incremental-conformance-planner/fixture/v2" || fixture.CaseID == "" || fixture.FixtureAnchor == "" || len(fixture.Activities) == 0 {
		return V2Fixture{}, fmt.Errorf("fixture %s must define v2 schema, case_id, fixture_anchor, and activities", path)
	}
	return fixture, nil
}

func LoadV2ActionsReceipt(path string) (V2ActionsReceipt, error) {
	var receipt V2ActionsReceipt
	if err := readJSON(path, &receipt); err != nil {
		return V2ActionsReceipt{}, err
	}
	if receipt.Schema != "gooo/incremental-conformance-planner/actions-receipt/v2" {
		return V2ActionsReceipt{}, fmt.Errorf("actions receipt %s has an invalid schema", path)
	}
	return receipt, nil
}
