package planner

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func ParseMeta(path string) (Meta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Meta{}, err
	}
	meta := Meta{
		Schema:     path,
		Precedence: []string{}, Authorities: []string{}, ReuseRules: []string{}, UnknownClasses: []string{}, FixedPointRules: []string{}, FixedPointCases: []FixedPointCase{},
		Activities: []string{}, ProofCells: []Cell{}, IndicatorCells: []Cell{},
		ForbiddenEffects: []string{}, SourcePath: path, SourceDigest: DigestBytes(data),
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
			return Meta{}, fmt.Errorf("line %d: declaration has no value", lineNumber)
		}
		switch fields[0] {
		case "schema":
			meta.Schema = fields[1]
		case "program":
			meta.Program = fields[1]
		case "namespace":
			meta.Namespace = fields[1]
		case "precedence":
			meta.Precedence = strings.Fields(strings.ReplaceAll(strings.Join(fields[1:], " "), ">", " "))
		case "authority":
			meta.Authorities = append(meta.Authorities, fields[1])
		case "unknown_class":
			if len(fields) < 3 || fields[1] != "enum" {
				return Meta{}, fmt.Errorf("line %d: unknown_class requires enum values", lineNumber)
			}
			meta.UnknownClasses = append(meta.UnknownClasses, fields[2:]...)
		case "fixed_point_rule":
			meta.FixedPointRules = append(meta.FixedPointRules, strings.Join(fields[1:], " "))
		case "fixed_point_case":
			if len(fields) != 3 {
				return Meta{}, fmt.Errorf("line %d: fixed_point_case requires case id and mode", lineNumber)
			}
			meta.FixedPointCases = append(meta.FixedPointCases, FixedPointCase{ID: fields[1], Mode: fields[2]})
		case "validation_unit":
			meta.ValidationUnit = strings.Join(fields[1:], " ")
		case "semantic_dependency":
			meta.Dependency = strings.Join(fields[1:], " ")
		case "cache_identity":
			meta.CacheIdentity = strings.Join(fields[1:], " ")
		case "reuse_rule":
			meta.ReuseRules = append(meta.ReuseRules, strings.Join(fields[1:], " "))
		case "execution_plan":
			meta.ExecutionPlan = strings.Join(fields[1:], " ")
		case "meta_activity":
			meta.Activities = append(meta.Activities, fields[1])
		case "proof_cell":
			if len(fields) != 3 {
				return Meta{}, fmt.Errorf("line %d: proof_cell requires state and id", lineNumber)
			}
			meta.ProofCells = append(meta.ProofCells, Cell{State: fields[1], ID: fields[2]})
		case "indicator_cell":
			if len(fields) != 3 {
				return Meta{}, fmt.Errorf("line %d: indicator_cell requires state and id", lineNumber)
			}
			meta.IndicatorCells = append(meta.IndicatorCells, Cell{State: fields[1], ID: fields[2]})
		case "optional_input":
			meta.OptionalInput = parseOptionalInput(fields[1:])
		case "forbidden_effect":
			meta.ForbiddenEffects = append(meta.ForbiddenEffects, fields[1])
		default:
			return Meta{}, fmt.Errorf("line %d: unknown .gooo declaration %q", lineNumber, fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return Meta{}, err
	}
	if err := validateMeta(meta); err != nil {
		return Meta{}, err
	}
	return meta, nil
}

func parseOptionalInput(fields []string) OptionalInput {
	input := OptionalInput{Name: fields[0]}
	for _, field := range fields[1:] {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		switch key {
		case "release":
			input.Release = value
		case "digest_pinned":
			input.DigestPinned = value == "true"
		case "required":
			input.Required = value == "true"
		case "cross_project_gate":
			input.CrossProjectGate = value == "true"
		case "copied":
			input.Copied = value == "true"
		}
	}
	return input
}

