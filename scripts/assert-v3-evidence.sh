#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "${CI:-}" != "true" || "${GITHUB_ACTIONS:-}" != "true" ]]; then
  echo "CI-only v3 assertions: refusing local execution" >&2
  exit 2
fi

evidence_dir=${1:?usage: assert-v3-evidence.sh EVIDENCE_DIR}
root="$evidence_dir/conformance-v3"
suite="$root/suite-report.json"
actions="$root/actions-receipt.json"

jq -e '
  .decision == "CLOSED" and .contract == "incremental-conformance-planner-v3" and .total_activities == 12 and
  .proof_totals == {FOUNDATION:4,COHERENCE:4,REGRESSION:4} and
  .indicator_totals == {DRIVER:4,OUTCOME:4,GUARDRAIL:4} and
  .legacy_exact_reuse_closed == true and .actions_metric_state == "OBSERVED" and
  ([.cases[] | select(.match == true)] | length == 4) and
  ([.cases[].id] | sort) == ["exact-reuse", "missing-evidence", "refuted-evidence", "required-rerun"] and
  (.operational.repository_writes == 0) and (.operational.local_test_executions == 0) and (.operational.cross_project_required_gates == 0) and (.operational.failed_runs_preserved == true)
' "$suite" >/dev/null

jq -e '
  .decision == "CLOSED" and .dossier_summary == {total_activities:12,reusable_evidence:12,required_runs:0,unknown:0,refuted:0,executed:0,skipped_with_proof:12} and
  ([.activities[] | select(.action == "REUSED_CLOSED" and .reusable_evidence == true and .already_verified == true and .skipped_with_proof == true)] | length == 12) and
  ([.indicators[] | select(.state == "OBSERVED" and (.delta | type == "number") and (.improvement | type == "number"))] | length == 4)
' "$root/cases/exact-reuse/report.json" >/dev/null

jq -e '
  .decision == "CLOSED" and .dossier_summary == {total_activities:12,reusable_evidence:8,required_runs:4,unknown:0,refuted:0,executed:4,skipped_with_proof:8} and
  ([.activities[] | select(.action == "REQUIRED_RUN" and .must_execute == true and .executed == true)] | length == 4) and
  ([.activities[] | select(.action == "REUSED_CLOSED")] | length == 8) and
  any(.activities[]; .mismatched_fields | index("test_identity") != null)
' "$root/cases/required-rerun/report.json" >/dev/null

jq -e '
  .decision == "UNKNOWN" and .dossier_summary.total_activities == 12 and
  any(.activities[]; .action == "UNKNOWN" and .unknown_record.unknown_class == "MISSING_IDENTITY" and (.unknown_record | keys | sort) == ["blocked_by","next_operation","reason","stage","step","unknown_class"]) and
  any(.indicators[]; .metric == "wall_ms" and .after == null and .delta == null and .improvement == null and .state == "UNKNOWN" and .unknown_record != null)
' "$root/cases/missing-evidence/report.json" >/dev/null

jq -e '
  .decision == "REFUTED" and ([.activities[] | select(.action == "REFUTED")] | length >= 5) and
  any(.activities[]; .reason == "FORGED_OR_STALE_RECEIPT") and
  any(.activities[]; .reason == "EVALUATOR_SELF_APPROVAL") and
  any(.activities[]; .reason == "AFFECTED_ACTIVITY_SKIPPED") and
  ([.indicators[] | select(.state == "REFUTED" and .delta == null and .improvement == null)] | length == 4)
' "$root/cases/refuted-evidence/report.json" >/dev/null

jq -e '
  .schema == "gooo/incremental-conformance-planner/actions-receipt/v3" and
  (.run_identity.run_id | type == "string" and length > 0) and (.run_identity.commit_sha | type == "string" and length > 0) and (.run_identity.receipt_digest | type == "string" and length > 0) and
  (.test_identity | type == "string" and length > 0) and (.conformance_identity | type == "string" and length > 0) and (.input_digest | type == "string" and length > 0) and (.toolchain_digest | type == "string" and length > 0) and (.semantic_ir_digest | type == "string" and length > 0) and
  (.build_ms | type == "number") and (.test_ms | type == "number") and (.wall_ms | type == "number") and (.peak_rss_kib | type == "number") and
  (.cache_hit | type == "boolean") and (.cache_miss | type == "boolean") and (.cache_hits | type == "number") and (.cache_misses | type == "number") and
  ([.activities[] | select(.activity_id == "BUILD_REPOSITORY" and (.duration_ms | type == "number"))] | length == 1) and
  ([.activities[] | select(.activity_id == "TEST_REPOSITORY" and (.duration_ms | type == "number"))] | length == 1)
' "$actions" >/dev/null

if rg -n '"(total_score|weighted_score|estimated_time|estimated_savings)"' "$root" >/dev/null; then
  echo "v3 output contains a forbidden aggregate or estimate field" >&2
  exit 1
fi

echo "v3 evidence assertions passed"
