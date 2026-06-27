---
title: stat command discards the child command's exit code and always exits 0
type: bug
priority: high
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger/stat.go` around lines 167-175, when the user's tracked
command fails (`cmdErr != nil` at line 167), the error is printed to stderr but
`run()` returns `nil` at line 175. Cobra propagates `nil` back to `main`, so the
`perfmonger stat -- <command>` process always exits with code 0 regardless of the
child's exit status.

Any script or CI job that runs `perfmonger stat -- ./my-test` and checks `$?` to
detect a failing child command will never see the failure, making `perfmonger stat`
unusable as a wrapper in automated pipelines.

Why it is a bug: a process wrapper must propagate the wrapped command's exit code;
swallowing it silently breaks the primary scripted use case.

Suggested direction: capture the child's exit code (e.g. from
`*exec.ExitError`) and return `cmdErr` (or an error carrying the child's exit
status) from `run()` after displaying the summary, so cobra/`main` exit with the
child's status. Print the summary first, then propagate the failure.

## Triage

- Complexity: medium
- Mechanical fix: no
- Requires user decision: yes
- Notes: stat discards the child command's exit code and always exits 0. Capturing *exec.ExitError is mechanical, but the behavior (show summary then propagate; exact code vs 1; priority vs recorder errors) needs confirmation.

## Implementation Notes

Capture the child's exit status from *exec.ExitError after userCmd.Run() and propagate it. Decision points: (a) should the summary always print even when the command failed (ticket suggests yes); (b) preserve the exact exit code vs accept 1 (cobra collapses any non-nil error to exit 1 unless os.Exit is used directly, which skips deferred cleanup); (c) when both the command and the recorder (#0030) fail, which error wins. Sketch: extract ExitCode(), print summary, then return an error (or os.Exit with the captured code).
