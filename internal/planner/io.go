package planner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func DigestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func DigestJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return DigestBytes(data), nil
}

func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func writeText(path, value string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(value), 0o644)
}

func pathFrom(root, value string) string {
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Clean(filepath.Join(root, value))
}

func physicalLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	lines := strings.Count(string(data), "\n")
	if data[len(data)-1] != '\n' {
		lines++
	}
	return lines
}

type Inventory struct {
	RootREADMEExcluded bool `json:"root_readme_excluded"`
	GitExcluded        bool `json:"git_excluded"`
	GeneratedExcluded  bool `json:"generated_excluded"`
	RegularFiles       int  `json:"regular_files"`
	DescendantDirs     int  `json:"descendant_dirs"`
	GoFiles            int  `json:"go_files"`
	GoooFiles          int  `json:"gooo_files"`
	PhysicalLines      int  `json:"physical_lines"`
	GoPhysicalLines    int  `json:"go_physical_lines"`
	GoooPhysicalLines  int  `json:"gooo_physical_lines"`
}

func BuildInventory(root string) (Inventory, error) {
	result := Inventory{RootREADMEExcluded: true, GitExcluded: true, GeneratedExcluded: true}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if entry.IsDir() && excludedDir(entry.Name()) {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			result.DescendantDirs++
			return nil
		}
		if relative == "README.md" || !entry.Type().IsRegular() {
			return nil
		}
		result.RegularFiles++
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := physicalLines(data)
		result.PhysicalLines += lines
		switch filepath.Ext(entry.Name()) {
		case ".go":
			result.GoFiles++
			result.GoPhysicalLines += lines
		case ".gooo":
			result.GoooFiles++
			result.GoooPhysicalLines += lines
		}
		return nil
	})
	return result, err
}

func excludedDir(name string) bool {
	name = strings.ToLower(name)
	if name == ".git" || name == ".cache" || name == "cache" || name == "vendor" || name == "generated" || name == "dist" || name == "bin" {
		return true
	}
	return strings.Contains(name, "toolchain")
}

func fileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return DigestBytes(data), nil
}

func sortedCaseSpecs(items []CaseSpec) []CaseSpec {
	items = append([]CaseSpec(nil), items...)
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}
