---
title: Goroutine draining player stdout is not joined before RunDirect returns
type: bug
priority: medium
status: open
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
