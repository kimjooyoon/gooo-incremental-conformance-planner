# Incremental conformance planner protocol v3

V3 is an append-only extension of the released v2 dossier. The authoritative
policy and activity declarations live in
`.gooo/incremental-conformance-planner-v3.gooo`; the fixed denominator is
`contracts/denominator-v3.json`.

## Deterministic selection

The evaluator processes activities in contract ordinal order. An activity is
`REUSED_CLOSED` only when its exact test identity, conformance identity,
scenario identity, input/toolchain/semantic-IR digests, preserved v2 digest
identity, complete prior run identity, and independent immutable `PASS`
evidence all match. A semantic impact or identity change is
`REQUIRED_RUN` and `must_execute=true`. Missing identity, provenance, prior run
identity, or current result is `UNKNOWN`. A cache hit or miss is only an
observation. Forged/stale receipts, self-approval, failed receipts, graph
contradictions, and skipped impacted activities are `REFUTED`.

The status precedence is `REFUTED > UNKNOWN > CLOSED`. The v3 evaluator has a
regression assertion that the released v2 exact-reuse semantics still produce
12 `REUSED_CLOSED` activities and a `CLOSED` case.

## Observations and deltas

Actions records `build_ms`, `test_ms`, `wall_ms`, `peak_rss_kib`, `cache_hit`,
and `cache_miss` as observed values. Missing values are JSON `null`, and the
associated state is `UNKNOWN`. A per-indicator integer delta is emitted only
when before and after have the same exact test/conformance/scenario identity,
input/toolchain/semantic-IR identity, preserved v2 digests, complete prior
run provenance, and both metric values. No aggregate score, weighted score,
expected time, or estimated saving is part of the contract or output.

Every UNKNOWN record has exactly these six fields:
`stage`, `step`, `reason`, `unknown_class`, `next_operation`, and `blocked_by`.
`FIXED_POINT` is accepted only when explicitly declared by the `.gooo` case
authority; it is never used as a top-level decision.

All generated files are written to the caller-owned Actions temporary output.
The exact inventory excludes the root `README.md`, `.git`, generated output,
caches, vendor, and toolchain directories. Failed Actions runs remain uploaded
as `OPERATIONAL_REFUTED` evidence.
