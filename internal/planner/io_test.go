package planner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadJSONRejectsTrailingValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipt.json")
	if err := os.WriteFile(path, []byte(`{"schema":"gooo/test/v1"} {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var value struct {
		Schema string `json:"schema"`
	}
	if err := readJSON(path, &value); err == nil {
		t.Fatal("readJSON accepted a trailing JSON value")
	}
}
