# Hardening backlog

Scope: extend V1 E2E coverage without changing the rootless Podman isolation model.

## P0 — persistent data and lifecycle

### Roll back persistent auto-enter failure

**Scenario**: create a persistent sandbox with auto-enter enabled, persist metadata, then force `enter` to fail.

**Acceptance criteria**:

- the command returns an actionable error;
- no partially created container, metadata record, or managed home remains;
- cleanup errors are preserved;
- the behavior matches the PRD rollback requirement instead of silently leaving a recoverable-looking sandbox.

### Keep an isolated home on delete

**Scenario**: create a persistent sandbox, then delete it with `--keep-home`.

**Acceptance criteria**:

- the container and metadata are removed;
- the managed home remains unchanged;
- deleting the same metadata cannot remove that home accidentally;
- the CLI output makes the retained home and its path explicit; this requires a CLI output change because the current command only prints `Sandbox deleted`.

### Re-enter after stop

**Scenario**: create a persistent sandbox, stop it, enter it again, and exit.

**Acceptance criteria**:

- `enter` starts the stopped container;
- the shell is usable;
- `list --json` reports `running` afterwards;
- delete removes the container, metadata, and home.

## P1 — failure and UX boundaries

### Create rollback after partial failure

**Scenario**: force `podman create`, `podman start`, and metadata persistence to fail after partial resource creation. Use an integration seam or deterministic fake runtime where the real Podman failure cannot be forced safely.

**Acceptance criteria**:

- no container, metadata record, or managed home remains after each failure;
- cleanup errors are preserved and actionable;
- cleanup never removes a resource belonging to another sandbox attempt.

### Cancel destructive confirmation

**Scenario**: delete a persistent sandbox without `--yes`, answer no.

**Acceptance criteria**:

- the command reports cancellation;
- the container, metadata, and home remain intact;
- no Podman delete command is issued.

### Corrupt metadata

**Scenario**: replace a sandbox JSON record with invalid JSON, then run `list` and `info`.

**Acceptance criteria**:

- both commands fail with an actionable metadata error;
- no container or home is deleted;
- the error identifies the affected sandbox.

### Delete failure and retry contract

**Scenario**: make container deletion, home deletion, or metadata deletion fail, then retry `delete`.

**Acceptance criteria**:

- the first failure identifies the resource that could not be removed;
- retries are safe and converge to a clean state;
- metadata is retained until cleanup can be retried, unless the CLI explicitly records a recoverable orphan;
- a failed delete never removes an unrelated home or container.

### Stale metadata / missing runtime object

**Scenario**: create metadata for a persistent sandbox, remove the Podman container externally, then run `list`, `info`, `enter`, `stop`, and `delete`.

**Acceptance criteria**:

- status is reported as missing or actionable, never as running;
- `enter` and `stop` fail without creating a replacement;
- `delete` removes metadata and managed home safely.

## P1 — distribution matrix

Run the persistent `--no-enter` lifecycle for `ubuntu`, `fedora`, and `debian` in addition to the existing Arch coverage.

**Acceptance criteria**:

- each supported image is pulled and starts rootless;
- `info` reports the selected distribution/image;
- cleanup leaves no container, metadata, or managed home.

Keep this matrix opt-in or serialised because it downloads multiple images.

## P0 — concurrency and ownership

### Same-name create race

Run two `create` commands for the same valid name concurrently.

**Acceptance criteria**:

- exactly one command succeeds;
- the loser reports a collision;
- no orphan container or home is left;
- metadata describes the winner only;
- the loser cannot remove or mutate the winner's home; reserve the name/home atomically or make cleanup conditional on ownership.

This is P0 because the current check-then-create flow can let the losing attempt delete the winner's home.

### Independent creates

Run two creates with different names concurrently.

**Acceptance criteria**:

- both succeed without cross-contamination;
- both homes and metadata records are isolated;
- deleting one does not affect the other.

## Out of scope

- Adversarial same-user filesystem TOCTOU races; the current limitation is documented and a named Podman volume or VM is the upgrade path.
- Installing arbitrary third-party tools such as Go/mise; that validates the container runtime, not the CLI lifecycle contract.
