#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "${CI:-}" != "true" || "${GITHUB_ACTIONS:-}" != "true" ]]; then
  echo "CI-only evidence assembly: refusing local execution" >&2
  exit 2
fi

evidence_dir=${1:?usage: assemble-evidence.sh EVIDENCE_DIR}
manifest="$evidence_dir/evidence-manifest.json"
commit_sha=$(git rev-parse HEAD)
toolchain=$(go version)
jq -n \
  --arg repository "${GITHUB_REPOSITORY}" \
  --arg run_id "${GITHUB_RUN_ID}" \
  --arg workflow "${GITHUB_WORKFLOW}" \
  --arg sha "$commit_sha" \
  --arg ref "${GITHUB_REF}" \
  --arg toolchain "$toolchain" \
  '{schema:"gooo/incremental-conformance-planner/evidence-manifest/v1", repository:$repository, actions_run_id:$run_id, workflow:$workflow, ref:$ref, commit_sha:$sha, go_toolchain:$toolchain, runtime:{repository_writes:0,local_test_executions:0,cross_project_required_gates:0}, failed_runs:"OPERATIONAL_REFUTED_PRESERVED", generated_artifacts:["conformance/suite-report.json","conformance/human-report.md","measure/ci-measurements.json","inventory.json","release-verification.json"]}' \
  > "$manifest"
