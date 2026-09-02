#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "${CI:-}" != "true" || "${GITHUB_ACTIONS:-}" != "true" ]]; then
  echo "CI-only v2 conformance: refusing local execution" >&2
  exit 2
fi

binary=${1:?usage: conformance-v2.sh BINARY OUTPUT_DIR ACTIONS_RECEIPT}
output_dir=${2:?usage: conformance-v2.sh BINARY OUTPUT_DIR ACTIONS_RECEIPT}
actions_receipt=${3:?usage: conformance-v2.sh BINARY OUTPUT_DIR ACTIONS_RECEIPT}
"$binary" conformance-v2 \
  --meta .gooo/incremental-conformance-planner-v2.gooo \
  --contract contracts/denominator-v2.json \
  --cases-root . \
  --actions-receipt "$actions_receipt" \
  --out "$output_dir"
