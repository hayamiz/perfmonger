---
title: Refactor recorder RunDirect to return error and fsync before close
type: refactor
priority: medium
status: resolved
created: 2026-08-07
updated: 2026-08-07
---

## Description

Prerequisite work for the C-embedding interface (#0009). Today the
recorder's `RunDirect` returns no error and `panic`s on failures (e.g.
`os.Create` failing to open the output path). cobra-driven errors exit 1
and panics exit 2 only incidentally, so there is no deliberate exit-code
contract a caller can rely on.

This ticket refactors `RunDirect` (and the `RunWithOption` wrapper) in
`core/cmd/perfmonger-core/recorder/` to:

1. Return `error` instead of `panic`ing on setup and collection failures.
2. Call `file.Sync()` on the output file before `Close`, so a successful
   exit guarantees the data is on disk (deterministic flush/exit).
3. Propagate errors to the `record` subcommand so it can map them onto
   the exit-code contract defined in #0009:
   - `0` normal, `1` setup error (nothing written), `2` collection/write
     error (output possibly incomplete).

This is the foundation for #0063 (`--ready-fd`/`--stop-fd`) and #0064
(`--meta`); those tickets assume `RunDirect` can report failure cleanly.
Removing the panics also lets other subcommands drop panic-based exits
over time, but that broader cleanup is out of scope here.

See #0009 "Prerequisite refactor" and "Exit codes" sections for the
agreed contract.

### Acceptance

- `RunDirect` returns `error`; no `panic` on `os.Create`/write failures.
- `file.Sync()` is invoked before the output file is closed on the normal
  path.
- `record` exits 0/1/2 per the contract.
- Existing Go unit tests and pytest integration tests still pass.

## Triage

- Complexity: medium
- Mechanical fix: yes
- Requires user decision: no
- Fix strategy: worktree (non-trivial refactor of the recorder package).
- Affected files: core/cmd/perfmonger-core/recorder/recorder.go,
  core/cmd/perfmonger/record.go, and recorder tests.
- Notes: Design fully settled in #0009. No dependencies — this is the
  prerequisite for #0063/#0064 and can start immediately. Care needed:
  RunWithOption/RunDirect currently panic in ~9 sites (recorder.go:221,
  310-316, 384, 458, 485-490, 602-641); convert setup vs collection
  failures to distinct error paths so record.go can map exit 1 vs 2. Add
  file.Sync() before Close on the normal path. Keep the deferred
  panic-recovery in RunWithOption (recorder.go:310) or replace it with
  error return consistently.

## Implementation Notes

Plan and acceptance criteria are in the Description above and the
"Prerequisite refactor" / "Exit codes" sections of #0009. Verify with the
commands in doc/tickets/CLAUDE.md (go test ./cmd/..., pytest).

## Resolution

Fixed on branch `ticket/0062-rundirect-error-return-and-fsync`
(commit `d4375b4`), not yet landed on `master`.

What changed:
- `RunDirect`/`RunWithOption` now return `error` instead of panicking.
  Added `SetupError` (pre-write failures) and `CollectionError`
  (post-open failures) wrapper types in
  `core/cmd/perfmonger-core/recorder/recorder.go`.
- `os.Create`, header/record encode, and flush panics converted to error
  returns. `newGzipBufWriter` now also returns the gzip writer so the
  trailer is closed explicitly before fsync (cleanup kept as a
  panic-safety net).
- `file.Sync()` is called (via a `syncFile` helper, regular files only)
  after the gzip trailer and before Close on the normal path; stdout/`-`
  and non-regular outputs are skipped.
- `core/cmd/perfmonger/record.go` classifies the returned error with
  `errors.As`: `SetupError` → exit 1, `CollectionError` → exit 2,
  normal/stop → exit 0. `live.go`/`stat.go` updated to propagate the new
  error (no observable behavior change).

Tests added (`recorder_test.go`):
`TestRunDirectSetupErrorOnUncreatableOutput`,
`TestRunDirectNormalPathReturnsNil`,
`TestRecorderErrorTypesAreDistinct`; existing panic-flush test kept.

Evaluator: PASS — Sonnet evaluator (Codex CLI unavailable: 401
Unauthorized). Ran the full verification suite (recorder unit tests
12/12, `go test ./cmd/...`, `go vet`, build) plus full pytest 41/41, and
exercised the exit-code contract on the built binary: normal→0,
setup error (uncreatable path)→1 with no panic, SIGINT/SIGTERM→0 with a
playable file, `/dev/full`→2 (real ENOSPC). Confirmed fsync via strace on
the success path, verified new tests fail against master's code, and
confirmed the branch touches only source + tests.

human-review: recommended (changes user-visible exit-code / error
behavior — previously any recorder failure panicked with a stack trace).

Review-state: ready-to-apply

Approved by human review 2026-08-07.
