# Inventory policy v1

The Actions inventory records regular files, descendant directories, Go file
count/lines, `.gooo` file count/lines, and all physical lines after excluding
the root `README.md`. It excludes `.git`, caller-owned generated output,
caches, vendor trees, and toolchain directories. The inventory is an evidence
artifact, not a semantic input and not a conformance gate across repositories.

The runtime authority vector is fixed at:

```text
repository_writes=0
local_test_executions=0
cross_project_required_gates=0
```

The actual counts and digest are produced by the PR/main Actions job and
stored in `inventory.json` under the caller-owned evidence directory.
