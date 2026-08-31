# AGENTS.md

Instructions for AI agents working on this repository.

## What this repo is

`sandbox` is a Go CLI (Cobra) that manages disposable or persistent Linux
environments via rootless Podman. Product contract for agents *using* the CLI:
see `docs/agents.md`. Documentation map: `docs/installation.md`,
`docs/usage.md`, `docs/architecture.md`, `docs/security.md`,
`docs/development.md`.

## Commands

```bash
make build                          # ./sandbox
make test                           # unit tests
go vet ./...
SANDBOX_E2E=1 make integration      # real Podman E2E (~3 min; needs network once)
SANDBOX_E2E_MATRIX=1 make integration  # opt-in distro matrix (downloads images)
gofmt -l .                          # must be empty
```

## Working conventions

- **TDD**: write the failing test first, then implement.
- **One branch per issue**, branch name describes the change, commit message
  ends with `Closes #N` when it resolves an issue. PRs merge with squash.
- **Errors must be actionable** and name the affected sandbox.
- **Names are reserved atomically** in `Manager.Create` via
  `metadata.Store.SaveExclusive`; never introduce a check-then-create race.
- **New test files per feature** rather than editing shared test files.
- `.gitignore` patterns must be anchored (`/sandbox`, never `sandbox`): an
  unanchored pattern silently excludes whole directories (this bit us once).
- Never run `sudo podman`; the CLI is rootless by design.

## Validation gates

- Pushes go through the **no-mistakes gate**: `git push no-mistakes <branch>`.
  The pipeline reviews with Codex; findings may require
  `no-mistakes axi respond --action fix|approve`. Review fixes land as extra
  commits on the branch.
- Do not commit on `main` while a pipeline run is active
  (`no-mistakes axi status`); sync with `no-mistakes axi sync` afterwards.
- The repo has no CI: pipelines report a `ci` step that waits without checks.
  That step never gates.

## Releases

1. Run the full E2E suite (`SANDBOX_E2E=1 make integration`).
2. Bump `Version` in `cmd/root.go` and align docs' version examples.
3. Push via no-mistakes, then tag SemVer (`vX.Y.Z`) on the pushed head and push
   the tag.
4. Verify `go install github.com/hocicopollo02/sandbox@vX.Y.Z`.

## Parallel work

Multiple lanes use the `enjambre` skill: treehouse-leased worktrees, one
`pi-subagents` workflowScript with stable keys and per-child `cwd`, children
report through `pi-intercom`, parent consolidates and opens one PR per lane.
