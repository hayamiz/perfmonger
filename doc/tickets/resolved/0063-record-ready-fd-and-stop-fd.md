---
title: Add --ready-fd and --stop-fd to record for C-driven start/stop
type: feature
priority: medium
status: resolved
created: 2026-08-07
updated: 2026-08-08
---

## Description

Implements the core start/stop fd contract for the C-embedding interface
(#0009). Depends on #0062 (RunDirect error return / exit-code contract).

### --ready-fd=N (ready signal)

The recorder child process writes one newline-terminated JSON object to
fd N **after the first sample is flushed** (the baseline is captured and
the measurement window is armed), then closes fd N:

    {"event":"ready","version":1,"pid":1234,"output":"/var/tmp/q42.pgr.gz","start_time":"2026-08-07T12:34:56.789012Z"}

Semantics (per #0009 D3): the emitter is the recorder, not the caller. A C
caller blocks on `read(ready_fd)` and only starts its query once it sees
the line (or EOF), closing the fork+exec race where the parent returns
before the child is sampling. When `--ready-fd` is set, suppress the
`[recording to ...]` stdout banner.

### --stop-fd=N (stop-on-fd-close)

A goroutine reads fd N and closes the existing `RecorderOption.StopCh`
when fd N reaches EOF or errors. stdin users pass `--stop-fd=0`. No pipe
auto-detection (a `/dev/null` stdin would stop immediately). On stop, the
existing graceful path runs: final sample → Flush → gzip Close →
`file.Sync()` → Close → exit 0. SIGTERM/SIGINT and `--timeout` share this
exit path.

### Constraints & wiring

- `--ready-fd`/`--stop-fd` are incompatible with `--background` (daemon
  re-exec detaches fds); using them together is a setup error (exit 1).
- Extend `RecorderOption` with `ReadyFd int` / `StopFd int` (`-1`
  disabled); add flags in `core/cmd/perfmonger/record.go`.
- Replace the `stat` subcommand's `time.Sleep(100ms) // Give recorder a
  moment to start` with the new ready signal.

### Acceptance

- ready JSON is emitted after the first flush and fd is closed.
- closing the stop fd stops recording cleanly with exit 0.
- `--ready-fd`/`--stop-fd` + `--background` exits 1.
- `stat` no longer sleeps 100 ms to wait for the recorder.
- Covered by pytest pipe-fixture tests (see #0065).

## Triage

- Complexity: high
- Mechanical fix: yes
- Requires user decision: no
- Fix strategy: worktree (non-trivial feature across recorder + CLI).
- Affected files: core/cmd/perfmonger-core/recorder/recorder.go,
  core/cmd/perfmonger/record.go, core/cmd/perfmonger/stat.go, and tests.
- Notes: Design settled in #0009 (D1/D2/D3/D6). #0062 is now landed, so
  the error-return contract (SetupError/CollectionError, exit 1/2) is
  available to report an invalid fd / --background conflict as exit 1.
  Touch points confirmed: RecorderOption at recorder.go:37-51
  (add ReadyFd/StopFd next to StopCh), flag defs ~recorder.go:85-97,
  record.go for CLI wiring and the --background exclusivity check, and
  stat.go:144-145 to drop the 100ms sleep. stop-fd reuses the existing
  StopCh select (recorder.go:571).

## Implementation Notes

Plan and acceptance criteria are in the Description above and the "CLI
contract" section of #0009. ready fires only after the first sample is
flushed (D3). Verify with pytest pipe fixtures (delivered in #0065).

## Resolution

Fixed on branch `ticket/0063-record-ready-fd-and-stop-fd`
(commit `7b48d73`), not yet landed on `master`.

What changed:
- `RecorderOption` gains `ReadyFd int` / `StopFd int` (default `-1`).
  Flags `--ready-fd` / `--stop-fd` added in `record.go`.
- `--ready-fd`: after the first sample is flushed, the recorder writes one
  JSON line `{"event":"ready","version":1,"pid":..,"output":..,
  "start_time":..}` to the fd and closes it; the `[recording to ...]`
  banner is suppressed when set. fd validated with `fcntl` (F_GETFD +
  writability); invalid → `*SetupError` (exit 1).
- `--stop-fd`: a goroutine reads the fd and closes `StopCh` on EOF/error
  (double-close guarded by `sync.Once`); clean stop → exit 0.
- `--ready-fd`/`--stop-fd` incompatible with `--background` → exit 1.
- `stat`'s fixed `time.Sleep(100ms)` replaced by an internal ready-pipe
  wait (falls back to the sleep only if `os.Pipe()` itself fails);
  observable behavior unchanged.

Tests: 3 Go unit tests (ready JSON emission, stop-fd stop, background
incompatibility) + `tests/test_embedding.py` (os.pipe fixtures, ready
JSON schema, stop-fd close → exit 0 → playable output).

Evaluator: PASS — Sonnet evaluator (Codex CLI unavailable: 401
Unauthorized). Ran recorder unit tests (incl. `-race`), `go test
./cmd/...`, `go vet`, `make build`, full pytest 42/42, and exercised the
contract on the built binary via os.pipe/pass_fds: ready line
content/timing/EOF, banner suppression, `--background` conflict → 1,
invalid/read-only fd → 1 with no file created, `stat` still works with no
fixed sleep. Verified the new test fails on base. No panics.

human-review: recommended (adds user-facing CLI flags and touches raw
fd/syscall handling — a systems-level surface).

Review-state: ready-to-apply

Approved by human review 2026-08-08. Reviewer note: `--stop-fd` does not
validate read-direction (passing a pipe's write end stops immediately);
spec-compliant but worth a caller-facing caveat + a reject-on-wrong-end
check in #0065.
