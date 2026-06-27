---
title: Goroutine draining player stdout is not joined before RunDirect returns
type: bug
priority: medium
status: resolved
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger-core/recorder/recorder.go` around lines 326-343, a
goroutine (started near line 328) reads from `player_stdout` and writes to
`os.Stdout`. `RunDirect` waits for the player **process** to exit via
`player_cmd.Wait()` (around line 486), but it does not wait for this goroutine to
finish draining the pipe.

After `Wait()` returns, the goroutine may still be in a `Read` call or in the
middle of writing to `os.Stdout`. In the `live` subcommand this means the process
can exit (cobra returns, `main` exits) while the goroutine is still writing,
resulting in truncated JSON lines, or a data race on `os.Stdout` if anything else
touches it during shutdown.

Why it is a bug: output can be truncated and there is an unsynchronized concurrent
writer to `os.Stdout` at shutdown.

Suggested direction: use a `sync.WaitGroup` (or a done channel) for the draining
goroutine and wait on it after `player_cmd.Wait()` returns, before `RunDirect`
returns, so all buffered player output is flushed to stdout.

## Triage

- Complexity: medium
- Mechanical fix: yes
- Requires user decision: no
- Notes: The goroutine draining player stdout is not joined before RunDirect returns. Use a sync.WaitGroup (Add before spawning, Done at goroutine end, Wait after player_cmd.Wait()). Standard synchronization pattern, no design ambiguity.

## Resolution

Fixed in `core/cmd/perfmonger-core/recorder/recorder.go`.

- Extracted the inline stdout-draining goroutine into a testable seam,
  `startPlayerDrain(r io.Reader, w io.Writer) *sync.WaitGroup`, which calls
  `wg.Add(1)` before spawning the goroutine and `defer wg.Done()` at the
  goroutine's end.
- `RunDirect` now stores the returned `*sync.WaitGroup` in `player_drain_wg`
  and calls `player_drain_wg.Wait()` after `player_cmd.Wait()`, so all buffered
  player output is flushed to `os.Stdout` before `RunDirect` returns.
- While extracting, reordered the loop so a final read returning `n > 0` with
  `io.EOF` writes its bytes before breaking (the original broke on EOF before
  writing, contributing to truncation).

Test added (TDD): `TestStartPlayerDrainJoinsBeforeReturn` in
`recorder_test.go`. It drives `startPlayerDrain` with a `slowReader` that emits
one byte at a time with delays, then asserts that after `wg.Wait()` the full
payload reached the destination writer. The test first failed RED with
`undefined: startPlayerDrain`, then passed after the fix. Verified with
`go test -count=1` and `go test -race -count=1` on the recorder package, and the
unified binary builds clean.
