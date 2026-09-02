#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "${CI:-}" != "true" || "${GITHUB_ACTIONS:-}" != "true" ]]; then
  echo "CI-only v3 conformance: refusing local execution" >&2
  exit 2
fi

binary=${1:?usage: conformance-v3.sh BINARY OUTPUT_DIR ACTIONS_RECEIPT}
output_dir=${2:?usage: conformance-v3.sh BINARY OUTPUT_DIR ACTIONS_RECEIPT}
actions_receipt=${3:?usage: conformance-v3.sh BINARY OUTPUT_DIR ACTIONS_RECEIPT}
"$binary" conformance-v3 \
  --meta .gooo/incremental-conformance-planner-v3.gooo \
  --contract contracts/denominator-v3.json \
  --cases-root . \
  --actions-receipt "$actions_receipt" \
  --out "$output_dir"
