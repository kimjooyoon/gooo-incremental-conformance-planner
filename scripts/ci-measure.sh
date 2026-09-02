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
cache_hit=${GOOO_ACTION_CACHE_HIT:-}
jq -n \
  --arg run_id "${GITHUB_RUN_ID:-}" \
  --arg cache_hit "$cache_hit" \
  --argjson build_ms "$build_ms" \
  --argjson test_ms "$test_ms" \
  --argjson wall_ms "$wall_ms" \
  --argjson peak_rss_kib "$peak_rss_kib" \
  --argjson build_exit_code "$build_status" \
  --argjson test_exit_code "$test_status" \
  --arg state "$operational_state" \
  '{schema:"gooo/incremental-conformance-planner/actions-receipt/v2", run_id:$run_id, build_ms:$build_ms, test_ms:$test_ms, wall_ms:$wall_ms, peak_rss_kib:$peak_rss_kib,
    cache_hits:(if $cache_hit == "true" then 1 elif $cache_hit == "false" then 0 else null end),
    cache_misses:(if $cache_hit == "true" then 0 elif $cache_hit == "false" then 1 else null end),
    activities:[
      {activity_id:"BUILD_REPOSITORY", status:(if $build_exit_code == 0 then "EXECUTED" else "OPERATIONAL_REFUTED" end), duration_ms:$build_ms, build_ms:$build_ms, test_ms:null, wall_ms:$build_ms, peak_rss_kib:$peak_rss_kib, cache_hit:(if $cache_hit == "true" then true elif $cache_hit == "false" then false else null end), cache_miss:(if $cache_hit == "true" then false elif $cache_hit == "false" then true else null end)},
      {activity_id:"TEST_REPOSITORY", status:(if $test_exit_code == 0 then "EXECUTED" else "OPERATIONAL_REFUTED" end), duration_ms:$test_ms, build_ms:null, test_ms:$test_ms, wall_ms:$test_ms, peak_rss_kib:$peak_rss_kib, cache_hit:(if $cache_hit == "true" then true elif $cache_hit == "false" then false else null end), cache_miss:(if $cache_hit == "true" then false elif $cache_hit == "false" then true else null end)}
    ], operational_state:$state, repository_writes:0, local_test_executions:0, cross_project_required_gates:0}' \
  > "$evidence_dir/actions-receipt.json"

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

test_identity=$(printf '%s' 'go test ./...' | sha256sum | cut -d' ' -f1)
conformance_identity=$(printf '%s' 'gooo-incremental-conformance-planner conformance-v3 --meta .gooo/incremental-conformance-planner-v3.gooo --contract contracts/denominator-v3.json --cases-root . --actions-receipt measure/actions-receipt-v3.json' | sha256sum | cut -d' ' -f1)
input_digest=$(printf '%s' "${GITHUB_SHA:-}" | sha256sum | cut -d' ' -f1)
receipt_digest=$(printf '%s|%s|%s|%s|%s' "${GITHUB_RUN_ID:-}" "${GITHUB_SHA:-}" "$build_ms" "$test_ms" "$wall_ms" | sha256sum | cut -d' ' -f1)
jq -n \
  --arg run_id "${GITHUB_RUN_ID:-}" \
  --arg commit_sha "${GITHUB_SHA:-}" \
  --arg receipt_digest "sha256:$receipt_digest" \
  --arg test_identity "sha256:$test_identity" \
  --arg conformance_identity "sha256:$conformance_identity" \
  --arg input_digest "sha256:$input_digest" \
  --arg toolchain_digest "sha256:go1.27.0" \
  --arg semantic_ir_digest "" \
  --arg cache_hit "$cache_hit" \
  --argjson build_ms "$build_ms" \
  --argjson test_ms "$test_ms" \
  --argjson wall_ms "$wall_ms" \
  --argjson peak_rss_kib "$peak_rss_kib" \
  --argjson build_exit_code "$build_status" \
  --argjson test_exit_code "$test_status" \
  --arg state "$operational_state" \
  '{schema:"gooo/incremental-conformance-planner/actions-receipt/v3", run_identity:{run_id:$run_id,commit_sha:$commit_sha,receipt_digest:$receipt_digest}, test_identity:$test_identity, conformance_identity:$conformance_identity, input_digest:$input_digest, toolchain_digest:$toolchain_digest, semantic_ir_digest:$semantic_ir_digest,
    build_ms:$build_ms, test_ms:$test_ms, wall_ms:$wall_ms, peak_rss_kib:$peak_rss_kib,
    cache_hit:(if $cache_hit == "true" then true elif $cache_hit == "false" then false else null end),
    cache_miss:(if $cache_hit == "true" then false elif $cache_hit == "false" then true else null end),
    cache_hits:(if $cache_hit == "true" then 1 elif $cache_hit == "false" then 0 else null end),
    cache_misses:(if $cache_hit == "true" then 0 elif $cache_hit == "false" then 1 else null end),
    activities:[
      {activity_id:"BUILD_REPOSITORY", status:(if $build_exit_code == 0 then "EXECUTED" else "OPERATIONAL_REFUTED" end), duration_ms:$build_ms, build_ms:$build_ms, test_ms:null, wall_ms:$build_ms, peak_rss_kib:$peak_rss_kib, cache_hit:(if $cache_hit == "true" then true elif $cache_hit == "false" then false else null end), cache_miss:(if $cache_hit == "true" then false elif $cache_hit == "false" then true else null end)},
      {activity_id:"TEST_REPOSITORY", status:(if $test_exit_code == 0 then "EXECUTED" else "OPERATIONAL_REFUTED" end), duration_ms:$test_ms, build_ms:null, test_ms:$test_ms, wall_ms:$test_ms, peak_rss_kib:$peak_rss_kib, cache_hit:(if $cache_hit == "true" then true elif $cache_hit == "false" then false else null end), cache_miss:(if $cache_hit == "true" then false elif $cache_hit == "false" then true else null end)}
    ], operational_state:$state, repository_writes:0, local_test_executions:0, cross_project_required_gates:0}' \
  > "$evidence_dir/actions-receipt-v3.json"

if (( exit_code != 0 )); then
  exit 1
fi
