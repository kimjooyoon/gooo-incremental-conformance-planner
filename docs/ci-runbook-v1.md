# CI runbook v1

## PR path

The pull-request workflow is the first validation path. It checks the Go
source formatting, builds with Go 1.27, runs tests and vet, checks shell syntax,
asserts JSON invariants, runs all 12 fixture cases, and measures build/test
time and peak RSS in the GitHub Actions runner. All outputs are written under
`RUNNER_TEMP`; the repository checkout is not used as an output directory.

The measurement command records separate exact integer values for build time,
test time, wall time, and peak resident memory. It also records the exit code
and uses `OPERATIONAL_REFUTED` for a failed execution. The artifact upload is
configured with `if: always()` so failure evidence remains available.

## Conformance assertions

The conformance command reads the fixed contract and verifies:

- exactly 12 cases are present;
- the fixed case vector contains four `CLOSED`, four `UNKNOWN`, and four
  `REFUTED` decisions;
- every fixture's expected action matches the planner's per-unit vector;
- exact cache hits reuse only complete identity matches;
- stale toolchain and semantic impact never reuse;
- missing fixture identity and missing receipts remain `UNKNOWN`;
- failed and counterexample evidence remains `REFUTED`;
- the optional slicer remains non-required and non-gating;
- runtime authority fields remain zero.

The evidence is a vector of rows and exact observations. It does not produce
an aggregate score or percentage.

## v2 evidence path

The same Actions job also runs `conformance-v2`. It consumes the actual
`measure/actions-receipt.json` and emits a semantic IR, generated evaluator,
four scenario dossiers, and the exact activity accounting required by the
append-only v2 contract. The scenarios cover exact reuse, semantic-impact
required reruns, missing provenance, and forged/self-approved/affected-skip
refutations. The v2 dossier is fail-closed with `REFUTED > UNKNOWN > CLOSED`.

## Main path

The release workflow is enabled only for `main` after a green PR merge. It
repeats the conformance and measurement path, creates an evidence manifest,
and calculates the asset SHA-256 digest. A pre-existing `v0.1.0` tag or release
causes a hard failure. No failed tag, release, or asset is deleted or
recreated.

## v3 selection path

The PR and main workflows also run the v3 deterministic selection contract.
The v3 machine dossier records exact test/conformance/input/toolchain/semantic
IR identities, prior run identity, the six Actions observations, per-activity
reuse or required-run selection, and the six-field UNKNOWN frontier. The
normal, UNKNOWN, and REFUTED cases are retained in the uploaded artifact. The
v3 assertions require the released exact-reuse behavior to remain closed and
require per-indicator deltas to be null unless an exact matched pair is
present.
