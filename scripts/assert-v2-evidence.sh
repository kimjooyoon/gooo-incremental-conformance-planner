#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "${CI:-}" != "true" || "${GITHUB_ACTIONS:-}" != "true" ]]; then
  echo "CI-only v2 assertions: refusing local execution" >&2
  exit 2
fi

evidence_dir=${1:?usage: assert-v2-evidence.sh EVIDENCE_DIR}
suite="$evidence_dir/conformance-v2/suite-report.json"
actions="$evidence_dir/measure/actions-receipt.json"

jq -e '
  .decision == "CLOSED" and .actions_metric_state == "OBSERVED" and .total_activities == 12 and (.cases | length == 4) and ([.cases[] | select(.match == true)] | length == 4) and
  ([.cases[].id] | sort) == ["exact-reuse", "forged-receipt", "missing-provenance", "required-rerun"] and
  (.operational.repository_writes == 0) and (.operational.local_test_executions == 0) and (.operational.cross_project_required_gates == 0) and (.operational.failed_runs_preserved == true)
' "$suite" >/dev/null

jq -e '
  (.decision == "CLOSED") and (.dossier_summary == {total_activities:12,required_runs:0,reused_closed:12,unknown:0,refuted:0,executed:0,skipped_with_proof:12}) and ([.activities[] | select(.action == "REUSED_CLOSED")] | length == 12) and ([.activities[] | select(.already_verified == true and .skipped_with_proof == true)] | length == 12) and ([.indicators[] | select(.state == "OBSERVED" and .improvement != null)] | length == 4)
' "$evidence_dir/conformance-v2/cases/exact-reuse/report.json" >/dev/null

jq -e '
  (.decision == "CLOSED") and (.dossier_summary == {total_activities:12,required_runs:3,reused_closed:9,unknown:0,refuted:0,executed:3,skipped_with_proof:9}) and ([.activities[] | select(.action == "REQUIRED_RUN" and .executed == true)] | length == 3) and ([.activities[] | select(.action == "REUSED_CLOSED")] | length == 9) and ([.activities[] | select(.action == "REUSED_CLOSED" and .cache_hit == true)] | length == 9)
' "$evidence_dir/conformance-v2/cases/required-rerun/report.json" >/dev/null

jq -e '
  (.decision == "UNKNOWN") and (.dossier_summary.total_activities == 12) and ([.activities[] | select(.action == "UNKNOWN" and (.reason == "MISSING_PROVENANCE"))] | length == 1) and ([.activities[] | select(.cache_hit == true and .action == "REUSED_CLOSED")] | length == 11) and any(.indicators[]; .metric == "wall_ms" and .after == null and .state == "UNKNOWN" and .improvement == null)
' "$evidence_dir/conformance-v2/cases/missing-provenance/report.json" >/dev/null

jq -e '
  (.decision == "REFUTED") and ([.activities[] | select(.action == "REFUTED")] | length >= 5) and any(.activities[]; .reason == "FORGED_OR_STALE_RECEIPT") and any(.activities[]; .reason == "EVALUATOR_SELF_APPROVAL") and any(.activities[]; .reason == "AFFECTED_ACTIVITY_SKIPPED") and ([.indicators[] | select(.state == "REFUTED" and .improvement == null)] | length == 4)
' "$evidence_dir/conformance-v2/cases/forged-receipt/report.json" >/dev/null

jq -e '
  .schema == "gooo/incremental-conformance-planner/actions-receipt/v2" and (.build_ms | type == "number") and (.test_ms | type == "number") and (.wall_ms | type == "number") and (.peak_rss_kib | type == "number") and (.cache_hits == null or (.cache_hits | type == "number")) and (.cache_misses == null or (.cache_misses | type == "number")) and ([.activities[] | select(.activity_id == "BUILD_REPOSITORY" and (.duration_ms | type == "number"))] | length == 1) and ([.activities[] | select(.activity_id == "TEST_REPOSITORY" and (.duration_ms | type == "number"))] | length == 1)
' "$actions" >/dev/null

echo "v2 evidence assertions passed"
