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

`repository_writes=0`, `local_test_executions=0`, and
`cross_project_required_gates=0` are recorded in planner evidence. A failed
run is uploaded with `OPERATIONAL_REFUTED` state and is not deleted or
rewritten.

The release workflow runs only from `main`, after a green PR merge. It creates
one annotated `v0.1.0` tag, creates the release as a draft, uploads one evidence
asset, verifies the tag target and asset digest, and then publishes it. It
fails if the tag or release already exists; it never deletes or recreates
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
