---
title: WriteSessionFile ignores syscall.Open error and operates on an invalid fd
type: bug
priority: high
status: resolved
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger-core/recorder/recorder.go` around line 219, the lock file
is opened with:

```go
fd, _ := syscall.Open(lockfile, syscall.O_RDONLY, 0000)
```

The error return is discarded with `_`. If the open fails (e.g. `/tmp` is not
writable, or the lock file was deleted between the stat check and the open), `fd`
is `-1` on Linux (or could otherwise be a meaningless value). The subsequent
`syscall.Flock` and `syscall.Close` calls then operate on the invalid descriptor:
with `fd == -1` they fail with `EBADF` but those errors are also ignored, and with
`fd == 0` they would silently lock stdin instead of the lock file.

The consequence is that `ioutil.WriteFile` proceeds to write the session PID file
**without holding the exclusive flock**. Two `perfmonger` processes calling
`WriteSessionFile` concurrently can both write the PID file, so one process's PID
overwrites the other's. This corrupts session tracking: `--status` reports the
wrong PID and `--kill` can kill the wrong process or fail to find the running one.

Why it is a bug: the lock is the entire point of this code path, and silently
losing it defeats mutual exclusion while presenting the appearance of success.

Suggested direction: check the error from `syscall.Open` and return/propagate it
(do not proceed to write the PID file if the lock could not be acquired). Also
check the `Flock` and `Close` return values so lock failures are surfaced rather
than swallowed.

## Triage

- Complexity: medium
- Mechanical fix: yes
- Requires user decision: no
- Notes: Check the error from syscall.Open and propagate it instead of discarding it; also check Flock and Close return values. Three call sites, but intended behavior (fail if lock acquisition fails) is clear with no design ambiguity.

## Resolution

Fixed in `core/cmd/perfmonger-core/recorder/recorder.go`.

- `WriteSessionFile()` now returns `(string, error)` instead of `string`. The
  `syscall.Open` error is checked and propagated (wrapped with `fmt.Errorf`)
  instead of being discarded; the function no longer writes the session PID
  file when the lock cannot be opened. The `syscall.Flock(LOCK_EX)`,
  `syscall.Flock(LOCK_UN)`, and `syscall.Close` return values are now also
  checked and surfaced as errors (with the fd closed on the lock/unlock
  failure paths).
- The sole caller, `RunDirect()` (background mode), now checks the returned
  error and `panic`s on failure, consistent with how the surrounding code in
  `RunDirect` reports fatal errors. `RemoveSessionFile()` was left unchanged;
  it already checked the open error.

Tests added in `core/cmd/perfmonger-core/recorder/recorder_test.go`:

- `TestWriteSessionFileSuccess` — verifies the happy path writes the current
  PID and returns no error under a writable TMPDIR.
- `TestWriteSessionFileLockFailureDoesNotWritePID` — regression test: with a
  writable TMPDIR but an unreadable (mode 0000) lock file, `WriteSessionFile`
  must return an error and must NOT write the session PID file. (Skipped when
  running as root, where file permissions are not enforced.)

Verification (all passing):

- `cd core/internal/perfmonger && go test -v -cover` — PASS (coverage 54.2%).
- `cd core && go test ./cmd/...` — PASS (recorder package now has tests).
- `cd core/internal/perfmonger && go vet ...` — clean (exit 0).
- `uv run pytest -v` — 41 passed (incl. background session tests). Note: the
  pre-existing `.venv` had a stale interpreter shebang and had to be recreated
  with `uv venv && uv pip install pytest pytest-timeout`.