func validateMeta(meta Meta) error {
	if meta.Schema != "gooo/incremental-conformance-planner/meta/v1" || meta.Program == "" || meta.Namespace == "" {
		return fmt.Errorf(".gooo must define the incremental planner schema, program, and namespace")
	}
	if len(meta.Precedence) != 3 || meta.Precedence[0] != DecisionRefuted || meta.Precedence[1] != DecisionUnknown || meta.Precedence[2] != DecisionClosed {
		return fmt.Errorf(".gooo precedence must be REFUTED > UNKNOWN > CLOSED")
	}
	for _, required := range RequiredAuthorities {
		if !contains(meta.Authorities, required) {
			return fmt.Errorf(".gooo is missing authority %q", required)
		}
	}
	if meta.ValidationUnit == "" || meta.Dependency == "" || meta.CacheIdentity == "" || meta.ExecutionPlan == "" {
		return fmt.Errorf(".gooo is missing one or more authority schemas")
	}
	if len(meta.UnknownClasses) != len(RequiredUnknownClasses) {
		return fmt.Errorf(".gooo must declare exactly %d UNKNOWN classes", len(RequiredUnknownClasses))
	}
	for _, required := range RequiredUnknownClasses {
		if !contains(meta.UnknownClasses, required) {
			return fmt.Errorf(".gooo is missing UNKNOWN class %q", required)
		}
	}
	if len(meta.FixedPointRules) != len(RequiredFixedPointRules) {
		return fmt.Errorf(".gooo must declare exactly %d fixed-point rules", len(RequiredFixedPointRules))
	}
	for _, required := range RequiredFixedPointRules {
		if !contains(meta.FixedPointRules, required) {
			return fmt.Errorf(".gooo is missing fixed-point rule %q", required)
		}
	}
	if len(meta.FixedPointCases) != 3 {
		return fmt.Errorf(".gooo must own exactly 3 fixed-point cases")
	}
	for _, required := range []FixedPointCase{
		{ID: "exact-cache-hit", Mode: "EXPLICIT_FIXED_POINT"},
		{ID: "unmatched-before-after", Mode: "FAIL_CLOSED_UNKNOWN"},
		{ID: "known-counterexample", Mode: "MALFORMED_OR_IMPLICIT_COUNTEREXAMPLE"},
	} {
		if !hasFixedPointCase(meta.FixedPointCases, required) {
			return fmt.Errorf(".gooo is missing fixed-point case %q/%q", required.ID, required.Mode)
		}
	}
	if len(meta.Activities) != len(RequiredActivities) {
		return fmt.Errorf(".gooo must define exactly %d meta activities", len(RequiredActivities))
	}
	for _, required := range RequiredActivities {
		if !contains(meta.Activities, required) {
			return fmt.Errorf(".gooo is missing meta activity %q", required)
		}
	}
	if len(meta.ProofCells) != 12 || len(meta.IndicatorCells) != 12 {
		return fmt.Errorf(".gooo must define exactly 12 proof cells and 12 indicator cells")
	}
	if cellStateCount(meta.ProofCells, DecisionClosed) != 4 || cellStateCount(meta.ProofCells, DecisionUnknown) != 4 || cellStateCount(meta.ProofCells, DecisionRefuted) != 4 {
		return fmt.Errorf(".gooo proof cells must be 4 CLOSED, 4 UNKNOWN, and 4 REFUTED")
	}
	if cellStateCount(meta.IndicatorCells, IndicatorObserved) != 4 || cellStateCount(meta.IndicatorCells, IndicatorUnknown) != 4 || cellStateCount(meta.IndicatorCells, IndicatorRefuted) != 4 {
		return fmt.Errorf(".gooo indicator cells must be 4 OBSERVED, 4 UNKNOWN, and 4 REFUTED")
	}
	if meta.OptionalInput.Name != "gooo-semantic-impact-slicer" || meta.OptionalInput.Release != "v0.1.1" || !meta.OptionalInput.DigestPinned || meta.OptionalInput.Required || meta.OptionalInput.CrossProjectGate || meta.OptionalInput.Copied {
		return fmt.Errorf("optional slicer input must be digest-pinned v0.1.1, non-required, non-gating, and non-copied")
	}
	return nil
}

func hasFixedPointCase(cases []FixedPointCase, wanted FixedPointCase) bool {
	for _, item := range cases {
		if item.ID == wanted.ID && item.Mode == wanted.Mode {
			return true
		}
	}
	return false
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func cellStateCount(cells []Cell, state string) int {
	count := 0
	for _, cell := range cells {
		if cell.State == state {
			count++
		}
	}
	return count
}

func LoadFixture(path string) (Fixture, error) {
	var fixture Fixture
	if err := readJSON(path, &fixture); err != nil {
		return Fixture{}, err
	}
	if fixture.Schema == "" || fixture.CaseID == "" || len(fixture.Units) == 0 {
		return Fixture{}, fmt.Errorf("fixture %s must define schema, case_id, and validation units", path)
	}
	return fixture, nil
}

func LoadDenominator(path string) (Denominator, error) {
	var denominator Denominator
	if err := readJSON(path, &denominator); err != nil {
		return Denominator{}, err
	}
	if !denominator.Fixed || denominator.Cells != 12 || len(denominator.Cases) != 12 {
		return Denominator{}, fmt.Errorf("denominator must be fixed at 12 cases/cells")
	}
	if denominator.ProofDenominator[DecisionClosed] != 4 || denominator.ProofDenominator[DecisionUnknown] != 4 || denominator.ProofDenominator[DecisionRefuted] != 4 {
		return Denominator{}, fmt.Errorf("proof denominator must be 4/4/4")
	}
	if denominator.IndicatorDenominator[IndicatorObserved] != 4 || denominator.IndicatorDenominator[IndicatorUnknown] != 4 || denominator.IndicatorDenominator[IndicatorRefuted] != 4 {
		return Denominator{}, fmt.Errorf("indicator denominator must be 4/4/4")
	}
	return denominator, nil
}
