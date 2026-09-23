package planner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseV3SourceRejectsDuplicateFixedPointCases(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".gooo", "incremental-conformance-planner-v3.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, []byte("\nfixed_point_case duplicate-test EXPLICIT_FIXED_POINT\nfixed_point_case duplicate-test EXPLICIT_FIXED_POINT\n")...)
	path := filepath.Join(t.TempDir(), "planner-v3.gooo")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ParseV3Source(path); err == nil {
		t.Fatal("ParseV3Source accepted duplicate fixed-point case IDs")
	}
}

func TestMissingActionsSemanticIRIsUnknownEvidence(t *testing.T) {
	missing := missingV3ActionsEvidence(V3ActionsReceipt{})
	for _, field := range missing {
		if field == "semantic_ir_digest" {
			return
		}
	}
	t.Fatalf("missing semantic IR provenance was not reported: %v", missing)
}
