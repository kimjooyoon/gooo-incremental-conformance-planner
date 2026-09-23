package planner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIdentityRequiresAllSixDigests(t *testing.T) {
	identity := CacheIdentity{SourceDigest: "source", SemanticIRDigest: "ir", FixtureDigest: "fixture", ContractDigest: "contract", GoToolchainDigest: "toolchain"}
	missing := identity.Missing()
	if len(missing) != 1 || missing[0] != "command_descriptor_digest" {
		t.Fatalf("missing = %v", missing)
	}
}

func TestFailedProofIsNotReusable(t *testing.T) {
	unit := ValidationUnit{
		ID: "unit", Command: "go test ./...", SemanticNodes: []string{"node"},
		CurrentIdentity: completeIdentity("v1"),
		Cache:           &CacheReceipt{State: ProofFailed, Immutable: true, Identity: completeIdentity("v1"), ResultDigest: "result"},
	}
	plan := planUnit(unit, ImpactClosure{}, nil)
	if plan.Action != ActionRefuted || plan.Reused || !plan.Refuted {
		t.Fatalf("plan = %+v", plan)
	}
}

func TestImpactClosurePropagatesForward(t *testing.T) {
	before := SemanticGraph{Nodes: []SemanticNode{{ID: "a", Digest: "a1"}, {ID: "b", Digest: "b1"}, {ID: "c", Digest: "c1"}}, Edges: []SemanticDependency{{From: "a", To: "b", Relation: "causal"}, {From: "b", To: "c", Relation: "causal"}}}
	after := SemanticGraph{Nodes: []SemanticNode{{ID: "a", Digest: "a2"}, {ID: "b", Digest: "b1"}, {ID: "c", Digest: "c1"}}, Edges: before.Edges}
	impact, refutations := computeImpact(before, after)
	if len(refutations) != 0 || !intersects([]string{"c"}, impact.ImpactedNodes) {
		t.Fatalf("impact = %+v, refutations = %+v", impact, refutations)
	}
}

func TestMetaOwnsUnknownClassesAndFixedPointCases(t *testing.T) {
	meta, err := ParseMeta(filepath.Join("..", "..", ".gooo", "incremental-conformance-planner.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	if len(meta.UnknownClasses) != 4 || len(meta.FixedPointRules) != 3 || len(meta.FixedPointCases) != 3 {
		t.Fatalf("meta unknown/fixed-point declarations = %v/%v/%v", meta.UnknownClasses, meta.FixedPointRules, meta.FixedPointCases)
	}
}

func TestOptionalInputRejectsUnknownAndMalformedFields(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".gooo", "incremental-conformance-planner.gooo"))
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name   string
		field  string
		wantIn string
	}{
		{name: "unknown field", field: "digest_pinnd=true", wantIn: "optional_input gooo-semantic-impact-slicer"},
		{name: "bare field", field: "required", wantIn: "optional_input gooo-semantic-impact-slicer"},
		{name: "invalid boolean", field: "required=maybe", wantIn: "optional_input gooo-semantic-impact-slicer"},
	} {
		t.Run(test.name, func(t *testing.T) {
			line := test.wantIn + " release=v0.1.1 digest_pinned=true required=false cross_project_gate=false copied=false " + test.field + "\n"
			mutated := strings.Replace(string(data), "optional_input gooo-semantic-impact-slicer release=v0.1.1 digest_pinned=true required=false cross_project_gate=false copied=false\n", line, 1)
			path := filepath.Join(t.TempDir(), "invalid.gooo")
			if err := os.WriteFile(path, []byte(mutated), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := ParseMeta(path); err == nil {
				t.Fatalf("ParseMeta accepted %s", test.field)
			}
		})
	}
}

func TestUnknownTopDecisionCannotBecomeFixedPoint(t *testing.T) {
	meta, err := ParseMeta(filepath.Join("..", "..", ".gooo", "incremental-conformance-planner.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := LoadFixture(filepath.Join("..", "..", "fixtures", "cases", "unmatched-before-after.json"))
	if err != nil {
		t.Fatal(err)
	}
	report, err := Plan(meta, fixture, "contract:test")
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision != DecisionUnknown || report.UnknownClass != "UNMATCHED_BEFORE_AFTER_IDENTITY" || report.FixedPoint.State != FixedPointUnknown {
		t.Fatalf("report = %+v", report)
	}
}

func TestImplicitFixedPointCounterexampleIsRefuted(t *testing.T) {
	meta, err := ParseMeta(filepath.Join("..", "..", ".gooo", "incremental-conformance-planner.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := LoadFixture(filepath.Join("..", "..", "fixtures", "cases", "known-counterexample.json"))
	if err != nil {
		t.Fatal(err)
	}
	report, err := Plan(meta, fixture, "contract:test")
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision != DecisionRefuted || report.FixedPoint.State != FixedPointRefuted {
		t.Fatalf("report = %+v", report)
	}
}

func completeIdentity(version string) CacheIdentity {
	return CacheIdentity{
		SourceDigest: "source:" + version, SemanticIRDigest: "ir:" + version,
		FixtureDigest: "fixture:" + version, ContractDigest: "contract:" + version,
		GoToolchainDigest: "toolchain:" + version, CommandDescriptorDigest: "command:" + version,
	}
}
