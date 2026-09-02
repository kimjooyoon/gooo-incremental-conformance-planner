#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "${CI:-false}" != "true" || "${GITHUB_ACTIONS:-false}" != "true" ]]; then
  echo "immutable release policy is Actions-only" >&2
  exit 1
fi

policy="${1:-.github/immutable-release-policy.json}"
test -f "$policy"

jq -e '
  .schema == "gooo/incremental-conformance-planner/immutable-release-policy/v1"
  and .active == true
  and .activation_count == 1
  and .repository_setting.enabled == true
  and .preceding_release.status == "OPERATIONAL_REFUTED_PRESERVED"
  and .preceding_release.platform_immutable == false
  and .preceding_release.tag == "v0.1.0"
  and .last_immutable_release.tag == "v0.1.1"
  and .last_immutable_release.platform_immutable == true
  and .last_immutable_release.draft == false
  and .last_immutable_release.prerelease == false
  and .next_release.tag == "v0.1.2"
  and .next_release.draft_first == true
  and .next_release.platform_immutable_required == true
  and .next_release.annotated_tag_required == true
  and .next_release.single_evidence_asset == true
  and .next_release.asset_digest_required == true
  and .next_release.tag_or_asset_mutation == "FORBIDDEN"
' "$policy" >/dev/null
