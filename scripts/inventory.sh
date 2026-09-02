#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "${CI:-}" != "true" || "${GITHUB_ACTIONS:-}" != "true" ]]; then
  echo "CI-only inventory: refusing local execution" >&2
  exit 2
fi

evidence_dir=${1:?usage: inventory.sh EVIDENCE_DIR}
mkdir -p "$evidence_dir"
files=$(find . -type f ! -path './.git/*' ! -path './.github/generated/*' ! -name README.md ! -path './vendor/*' ! -path './toolchain/*' ! -path './dist/*' ! -path './bin/*' -print)
regular_files=$(printf '%s\n' "$files" | sed '/^$/d' | wc -l | tr -d ' ')
descendant_dirs=$(find . -type d ! -path './.git*' ! -path './vendor*' ! -path './toolchain*' ! -path './dist*' ! -path './bin*' | sed '1d' | wc -l | tr -d ' ')
go_files=$(printf '%s\n' "$files" | sed '/^$/d' | awk '$0 ~ /\.go$/ {n++} END {print n+0}')
gooo_files=$(printf '%s\n' "$files" | sed '/^$/d' | awk '$0 ~ /\.gooo$/ {n++} END {print n+0}')
physical_lines=$(printf '%s\n' "$files" | sed '/^$/d' | xargs wc -l | tail -n 1 | awk '{print $1}')
go_physical_lines=$(find . -type f -name '*.go' ! -path './.git/*' ! -path './vendor/*' ! -path './toolchain/*' ! -path './dist/*' ! -path './bin/*' -print0 | xargs -0 wc -l | tail -n 1 | awk '{print $1}')
gooo_physical_lines=$(find . -type f -name '*.gooo' ! -path './.git/*' -print0 | xargs -0 wc -l | tail -n 1 | awk '{print $1}')
included_files=$(printf '%s\n' "$files" | sed '/^$/d' | sort | jq -R -s 'split("\n") | map(select(length > 0))')
included_files_digest=$(printf '%s\n' "$files" | sed '/^$/d' | sort | sha256sum | cut -d' ' -f1)

jq -n \
  --argjson regular_files "$regular_files" \
  --argjson descendant_dirs "$descendant_dirs" \
  --argjson go_files "$go_files" \
  --argjson gooo_files "$gooo_files" \
  --argjson physical_lines "${physical_lines:-0}" \
  --argjson go_physical_lines "${go_physical_lines:-0}" \
  --argjson gooo_physical_lines "${gooo_physical_lines:-0}" \
  --argjson included_files "$included_files" \
  --arg included_files_digest "sha256:$included_files_digest" \
  '{schema:"gooo/incremental-conformance-planner/inventory/v1", root_readme_excluded:true, git_excluded:true, generated_excluded:true, included_files:$included_files, included_files_digest:$included_files_digest, regular_files:$regular_files, descendant_dirs:$descendant_dirs, go_files:$go_files, gooo_files:$gooo_files, physical_lines:$physical_lines, go_physical_lines:$go_physical_lines, gooo_physical_lines:$gooo_physical_lines, repository_writes:0, local_test_executions:0, cross_project_required_gates:0}' \
  > "$evidence_dir/inventory.json"
