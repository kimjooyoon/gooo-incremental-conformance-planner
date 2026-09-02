#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "${CI:-}" != "true" || "${GITHUB_ACTIONS:-}" != "true" ]]; then
  echo "CI-only conformance: refusing local execution" >&2
  exit 2
fi

binary=${1:?usage: conformance.sh BINARY OUTPUT_DIR}
output_dir=${2:?usage: conformance.sh BINARY OUTPUT_DIR}
"$binary" conformance \
  --meta .gooo/incremental-conformance-planner.gooo \
  --contract contracts/denominator-v1.json \
  --cases-root . \
  --out "$output_dir"
