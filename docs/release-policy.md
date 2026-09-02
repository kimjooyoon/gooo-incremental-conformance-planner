# Release policy v2

`v0.1.0` is retained as an evidence-backed operationally refuted predecessor.
Its GitHub release reported `immutable=false`; its release, annotated tag,
evidence asset, and source Actions evidence are preserved as
`OPERATIONAL_REFUTED_PRESERVED`. They are never deleted or rewritten. The
exact lineage is recorded in `.github/immutable-release-policy.json`.

The repository immutable-release setting is activated exactly once after that
audit. Future releases use this draft-first policy:

1. A PR must pass the PR workflow and be merged to `main`.
2. The main workflow enforces the recorded repository immutable-release policy.
3. It creates one annotated `v0.1.3` tag and a draft release. The preceding
   `v0.1.2` release is already immutable and remains unchanged.
4. It uploads one evidence asset and verifies the tag target and downloaded
   asset digest.
5. It publishes only when the resulting release reports `immutable=true`.

If a run stops after creating the tag or draft, a later run may recover that
same draft only when the existing annotated tag still points to a commit in
the current `main` history. The tag is never moved. Release verification
records both the preserved tag target and the Actions run/head that produced
the final evidence.

The workflow records Actions run identity, commit digest, Go toolchain identity,
exact measurement values, and artifact digests. A failed run remains retained
as operational evidence. A tag, release, or asset is never overwritten,
deleted, or recreated by this policy.

The technical basis remains limited to official Go documentation:
[Go 1.27 Release Notes](https://go.dev/doc/go1.27), [About the go command](https://go.dev/doc/articles/go_command), and [Go Modules Reference](https://go.dev/ref/mod).
