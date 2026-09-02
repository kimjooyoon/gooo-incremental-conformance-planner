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
		fatal("usage: gooo-incremental-conformance-planner <plan|conformance|conformance-v2> [flags]")
	}
	switch os.Args[1] {
	case "plan":
		plan(os.Args[2:])
	case "conformance":
		conformance(os.Args[2:])
	case "conformance-v2":
		conformanceV2(os.Args[2:])
	default:
		fatal("command must be plan, conformance, or conformance-v2")
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

func conformanceV2(args []string) {
	flags := flag.NewFlagSet("conformance-v2", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	meta := flags.String("meta", ".gooo/incremental-conformance-planner-v2.gooo", "authoritative v2 .gooo source")
	contract := flags.String("contract", "contracts/denominator-v2.json", "append-only v2 activity contract")
	casesRoot := flags.String("cases-root", ".", "repository root containing fixture paths")
	actionsReceipt := flags.String("actions-receipt", "", "actual GitHub Actions build/test receipt")
	out := flags.String("out", "", "absolute caller-owned output directory")
	if err := flags.Parse(args); err != nil {
		os.Exit(2)
	}
	if *out == "" || !filepath.IsAbs(*out) {
		fatal("--out must be an absolute caller-owned path")
	}
	report, err := planner.RunV2Suite(planner.V2SuiteOptions{MetaPath: *meta, ContractPath: *contract, CasesRoot: *casesRoot, OutputDir: *out, ActionsReceiptPath: *actionsReceipt})
	if err != nil {
		fatal(err.Error())
	}
	printJSON(struct {
		Decision string `json:"decision"`
		Report   string `json:"report"`
	}{report.Decision, filepath.Join(*out, "suite-report.json")})
	if report.Decision != planner.V2DecisionClosed {
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
