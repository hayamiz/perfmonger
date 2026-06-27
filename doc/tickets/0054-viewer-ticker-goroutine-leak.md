---
title: viewer background ticker goroutine leaks after MainLoop exits
type: bug
priority: medium
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger-core/viewer/viewer.go` around lines 25-30, a goroutine
started at line 25 loops forever with no exit condition. When the user quits
(presses `q` or Ctrl-C), `MainLoop()` returns and `defer g.Close()` runs, but this
goroutine keeps running.

Every second the goroutine calls `g.Update(layout)`, which itself spawns another
goroutine that tries to send to `g.userEvents` (capacity 20). Since `MainLoop` is
no longer reading from that channel, after roughly 20 seconds those inner
goroutines accumulate and block indefinitely on the channel send. The outer ticker
goroutine never terminates either, and it keeps the (already-closed) termbox `g`
alive via references.

For a CLI that calls `Run()` once and then exits, process exit masks the leak, but
if `Run()` is ever invoked more than once (tests, a restart loop), goroutines
accumulate without bound.

Why it is a bug: an unbounded goroutine/channel leak that becomes observable under
repeated invocation and keeps closed resources alive.

Suggested direction: give the ticker goroutine a stop channel (or context) that
`MainLoop` closes on exit, so the goroutine returns and stops calling `g.Update`
once the UI is torn down.

## Triage

- Complexity: medium
- Mechanical fix: no
- Requires user decision: yes
- Notes: The background ticker goroutine loops forever and keeps calling g.Update after MainLoop exits, leaking goroutines. Fix needs a choice of signaling mechanism (stop channel vs context vs atomic flag).

## Implementation Notes

A `go func(){ for { time.Sleep(1s); g.Update(layout) } }()` never terminates after MainLoop returns; inner goroutines block on the no-longer-read userEvents channel.

Options:
- Option A (recommended): a stopCh closed after MainLoop returns; the goroutine selects on it and returns.
- Option B: context.Context + cancel() after MainLoop; composable.
- Option C: atomic flag checked in the loop (less idiomatic, possible race).

Open questions: project preference for channels vs context; whether Run() must be reusable across multiple invocations; whether to join the goroutine explicitly or best-effort close.
