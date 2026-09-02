package planner

import (
	"fmt"
	"strings"
)

func RenderV2Report(report V2Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\nDecision: **%s**\n\n", report.CaseID, report.Decision)
	fmt.Fprintf(&b, "Precedence: `%s`\n\n", strings.Join(report.Precedence, " > "))
	fmt.Fprintf(&b, "Semantic IR: `%s`; evaluator: `%s`; fixture: `%s`; contract: `%s`\n\n", report.SemanticIR.Digest, report.EvaluatorDigest, report.FixtureDigest, report.ContractDigest)
	fmt.Fprintf(&b, "## CI dossier denominator\n\nTotal activities: **%d**  \nRequired runs: **%d**  \nReused closed: **%d**  \nUNKNOWN: **%d**  \nREFUTED: **%d**  \nExecuted: **%d**  \nSkipped with proof: **%d**\n\n", report.Summary.TotalActivities, report.Summary.RequiredRuns, report.Summary.ReusedClosed, report.Summary.Unknown, report.Summary.Refuted, report.Summary.Executed, report.Summary.SkippedWithProof)
	b.WriteString("A reused activity is closed only by an exact identity and an independent immutable PASS receipt. A cache hit is recorded as a fact and never closes an activity by itself.\n\n")
	b.WriteString("## Activity vector\n\n")
	b.WriteString("| Activity | Proof | Indicator | Action | Already verified | Required run | Executed | Reused closed | Skipped with proof | Cache hit | Cache miss | Duration ms | Reason |\n|---|---|---|---|---:|---:|---:|---:|---:|---|---|---:|---|\n")
	for _, activity := range report.Activities {
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | `%s` | %t | %t | %t | %t | %t | %s | %s | %s | %s |\n", activity.ActivityID, activity.ProofChoice, activity.IndicatorClass, activity.Action, activity.AlreadyVerified, activity.RequiredRun, activity.Executed, activity.ReusedClosed, activity.SkippedWithProof, boolText(activity.CacheHit), boolText(activity.CacheMiss), metricText(activity.DurationMS), activity.Reason)
	}
	b.WriteString("\n`REQUIRED_RUN` means this activity must be reverified now. `REUSED_CLOSED` means the prior proof is reused for this unaffected activity.\n\n")
	b.WriteString("## Actions observations\n\n")
	b.WriteString("| Metric | Before | After | Improvement | Same exact pair | State | Reason |\n|---|---:|---:|---:|---:|---|---|\n")
	for _, indicator := range report.Indicators {
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %t | `%s` | %s |\n", indicator.Metric, metricText(indicator.Before), metricText(indicator.After), metricText(indicator.Improvement), indicator.SameIdentity, indicator.State, indicator.Reason)
	}
	b.WriteString("\nImprovement is null and UNKNOWN unless scenario, source, semantic IR, fixture, contract, evaluator, toolchain, and activity identities match exactly.\n\n")
	if len(report.UnknownReasons) > 0 {
		b.WriteString("## UNKNOWN evidence\n\n")
		for _, reason := range report.UnknownReasons {
			fmt.Fprintf(&b, "- `%s`\n", reason)
		}
		b.WriteString("\n")
	}
	if len(report.RefutedReasons) > 0 {
		b.WriteString("## REFUTED evidence\n\n")
		for _, reason := range report.RefutedReasons {
			fmt.Fprintf(&b, "- `%s`\n", reason)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "Operational boundary: repository_writes=%d, local_test_executions=%d, cross_project_required_gates=%d, failed_runs_preserved=%t, verification_authority=%s.\n", report.Operational.RepositoryWrites, report.Operational.LocalTestExecutions, report.Operational.CrossProjectRequiredGates, report.Operational.FailedRunsPreserved, report.Operational.VerificationAuthority)
	return b.String()
}

func RenderV2SuiteReport(report V2SuiteReport) string {
	var b strings.Builder
	b.WriteString("# Incremental conformance planner v2\n\n")
	fmt.Fprintf(&b, "Suite decision: **%s**\n\n", report.Decision)
	fmt.Fprintf(&b, "Append-only contract: `%s` (%s), total activities=%d; precedence=`REFUTED > UNKNOWN > CLOSED`.\n\n", report.Contract, report.ContractDigest, report.TotalActivities)
	fmt.Fprintf(&b, "Actions metric state: `%s`; missing metrics: `%s`.\n\n", report.ActionsMetricState, joinOrNone(report.MissingActionsMetrics))
	b.WriteString("| Scenario | Expected | Actual | Match | Report |\n|---|---|---|---:|---|\n")
	for _, item := range report.Cases {
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | %t | `%s` |\n", item.ID, item.Expected, item.Actual, item.Match, item.Report)
	}
	b.WriteString("\nThe dossier reports exact activity counts and observations. It emits no aggregate score, percentage, or estimated time.\n\n")
	fmt.Fprintf(&b, "Operational boundary: repository_writes=%d, local_test_executions=%d, cross_project_required_gates=%d, failed_runs_preserved=%t, verification_authority=%s.\n", report.Operational.RepositoryWrites, report.Operational.LocalTestExecutions, report.Operational.CrossProjectRequiredGates, report.Operational.FailedRunsPreserved, report.Operational.VerificationAuthority)
	return b.String()
}

func boolText(value *bool) string {
	if value == nil {
		return "UNKNOWN"
	}
	if *value {
		return "true"
	}
	return "false"
}
