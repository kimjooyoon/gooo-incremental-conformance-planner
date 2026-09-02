# Incremental conformance planner protocol v2

v2 is an append-only extension of the released `contracts/denominator-v1.json`.
The authoritative source is `.gooo/incremental-conformance-planner-v2.gooo`.
The execution chain is:

`.gooo` → `conformance-v2/semantic-ir.json` → `conformance-v2/generated/evaluator.json` → per-scenario machine reports → human CI dossier

Each of the 12 activities has a proof choice (`FOUNDATION`, `COHERENCE`, or
`REGRESSION`) and an indicator class (`DRIVER`, `OUTCOME`, or `GUARDRAIL`).
The contract keeps four activities in each proof choice and four in each
indicator class.

## Activity decision

`REUSED_CLOSED` means the activity is unaffected and has an exact match across
the six required digest fields—source, semantic IR, fixture, contract,
evaluator, and Go toolchain—plus complete scenario/build/test activity
identity and an independent immutable PASS receipt. `REQUIRED_RUN` means the
semantic impact closure or a known identity change requires current
reverification. A cache hit is an observation only.

Missing identity or provenance is `UNKNOWN`. A missing metric is emitted as
JSON `null` and the corresponding indicator is `UNKNOWN`, never zero. Forged
or stale receipts, evaluator self-approval, dependency/proof graph
contradictions, and skipped impacted activities are `REFUTED`.

The precedence is `REFUTED > UNKNOWN > CLOSED`. The dossier reports exact
counts for total activities, required runs, reused closed, unknown, refuted,
executed, and skipped-with-proof. It does not emit a score, percentage, or
estimated time.

## Measurement and comparison

GitHub Actions writes the caller-owned `actions-receipt.json` with integer
`build_ms`, `test_ms`, `wall_ms`, `peak_rss_kib`, cache hit/miss observations,
and per-activity build/test durations. The receipt retains an operationally
refuted state when a build or test fails.

An improvement value is present only when the same scenario, source, semantic
IR, fixture, contract, evaluator, toolchain, and activity identity pair is
present before and after. Otherwise it is `null` with `UNKNOWN` state.

The repository has zero repository-write authority, zero local test
executions, and zero required cross-project gates. All validation is performed
by GitHub Actions.
