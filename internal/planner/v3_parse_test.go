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

func TestParseV3SourceRejectsImplicitFixedPointMode(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".gooo", "incremental-conformance-planner-v3.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, []byte("\nfixed_point_case implicit-test NOT_APPLICABLE\n")...)
	path := filepath.Join(t.TempDir(), "planner-v3.gooo")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ParseV3Source(path); err == nil {
		t.Fatal("ParseV3Source accepted an implicit fixed-point mode")
	}
}
