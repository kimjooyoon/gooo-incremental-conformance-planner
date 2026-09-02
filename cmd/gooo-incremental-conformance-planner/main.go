package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-incremental-conformance-planner/internal/planner"
)

func main() {
	if len(os.Args) < 2 {
		fatal("usage: gooo-incremental-conformance-planner <plan|conformance> [flags]")
	}
	switch os.Args[1] {
	case "plan":
		plan(os.Args[2:])
	case "conformance":
		conformance(os.Args[2:])
	default:
		fatal("command must be plan or conformance")
	}
}

func plan(args []string) {
	flags := flag.NewFlagSet("plan", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	meta := flags.String("meta", ".gooo/incremental-conformance-planner.gooo", "authoritative .gooo source")
	contract := flags.String("contract", "contracts/denominator-v1.json", "fixed 12-cell contract")
	fixture := flags.String("fixture", "", "fixture JSON")
	out := flags.String("out", "", "absolute caller-owned output directory")
	if err := flags.Parse(args); err != nil {
		os.Exit(2)
	}
	if *fixture == "" || *out == "" {
		fatal("--fixture and --out are required")
	}
	if !filepath.IsAbs(*out) {
		fatal("--out must be an absolute caller-owned output directory")
	}
	report, err := planner.Run(planner.RunOptions{MetaPath: *meta, ContractPath: *contract, FixturePath: *fixture, OutputDir: *out})
	if err != nil {
		fatal(err.Error())
	}
	printJSON(struct {
		CaseID   string `json:"case_id"`
		Decision string `json:"decision"`
		Report   string `json:"report"`
	}{report.CaseID, report.Decision, filepath.Join(*out, "report.json")})
}

func conformance(args []string) {
	flags := flag.NewFlagSet("conformance", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	meta := flags.String("meta", ".gooo/incremental-conformance-planner.gooo", "authoritative .gooo source")
	contract := flags.String("contract", "contracts/denominator-v1.json", "fixed 12-cell contract")
	casesRoot := flags.String("cases-root", ".", "repository root containing fixture paths")
	out := flags.String("out", "", "absolute caller-owned output directory")
	if err := flags.Parse(args); err != nil {
		os.Exit(2)
	}
	if *out == "" || !filepath.IsAbs(*out) {
		fatal("--out must be an absolute caller-owned output directory")
	}
	report, err := planner.RunSuite(planner.SuiteOptions{MetaPath: *meta, ContractPath: *contract, CasesRoot: *casesRoot, OutputDir: *out})
	if err != nil {
		fatal(err.Error())
	}
	printJSON(struct {
		Decision string `json:"decision"`
		Report   string `json:"report"`
	}{report.Decision, filepath.Join(*out, "suite-report.json")})
	if report.Decision != planner.DecisionClosed {
		os.Exit(1)
	}
}

func printJSON(value any) {
	data, err := json.Marshal(value)
	if err != nil {
		fatal(err.Error())
	}
	fmt.Println(string(data))
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
