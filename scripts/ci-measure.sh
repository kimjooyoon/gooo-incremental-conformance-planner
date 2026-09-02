#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "${CI:-}" != "true" || "${GITHUB_ACTIONS:-}" != "true" ]]; then
  echo "CI-only measurement: refusing local execution" >&2
  exit 2
fi

evidence_dir=${1:?usage: ci-measure.sh EVIDENCE_DIR}
mkdir -p "$evidence_dir"

run_stage() {
  local stage=$1
  shift
  local time_file="$evidence_dir/${stage}.time"
  local start_ns end_ns status elapsed_s rss_kib elapsed_ms
  start_ns=$(date +%s%N)
  set +e
  /usr/bin/time -f '%e %M' -o "$time_file" "$@"
  status=$?
  set -e
  end_ns=$(date +%s%N)
  read -r elapsed_s rss_kib < "$time_file"
  elapsed_ms=$(awk -v value="$elapsed_s" 'BEGIN { printf "%.0f", value * 1000 }')
  jq -n \
    --arg stage "$stage" \
    --argjson exit_code "$status" \
    --argjson elapsed_ms "$elapsed_ms" \
    --argjson peak_rss_kib "$rss_kib" \
    '{stage:$stage, exit_code:$exit_code, elapsed_ms:$elapsed_ms, peak_rss_kib:$peak_rss_kib}' \
    > "$evidence_dir/${stage}.json"
  return "$status"
}

build_status=0
test_status=0
overall_start=$(date +%s%N)
if run_stage build go build -o "$RUNNER_TEMP/gooo-incremental-conformance-planner" ./cmd/gooo-incremental-conformance-planner; then
  build_status=0
else
  build_status=$?
fi
if run_stage test go test ./...; then
  test_status=0
else
  test_status=$?
fi
overall_end=$(date +%s%N)

build_ms=$(jq -r '.elapsed_ms' "$evidence_dir/build.json")
test_ms=$(jq -r '.elapsed_ms' "$evidence_dir/test.json")
build_rss=$(jq -r '.peak_rss_kib' "$evidence_dir/build.json")
test_rss=$(jq -r '.peak_rss_kib' "$evidence_dir/test.json")
wall_ms=$(( (overall_end - overall_start) / 1000000 ))
peak_rss_kib=$build_rss
if (( test_rss > peak_rss_kib )); then
  peak_rss_kib=$test_rss
fi
exit_code=0
if (( build_status != 0 )); then
  exit_code=$build_status
elif (( test_status != 0 )); then
  exit_code=$test_status
fi
operational_state=PASS
if (( exit_code != 0 )); then
  operational_state=OPERATIONAL_REFUTED
fi

jq -n \
  --argjson build_ms "$build_ms" \
  --argjson test_ms "$test_ms" \
  --argjson wall_ms "$wall_ms" \
  --argjson peak_rss_kib "$peak_rss_kib" \
  --argjson build_exit_code "$build_status" \
  --argjson test_exit_code "$test_status" \
  --argjson exit_code "$exit_code" \
  --arg state "$operational_state" \
  '{schema:"gooo/incremental-conformance-planner/ci-measurements/v1", build_ms:$build_ms, test_ms:$test_ms, wall_ms:$wall_ms, peak_rss_kib:$peak_rss_kib, build_exit_code:$build_exit_code, test_exit_code:$test_exit_code, exit_code:$exit_code, operational_state:$state, repository_writes:0, local_test_executions:0, cross_project_required_gates:0}' \
  > "$evidence_dir/ci-measurements.json"

if (( exit_code != 0 )); then
  exit 1
fi
