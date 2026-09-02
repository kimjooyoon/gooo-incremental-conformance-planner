package planner

import (
	"fmt"
	"strings"
)

func RenderReport(report Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\nDecision: **%s**\n\n", report.CaseID, report.Decision)
	fmt.Fprintf(&b, "Precedence: `%s`\n\n", strings.Join(report.Precedence, " > "))
	b.WriteString("## Semantic impact\n\n")
	fmt.Fprintf(&b, "Changed nodes: `%s`  \nImpacted nodes: `%s`  \nCausal edges: `%s`\n\n", joinOrNone(report.Impact.ChangedNodes), joinOrNone(report.Impact.ImpactedNodes), joinOrNone(report.Impact.Edges))
	fmt.Fprintf(&b, "Fixed-point assessment: state=`%s`, declared=%t, case_mode=`%s`, rule=`%s`; %s\n\n", report.FixedPoint.State, report.FixedPoint.Declared, report.FixedPoint.CaseMode, report.FixedPoint.Rule, report.FixedPoint.Reason)
	b.WriteString("## Per-unit execution plan\n\n")
	b.WriteString("| Unit | Planned | Action | Executed | Reused | Unknown | Refuted | build_ms | test_ms | wall_ms | peak_rss_kib | Reason |\n|---|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|---|\n")
	for _, unit := range report.Units {
		fmt.Fprintf(&b, "| `%s` | %t | `%s` | %t | %t | %t | %t | %s | %s | %s | %s | %s |\n", unit.UnitID, unit.Planned, unit.Action, unit.Executed, unit.Reused, unit.Unknown, unit.Refuted, metricText(unit.Metrics.BuildMS), metricText(unit.Metrics.TestMS), metricText(unit.Metrics.WallMS), metricText(unit.Metrics.PeakRSSKiB), unit.Reason)
	}
	b.WriteString("\nEach row is an independent validation unit. Missing metrics are `UNKNOWN`; they are never rendered as zero. No aggregate score or percentage is emitted.\n\n")
	b.WriteString("## Matched-identity indicators\n\n")
	b.WriteString("| Metric | Before | After | Signed delta | Same identity | State | Reason |\n|---|---:|---:|---:|---:|---|---|\n")
	for _, item := range report.Indicators {
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %t | `%s` | %s |\n", item.Metric, metricText(item.Before), metricText(item.After), metricText(item.SignedDelta), item.IdentityPair, item.State, item.Reason)
	}
	b.WriteString("\nSpeed and memory improvement claims remain `UNKNOWN` unless before and after carry the same complete identity.\n\n")
	if len(report.Unknowns) > 0 {
		b.WriteString("## UNKNOWN evidence\n\n")
		for _, item := range report.Unknowns {
			fmt.Fprintf(&b, "- `%s/%s`: class=`%s`; %s; next `%s`; blocked by `%s`\n", item.Stage, item.Step, item.Class, item.Reason, item.Next, joinOrNone(item.BlockedBy))
		}
		b.WriteString("\n")
	}
	if len(report.Refutations) > 0 {
		b.WriteString("## REFUTED evidence\n\n")
		for _, item := range report.Refutations {
			fmt.Fprintf(&b, "- `%s/%s`: %s; next `%s`; blocked by `%s`\n", item.Stage, item.Step, item.Reason, item.Next, joinOrNone(item.BlockedBy))
		}
		b.WriteString("\n")
	}
	b.WriteString("## Boundaries\n\n")
	fmt.Fprintf(&b, "Optional slicer: consumed=%t, required=%t, cross_project_gate=%t; %s\n\n", report.OptionalSlicer.Consumed, report.OptionalSlicer.Required, report.OptionalSlicer.CrossProjectGate, report.OptionalSlicer.Reason)
	fmt.Fprintf(&b, "Operational audit: state=`%s`, repository_writes=%d, local_test_executions=%d, cross_project_required_gates=%d, failed_runs_preserved=%t.\n", report.Operational.State, report.Operational.RepositoryWrites, report.Operational.LocalTestExecutions, report.Operational.CrossProjectRequiredGates, report.Operational.FailedRunsPreserved)
	return b.String()
}

func RenderSuiteReport(denominator Denominator, report SuiteReport) string {
	var b strings.Builder
	b.WriteString("# Incremental conformance planner\n\n")
	fmt.Fprintf(&b, "Suite decision: **%s**\n\n", report.Decision)
	fmt.Fprintf(&b, "Fixed contract: 12 cells; proof 4 CLOSED / 4 UNKNOWN / 4 REFUTED; indicator 4 OBSERVED / 4 UNKNOWN / 4 REFUTED.\n\n")
	b.WriteString("| Case | Expected | Actual | Match | Report |\n|---|---|---|---:|---|\n")
	for _, item := range report.Cases {
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | %t | `%s` |\n", item.ID, item.Expected, item.Actual, item.Match, item.Report)
	}
	b.WriteString("\nThe case vector is the conformance evidence. It does not calculate an aggregate score or percentage.\n\n")
	fmt.Fprintf(&b, "Operational audit: state=`%s`, repository_writes=%d, local_test_executions=%d, cross_project_required_gates=%d, failed_runs_preserved=%t.\n", report.Operational.State, report.Operational.RepositoryWrites, report.Operational.LocalTestExecutions, report.Operational.CrossProjectRequiredGates, report.Operational.FailedRunsPreserved)
	_ = denominator
	return b.String()
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func metricText(value *int64) string {
	if value == nil {
		return "UNKNOWN"
	}
	return fmt.Sprintf("%d", *value)
}
