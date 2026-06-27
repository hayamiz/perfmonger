---
title: player_stdin pipe fd leaked when StdoutPipe fails in recorder setup
type: bug
priority: medium
status: resolved
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger-core/recorder/recorder.go` around lines 307-312, when
`option.PlayerBin` is set, `player_stdin` is obtained from
`player_cmd.StdinPipe()` (around line 300). If the subsequent
`player_cmd.StdoutPipe()` call (around line 307) fails, the code sets
`player_stdin = nil` (around line 311) without first calling
`player_stdin.Close()`. The underlying write-end pipe file descriptor is
abandoned.

Because `player_cmd.Start()` is never reached on this error path, the player
subprocess is never started and the dangling pipe fd stays open for the lifetime of
the `perfmonger` process. On a path that is hit repeatedly (e.g. live monitoring
with frequent restarts), the process fd table can eventually be exhausted.

Why it is a bug: an error-handling path leaks an OS file descriptor.

Suggested direction: in the `StdoutPipe()` failure branch, call
`player_stdin.Close()` before setting it to nil (and before returning/falling
back), so the pipe is released.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: When player_cmd.StdoutPipe() fails, call player_stdin.Close() before setting it to nil. Unambiguous fd-leak fix on an existing error path.

## Resolution

Extracted the leak-prone inline pipe-setup logic in `RunDirect` into a testable
seam, `setupPlayerPipes(cmd playerPipeSource)` in
`core/cmd/perfmonger-core/recorder/recorder.go`. The function now calls
`stdin.Close()` before abandoning the stdin pipe when `StdoutPipe()` fails,
releasing the previously-leaked write-end pipe fd. `*exec.Cmd` is adapted to the
new `playerPipeSource` interface via `execCmdPipeSource`, and `RunDirect` was
rewired to call `setupPlayerPipes` so the fix lives on the production path.

Regression test added (TDD):
`TestSetupPlayerPipesClosesStdinWhenStdoutPipeFails` in `recorder_test.go`. It
injects a fake `playerPipeSource` whose `StdoutPipe()` fails and asserts the
stdin pipe's `Close()` was invoked. Confirmed RED before the fix
("stdin pipe was not closed when StdoutPipe failed; the pipe fd is leaked"),
GREEN after.

Verification: `go test -count=1 ./cmd/perfmonger-core/recorder/` passes; the
unified binary builds successfully.
