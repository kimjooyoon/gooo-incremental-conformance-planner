# Gooo incremental conformance planner

This repository contains a language-level planner for validation work whose
build and test cost grows as a Gooo semantic graph evolves. The authoritative
model is the `.gooo` declaration at
`.gooo/incremental-conformance-planner.gooo`. Go supplies parsing, digest
comparison, graph closure, planning, and evidence rendering; it does not
override the `.gooo` model.

For each `ValidationUnit`, the planner emits one row with the action
`EXECUTE`, `REUSE`, `UNKNOWN`, or `REFUTED` and the booleans `planned`,
`executed`, `reused`, `unknown`, and `refuted`. It also carries exact
`build_ms`, `test_ms`, `wall_ms`, and `peak_rss_kib` values. Missing values are
`UNKNOWN`, never zero. The report emits no aggregate score or percentage.

## v2 evidence dossier

The append-only v2 contract at `contracts/denominator-v2.json` extends the
released v1 denominator without changing it. Its authoritative source is
`.gooo/incremental-conformance-planner-v2.gooo`; Actions materializes that
source as semantic IR, a generated evaluator artifact, per-activity vectors,
and a human CI dossier.

The v2 dossier distinguishes `REUSED_CLOSED` (exact six-digest identity plus
complete activity provenance and an independent immutable PASS receipt) from
`REQUIRED_RUN` (semantic impact or identity change requires current
reverification). Missing identity/provenance is `UNKNOWN`; forged or stale
receipts, evaluator self-approval, dependency/proof contradictions, and
skipped impacted activities are `REFUTED`. Cache hits and misses are reported
as observations, never as proof or a success metric.

The human dossier includes the exact denominator and counts for total
activities, required runs, reused closed, unknown, refuted, executed, and
skipped-with-proof. Actual Actions receipts include integer build/test/wall/RSS
measurements, cache observations, and build/test activity durations. Missing
metrics remain `null` with `UNKNOWN`; no score, percentage, or estimated time
is emitted. See [docs/protocol-v2.md](docs/protocol-v2.md).

## v3 deterministic validation selection

The append-only v3 contract at `contracts/denominator-v3.json` is owned by
`.gooo/incremental-conformance-planner-v3.gooo`. It keeps the same 12 activity
denominator and the independent proof buckets `FOUNDATION / COHERENCE /
REGRESSION = 4 / 4 / 4` and indicator buckets `DRIVER / OUTCOME / GUARDRAIL =
4 / 4 / 4`.

Each v3 activity records exact `test_identity`, `conformance_identity`,
`input_digest`, `toolchain_digest`, `semantic_ir_digest`, prior run identity,
and observed `build_ms`, `test_ms`, `wall_ms`, `peak_rss_kib`, `cache_hit`, and
`cache_miss`. The deterministic selection emits reusable evidence and the
activities that must execute separately. Semantic impact or identity change
selects a required run; missing provenance remains UNKNOWN; a cache observation
never closes an activity by itself. The v3 suite also asserts that the v2
exact-reuse behavior remains closed.

Per-indicator integer deltas are emitted only for an exact matched before/after
pair. Missing measurements or non-matching identity produce JSON `null` and
`UNKNOWN`. V3 emits no total score, weighted score, expected time, or estimated
saving. UNKNOWN evidence uses exactly the six fields `stage`, `step`, `reason`,
`unknown_class`, `next_operation`, and `blocked_by`.
See [docs/protocol-v3.md](docs/protocol-v3.md) and
[docs/generated-artifacts-v3.json](docs/generated-artifacts-v3.json).

## Reuse boundary

Reuse requires an immutable `PASS` receipt and exact equality of all six
`CacheIdentity` fields:

```text
source_digest
semantic_ir_digest
fixture_digest
contract_digest
go_toolchain_digest
command_descriptor_digest
```

If a field is missing, the unit is `UNKNOWN`. If a complete field differs,
reuse is forbidden and the unit is planned for `EXECUTE`. A `FAILED` result or
`COUNTEREXAMPLE` is `REFUTED`; it is preserved and cannot be used to turn a
candidate into `CLOSED`.

Semantic dependencies are traversed forward from changed nodes. A unit whose
semantic nodes are in that closure executes even when an older receipt exists.
An unrelated semantic branch can remain reusable only when its own six-field
identity still matches exactly.

The fixed conformance contract has 12 proof cells: four `CLOSED`, four
`UNKNOWN`, and four `REFUTED`. Its indicator cells are four `OBSERVED`, four
`UNKNOWN`, and four `REFUTED`. The fixtures include exact cache hit, stale
toolchain, missing fixture digest, semantic impact propagation, known failed
test, unrelated change, graph contradiction, counterexample, missing receipt,
and unmatched before/after evidence.

The `.gooo` source also owns the `unknown_class` enum and the fixed-point
rules. `FIXED_POINT` is only an explicit case assessment, never a top-level
decision. An UNKNOWN top decision remains fail-closed UNKNOWN, while malformed
or implicit fixed-point counterexamples remain REFUTED.

## Optional slicer input

An immutable, digest-pinned `gooo-semantic-impact-slicer` `v0.1.1` release may
be supplied as an optional input. The planner consumes only its release label
and digest; it does not copy the release and does not require a cross-project
gate. Omitting the input leaves the local planner usable.

## Commands

The repository deliberately does not run validation locally. GitHub Actions is
the validation authority. In a PR Action, the workflow runs formatting,
build, test, vet, shell syntax checks, JSON assertions, conformance, and the
real build/test measurements in a caller-owned temporary directory:

```text
gooo-incremental-conformance-planner conformance \
  --out "$RUNNER_TEMP/gooo-incremental-conformance-planner"
```

The Actions workflow also runs `conformance-v3` and uploads its machine
evidence, human report, exact inventory, and release-bound artifacts.

`repository_writes=0`, `local_test_executions=0`, and
`cross_project_required_gates=0` are recorded in planner evidence. A failed
run is uploaded with `OPERATIONAL_REFUTED` state and is not deleted or
rewritten.

The release workflow runs only from `main`, after a green PR merge. The
previous `v0.1.0` release was audited as `platform_immutable=false` and is
preserved as an `OPERATIONAL_REFUTED_PRESERVED` lineage record. The repository
immutable-release setting is activated once, and future release publication is
draft-first. The preceding `v0.1.1`, `v0.1.2`, and `v0.1.3` releases are immutable and
remain unchanged. The workflow now creates one annotated `v0.1.4` tag, uploads
one evidence asset, verifies the tag target and asset digest, requires the
platform immutable flag after publication, and never deletes or recreates
failed release state.

## Technical basis

The implementation uses only the official Go documentation for language and
toolchain claims:

- [Go 1.27 Release Notes](https://go.dev/doc/go1.27)
- [Go release history](https://go.dev/doc/devel/release)
- [About the go command](https://go.dev/doc/articles/go_command)
- [Command go](https://go.dev/cmd/go/)
- [Go Modules Reference](https://go.dev/ref/mod)

The inventory policy excludes the root `README.md`, `.git`, caller-owned
generated output, caches, vendor trees, and toolchain directories. Generated
artifact names and their digest policy are recorded in
`docs/generated-artifacts-v1.json`.
