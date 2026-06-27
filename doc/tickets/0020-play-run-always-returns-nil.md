---
title: play run() always returns nil, masking RunDirect failures from cobra exit-code handling
type: bug
priority: medium
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger/play.go` around lines 51-52, `playCommand.run()` calls
`player.RunDirect(cmd.PlayerOpt)` and unconditionally returns `nil`.

`RunDirect` uses `panic` for hard error conditions (file open failure, gob decode
error, etc.) rather than returning errors, and logs-and-skips for softer failures
(e.g. `showStat` errors written to stderr). Because `run()` always returns `nil`,
cobra's `RunE` mechanism never sees a non-nil error, so the process exits with
code 0 even when playback failed in a non-panicking way. Users piping the JSON
output programmatically cannot detect partial or failed playback from the exit
code.

Why it is a bug: the exit code does not reflect failure, defeating scripted/CI use.

Suggested direction: change `RunDirect` to return an `error` instead of panicking
(or have `run()` recover panics and translate them to errors), and return that
error from `run()` so cobra produces a non-zero exit code on failure. This is part
of a broader pattern across subcommands (see related tickets for `stat`, `live`,
`fingerprint`).

## Triage

- Complexity: medium
- Mechanical fix: no
- Requires user decision: yes
- Notes: play run() always returns nil, hiding failures from cobra exit codes. RunDirect panics on some errors but logs-and-skips on others, so a clean fix needs a decision on the broader error-handling strategy.

## Implementation Notes

playCommand.run() unconditionally returns nil, so cobra exits 0 even when RunDirect panics (crashes anyway) or silently logs-and-continues (e.g. showStat failures). Partial/failed playback produces wrong output with exit code 0.

Options:
- Option A (preferred long-term): make RunDirect return an error instead of panicking; collect showStat errors; update RunWithOption/Run and run() to propagate. Larger refactor but aligns with Go conventions and supports #0042.
- Option B (tactical): have run() recover panics and translate to errors. Minimal, but doesn't fix silent showStat failures.
- Option C: inspect stderr side-effects (fragile, rejected).

Decision should align with the broader error strategy and coordinate with #0042/#0044.
