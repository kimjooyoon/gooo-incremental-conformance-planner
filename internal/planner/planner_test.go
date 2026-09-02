package planner

import "testing"

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

func completeIdentity(version string) CacheIdentity {
	return CacheIdentity{
		SourceDigest: "source:" + version, SemanticIRDigest: "ir:" + version,
		FixtureDigest: "fixture:" + version, ContractDigest: "contract:" + version,
		GoToolchainDigest: "toolchain:" + version, CommandDescriptorDigest: "command:" + version,
	}
}
