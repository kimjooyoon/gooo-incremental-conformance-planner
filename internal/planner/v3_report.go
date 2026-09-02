package planner

import (
	"fmt"
	"strings"
)

func RenderV3Report(report V3Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\nDecision: **%s**\n\n", report.CaseID, report.Decision)
	fmt.Fprintf(&b, "Precedence: `%s`\n\n", strings.Join(report.Precedence, " > "))
	fmt.Fprintf(&b, "Semantic IR: `%s`; evaluator: `%s`; fixture: `%s`; contract: `%s`; fixed-point: `%s`.\n\n", report.SemanticIR.Digest, report.EvaluatorDigest, report.FixtureDigest, report.ContractDigest, report.FixedPointState)
	b.WriteString("## Deterministic next CI selection\n\n")
	b.WriteString("| Activity | Proof | Indicator | Action | Reusable evidence | Must execute | Already verified | Required run | Executed | Reuse state | Measurement state | build_ms | test_ms | wall_ms | peak_rss_kib | cache_hit | cache_miss | Reason |\n|---|---|---|---|---:|---:|---:|---:|---:|---|---|---:|---:|---:|---:|---|---|---|\n")
	for _, activity := range report.Activities {
		priorRun := "none"
		if activity.PriorRunIdentity != nil {
			priorRun = activity.PriorRunIdentity.RunID
		}
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | `%s` | %t | %t | %t | %t | %t | `%s` | `%s` | %s | %s | %s | %s | %s | %s | %s | %s |\n", activity.ActivityID, activity.ProofChoice, activity.IndicatorClass, activity.Action, activity.ReusableEvidence, activity.MustExecute, activity.AlreadyVerified, activity.RequiredRun, activity.Executed, activity.ReuseClaimState, activity.MeasurementState, metricText(activity.BuildMS), metricText(activity.TestMS), metricText(activity.WallMS), metricText(activity.PeakRSSKiB), boolText(activity.CacheHit), boolText(activity.CacheMiss), priorRun, activity.Reason)
	}
	fmt.Fprintf(&b, "\nExact denominator: %d activities; reusable evidence=%d; required runs=%d; UNKNOWN=%d; REFUTED=%d; executed=%d; skipped with proof=%d.\n\n", report.Summary.TotalActivities, report.Summary.ReusableEvidence, report.Summary.RequiredRuns, report.Summary.Unknown, report.Summary.Refuted, report.Summary.Executed, report.Summary.SkippedWithProof)
	b.WriteString("The selection is determined per activity by semantic impact, exact identity, independent immutable PASS provenance, and current observation status. Cache hits and misses are observations only.\n\n")
	b.WriteString("## Per-indicator exact pair observations\n\n")
	b.WriteString("| Indicator | Before | After | Integer delta | Same exact pair | State | Reason |\n|---|---:|---:|---:|---:|---|---|\n")
	for _, indicator := range report.Indicators {
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %t | `%s` | %s |\n", indicator.Metric, metricText(indicator.Before), metricText(indicator.After), metricText(indicator.Delta), indicator.SameIdentity, indicator.State, indicator.Reason)
	}
	b.WriteString("\nAn integer delta is emitted only for an exact matched before/after identity pair with both observations present. Otherwise the value is `null` in machine evidence and the state is `UNKNOWN`. No total or weighted score, expected time, or estimated saving is produced.\n\n")
	if len(report.Unknowns) > 0 {
		b.WriteString("## UNKNOWN six-field evidence\n\n")
		for _, unknown := range report.Unknowns {
			fmt.Fprintf(&b, "- stage=`%s`; step=`%s`; reason=`%s`; unknown_class=`%s`; next_operation=`%s`; blocked_by=`%s`\n", unknown.Stage, unknown.Step, unknown.Reason, unknown.UnknownClass, unknown.NextOperation, joinOrNone(unknown.BlockedBy))
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

func RenderV3SuiteReport(report V3SuiteReport) string {
	var b strings.Builder
	b.WriteString("# Incremental conformance planner v3\n\n")
	fmt.Fprintf(&b, "Suite decision: **%s**\n\n", report.Decision)
	fmt.Fprintf(&b, "Append-only contract: `%s` (%s), total activities=%d; precedence=`REFUTED > UNKNOWN > CLOSED`.\n\n", report.Contract, report.ContractDigest, report.TotalActivities)
	fmt.Fprintf(&b, "Proof denominator: FOUNDATION=%d, COHERENCE=%d, REGRESSION=%d. Indicator denominator: DRIVER=%d, OUTCOME=%d, GUARDRAIL=%d.\n\n", report.ProofTotals[V3ProofFoundation], report.ProofTotals[V3ProofCoherence], report.ProofTotals[V3ProofRegression], report.IndicatorTotals[V3IndicatorDriver], report.IndicatorTotals[V3IndicatorOutcome], report.IndicatorTotals[V3IndicatorGuardrail])
	fmt.Fprintf(&b, "Legacy exact-reuse semantics preserved: **%t**. Actions metric state: `%s`; missing fields: `%s`.\n\n", report.LegacyExactReuseClosed, report.ActionsMetricState, joinOrNone(report.MissingActionsMetrics))
	b.WriteString("| Scenario | Expected | Actual | Match | Report |\n|---|---|---|---:|---|\n")
	for _, item := range report.Cases {
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | %t | `%s` |\n", item.ID, item.Expected, item.Actual, item.Match, item.Report)
	}
	b.WriteString("\nThe dossier contains exact activity selection, exact identity/provenance, observations, and per-indicator integer deltas. It does not calculate a score, percentage, expected time, or estimated saving.\n\n")
	fmt.Fprintf(&b, "Operational boundary: repository_writes=%d, local_test_executions=%d, cross_project_required_gates=%d, failed_runs_preserved=%t, verification_authority=%s.\n", report.Operational.RepositoryWrites, report.Operational.LocalTestExecutions, report.Operational.CrossProjectRequiredGates, report.Operational.FailedRunsPreserved, report.Operational.VerificationAuthority)
	return b.String()
}

