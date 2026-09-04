# Agent interface

`sandbox` is built to be driven by agents, scripts and CI without a TTY. This
is the machine contract: what you may rely on, and how to drive the full
lifecycle with minimal round-trips.

## The agent loop

```bash
sandbox create task-42 \
  --distro ubuntu \
  --persistent \
  --isolated-home \
  --no-enter \
  --yes \
  --json

sandbox exec task-42 -- git clone https://github.com/org/repo.git
sandbox exec task-42 -- bash -lc 'cd repo && make test'

sandbox list --json
sandbox doctor --json
sandbox stop task-42 --json
sandbox delete task-42 --yes --if-exists --json
```

## Guarantees

- `create --yes --no-enter` is fully non-interactive only when `--distro`,
  `--persistent`/`--disposable`, and `--isolated-home` are supplied explicitly.
  If any of those options are omitted, the interactive wizard runs even with
  `--yes --no-enter`. `--shared-home` is rejected immediately with an error,
  before any prompt. It fails if the name is taken, unless
  `--if-not-exists` is supplied; then an existing sandbox is a successful
  no-op. An incomplete metadata reservation remains an error with cleanup
  guidance. The name reservation is atomic, so concurrent creates cannot
  corrupt each other.
- `exec NAME -- COMMAND [ARG...]`:
  - requires no TTY;
  - passes arguments to the process without shell reinterpretation;
  - preserves stdout and stderr;
  - exits with the executed process's exit code;
  - auto-starts a stopped sandbox first;
  - fails for unknown sandboxes without creating anything.
- `stop` on an already-stopped sandbox succeeds (idempotent).
- `delete --yes` removes container, metadata and managed home unconditionally.
  `--keep-home` retains the home and prints its path. `--if-exists` makes a
  missing sandbox a successful no-op; when metadata exists, cleanup still runs.
- `list --json` emits objects with stable keys: `name`, `distro`,
  `persistence`, `home`, `status`. `status` is one of `running`, `stopped`,
  `missing`, or `unknown`. `missing` means stale metadata: the container is
  gone. `unknown` means a tracked container has a blank runtime state and must
  be treated as not usable.
- `doctor --json` emits one object with `ok` and `checks` keys. `checks` is an
  ordered array of `{name, ok, detail}` results for Podman installation,
  runtime operation, rootless mode, and user namespaces. `detail` is omitted
  when a check succeeds. The command writes only JSON to stdout and exits 1
  when `ok` is false.
- `info NAME --json` emits a single object with stable keys: `name`,
  `distribution`, `image`, `persistence`, `home_mode`, `home_path`,
  `created_at`, and `status`. `status` uses the same values and meanings as
  `list --json`.
- `create`, `stop`, and `delete` accept a command-local `--json` flag for
  successful results. Each writes exactly one JSON object to stdout and skips
  human status output. `create --json` requires `NAME`, `--distro`,
  `--persistent`, `--isolated-home`, and `--no-enter`, so it never starts the
  wizard or attaches a guest shell. `delete --json` requires `--yes`.
- `upgrade` resolves the latest stable module version with Go and installs it
  only when newer. It requires Go 1.24+, network access, and the running
  executable to be in Go's configured `GOBIN` or `GOPATH/bin`. With `--json`,
  it emits exactly one object with `name`, `current_version`,
  `latest_version`, and `result`, where `result` is `upgraded` or `unchanged`.

  ```json
  {"name":"task-42","result":"created"}
  {"name":"task-42","result":"stopped"}
  {"name":"task-42","result":"deleted","retained_home":null}
  ```

  `create.result` is `created` or `unchanged`; `stop.result` is `stopped` or
  `unchanged`; `delete.result` is `deleted` or `unchanged`. The idempotent flags
  report `unchanged` on a no-op. `delete` always includes `retained_home`; it is
  `null` unless `--keep-home` retained an isolated home, in which case it is
  the absolute managed path. The global `--error-format json` remains the
  separate stderr contract for failures.
- The container runtime is the source of truth for liveness; metadata is
  descriptive. Never parse `list` output assuming metadata implies a live
  container.
- Sandbox names match `[a-z0-9_-]+`.
- Isolation: only the managed home is mounted; the host filesystem, `$HOME`
  and Podman internals of the host are never exposed inside the sandbox.
- `NO_COLOR=1` disables color output.

## Exit codes

| Code | Meaning |
| ------ | --------- |
| `0` | success |
| `1` | operational error (validation, collision, runtime failure) |
| process exit code | `exec` propagates the executed command's exit code |

## Stability

The JSON keys above, the command surface and existing flags follow SemVer:
breaking changes require a major version. New flags are additive. This document
describes the contract; if code and this document disagree, it is a bug.

## Known limits

- Not a strong security boundary: do not run malware; use a VM for untrusted
  software.
- `--shared-home` is disabled and only exists to return a clear error.
- One sandbox per task is the intended usage; names are cheap.

## Ergonomics roadmap

See the agent ergonomics roadmap in [`PRD.md`](../PRD.md#49-interfaz-para-agentes).

Implemented (see `PRD.md` section 49 and issues #19, #20, and #22): `info NAME --json`,
`doctor --json` (exit 1 when the host is not ready), idempotent lifecycle flags
`create --if-not-exists` / `delete --if-exists`, success JSON for `create`,
`stop`, and `delete`, and `upgrade [--json]`. A stale reservation is reported
as an actionable error pointing at `delete --if-exists --yes`.

## Machine error codes

Every operational error maps to a stable code emitted as a single JSON object
on stderr with the global flag `--error-format json`:

```bash
sandbox --error-format json exec missing -- echo hi
```

```json
{"error":{"code":"E_NOT_FOUND","message":"sandbox \"missing\" does not exist"}}
```

| Code | Meaning |
| ------ | --------- |
| `E_EXISTS` | name or isolated home already taken |
| `E_NOT_FOUND` | sandbox or metadata does not exist |
| `E_INVALID_NAME` | sandbox name does not match `[a-z0-9_-]+` |
| `E_RUNTIME` | Podman unavailable or not working |
| `E_ERROR` | any other failure |

With `--error-format json` the human message is not printed; exit code remains
`1` for operational errors. `exec` still propagates the executed process's own
exit code and never wraps it in JSON. Verbose command diagnostics are suppressed
in JSON mode so stderr contains exactly one JSON object.
