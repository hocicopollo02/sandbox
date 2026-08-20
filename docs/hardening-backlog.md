# Hardening backlog

Scope: extend V1 E2E coverage without changing the rootless Podman isolation model.

## P0 — persistent data and lifecycle

### Keep an isolated home on delete

**Scenario**: create a persistent sandbox, then delete it with `--keep-home`.

**Acceptance criteria**:

- the container and metadata are removed;
- the managed home remains unchanged;
- deleting the same metadata cannot remove that home accidentally;
- the CLI output makes the retained home explicit.

### Re-enter after stop

**Scenario**: create a persistent sandbox, stop it, enter it again, and exit.

**Acceptance criteria**:

- `enter` starts the stopped container;
- the shell is usable;
- `list --json` reports `running` afterwards;
- delete removes the container, metadata, and home.

## P1 — failure and UX boundaries

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

## P2 — concurrency

### Same-name create race

Run two `create` commands for the same valid name concurrently.

**Acceptance criteria**:

- exactly one command succeeds;
- the loser reports a collision;
- no orphan container or home is left;
- metadata describes the winner only.

### Independent creates

Run two creates with different names concurrently.

**Acceptance criteria**:

- both succeed without cross-contamination;
- both homes and metadata records are isolated;
- deleting one does not affect the other.

## Out of scope

- Adversarial same-user filesystem TOCTOU races; the current limitation is documented and a named Podman volume or VM is the upgrade path.
- Installing arbitrary third-party tools such as Go/mise; that validates the container runtime, not the CLI lifecycle contract.
