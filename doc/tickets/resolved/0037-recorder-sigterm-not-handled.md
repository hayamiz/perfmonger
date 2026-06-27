---
title: SIGTERM kills the recorder without flushing bufio or closing the gzip writer
type: bug
priority: high
status: resolved
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger-core/recorder/recorder.go`, `signal.Notify` at line 408
registers only `os.Interrupt` (SIGINT). SIGTERM is not intercepted, so the Go
runtime terminates the process on SIGTERM without running any deferred functions.

Consequently the `bufio.Writer`'s in-memory buffer (around line 366) is never
flushed into the `gzip.Writer`, and `gzip.Writer.Close()` (deferred near line 364)
is never called, so the gzip trailer (CRC32 checksum + size) is never written. The
resulting `.pgr.gz` file is unreadable by any gzip reader.

SIGTERM is the common termination signal sent by systemd, container runtimes, and
a plain `kill <pid>`, so this is a frequent real-world condition.

Why it is a bug: ordinary process termination corrupts the output file.

Suggested direction: include `syscall.SIGTERM` in the `signal.Notify` call and
handle it on the same graceful-shutdown path as SIGINT, so the buffer is flushed
and the gzip writer is closed before exit. Pair this with the `signal.Stop`
cleanup tracked separately.

## Triage

- Complexity: medium
- Mechanical fix: yes
- Requires user decision: no
- Notes: Add syscall.SIGTERM to the existing signal.Notify call and handle it the same as SIGINT (set running=false for graceful exit). Pairs naturally with #0014 (defer signal.Stop). No design ambiguity.

## Resolution

Fixed in `core/cmd/perfmonger-core/recorder/recorder.go`: the `signalNotify`
call (built on the #0014 test seam) now registers `syscall.SIGTERM` alongside
`os.Interrupt`. Both signals are delivered on the same `sigint_ch` channel, whose
`select` case sets `running = false`, so SIGTERM now follows the identical
graceful-shutdown path as SIGINT — the recording loop breaks, the bufio buffer is
flushed, and the deferred `gzip.Writer.Close()` runs, writing a valid gzip
trailer instead of corrupting the output file.

Test added (TDD): `TestRunDirectHandlesSIGTERM` in `recorder_test.go` installs
the observable `signalNotify` seam, runs `RunDirect`, and asserts that both
`os.Interrupt` and `syscall.SIGTERM` are passed to `signal.Notify`.

RED (before fix):

```
--- FAIL: TestRunDirectHandlesSIGTERM (0.01s)
    recorder_test.go:232: signal.Notify was not called with syscall.SIGTERM; SIGTERM is not handled gracefully (got [interrupt])
FAIL
```

GREEN (after fix): full recorder package test suite passes
(`go test -count=1 ./cmd/perfmonger-core/recorder/` → ok), and the unified
binary builds successfully.
