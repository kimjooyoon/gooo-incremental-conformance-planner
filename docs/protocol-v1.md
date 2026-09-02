# Incremental conformance planner protocol v1

## Authority

`.gooo/incremental-conformance-planner.gooo` is the sole authority for the
following declarations:

`ValidationUnit`, `SemanticDependency`, `CacheIdentity`, `ReuseRule`, and
`ExecutionPlan`. The file also owns exactly 12 meta activities, 12 proof
cells, 12 indicator cells, and the optional-input boundary.

The Go package rejects a `.gooo` file that changes the precedence, omits one
of the authority declarations, changes the activity cardinality, or changes
the 4/4/4 cell partitions.

## UNKNOWN and FIXED_POINT authority

The `.gooo` source owns the `unknown_class` enum. The four fixed UNKNOWN cases
declare `MISSING_IDENTITY`, `MISSING_CACHE_RECEIPT`, `STALE_TOOLCHAIN`, or
`UNMATCHED_BEFORE_AFTER_IDENTITY`; the parser validates those literals and the
evaluator carries the class into UNKNOWN evidence.

`FIXED_POINT` is not a fourth top-level decision. It is accepted only for an
explicitly declared case owned by `.gooo`. An UNKNOWN top decision is
fail-closed and is never promoted to `FIXED_POINT`. A malformed or implicit
fixed-point counterexample is `REFUTED` and remains preserved evidence.

## Unit decision rule

The planner evaluates each unit independently, then resolves the plan state
with `REFUTED > UNKNOWN > CLOSED`:

| Evidence | Action | Reuse allowed |
|---|---|---:|
| Complete current identity, immutable PASS, exact receipt, outside impact closure | `REUSE` | yes |
| Complete identity differs from receipt, or unit is in semantic impact closure | `EXECUTE` | no |
| Missing identity field, missing receipt, incomplete receipt, or no matched pair | `UNKNOWN` | no |
| Graph contradiction, FAILED result, or COUNTEREXAMPLE | `REFUTED` | no |

The `REFUTED` and `UNKNOWN` states are evidence states, not a license to
rewrite the evidence as `CLOSED`. A failed execution is labeled
`OPERATIONAL_REFUTED`, retained in the evidence directory, and uploaded by
Actions even when the workflow fails.

## Identity

The six required fields are exact strings. Identity is scoped to the
validation unit's source/semantic slice, which permits an unrelated semantic
branch to remain reusable while a changed branch executes. The source slice
must not be widened merely to manufacture a cache hit.

## Indicators

The indicator vector has exactly four metrics: `build_ms`, `test_ms`,
`wall_ms`, and `peak_rss_kib`. Each indicator carries its own before value,
after value, signed delta, same-identity boolean, and state. If either side is
missing or the two complete identities differ, the state is `UNKNOWN` and no
speed or memory improvement claim is made.

## Optional external release

The optional slicer input is valid only with release `v0.1.1` and a non-empty
`sha256:` digest. It is consumed by reference, never copied into this
repository, and never used as a required cross-project gate.
