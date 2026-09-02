#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "${CI:-}" != "true" || "${GITHUB_ACTIONS:-}" != "true" ]]; then
  echo "CI-only assertions: refusing local execution" >&2
  exit 2
fi

evidence_dir=${1:?usage: assert-evidence.sh EVIDENCE_DIR}
suite="$evidence_dir/conformance/suite-report.json"
measure="$evidence_dir/measure/ci-measurements.json"

jq -e '
  .decision == "CLOSED" and
  (.cases | length == 12) and
  ([.cases[] | select(.match == true)] | length == 12) and
  ([.cases[] | .actual] | group_by(.) | map({state: .[0], count: length}) | sort_by(.state)) ==
    [{state:"CLOSED",count:4},{state:"REFUTED",count:4},{state:"UNKNOWN",count:4}]
' "$suite" >/dev/null

jq -e '
  (.operational_audit.repository_writes == 0) and
  (.operational_audit.local_test_executions == 0) and
  (.operational_audit.cross_project_required_gates == 0) and
  (.operational_audit.failed_runs_preserved == true)
' "$suite" >/dev/null

jq -e '
  (.schema == "gooo/incremental-conformance-planner/ci-measurements/v1") and
  (.build_ms | type == "number") and (.test_ms | type == "number") and
  (.wall_ms | type == "number") and (.peak_rss_kib | type == "number") and
  (.repository_writes == 0) and (.local_test_executions == 0) and
  (.cross_project_required_gates == 0)
' "$measure" >/dev/null

echo "evidence assertions passed"
