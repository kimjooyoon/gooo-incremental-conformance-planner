# Release policy v1

`v0.1.0` is a draft-first, evidence-backed immutable release:

1. A PR is opened and must pass the PR workflow.
2. The green PR is merged to `main`.
3. The main workflow repeats the checks and records Actions run identity,
   commit digest, Go toolchain identity, exact measurement values, and artifact
   digests.
4. The workflow creates one annotated `v0.1.0` tag and verifies its target.
5. It creates a draft release, uploads the single evidence tarball, verifies
   the server asset digest, and publishes the draft.

The workflow uses the standard `github.token`. It does not require a shared
ledger, another repository, or an external cross-project gate. Failed runs
remain `OPERATIONAL_REFUTED` and are retained as evidence. A tag, release, or
asset is never overwritten, deleted, or recreated by this policy.

If a verification step fails after the tag and draft release exist, a later
workflow run may recover them only when it finds exactly one matching draft,
verifies the annotated tag target, and verifies the downloaded asset digest.
Recovery preserves the existing tag and asset; it does not rebuild or replace
the released evidence.
