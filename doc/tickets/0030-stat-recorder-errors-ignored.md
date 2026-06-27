---
title: recorder.RunWithOption has no return value, so stat ignores recording errors
type: bug
priority: low
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger/stat.go` around line 141, `recorder.RunWithOption` (and the
underlying `RunDirect`) returns nothing. If `enc.Encode` fails mid-recording (e.g.
the disk fills), the recorder breaks out of its loop and returns normally. The
`recorderDone` channel then receives `true` (around line 139), and `stat.run()`
proceeds to display a summary computed from a partially written or empty file.

There is no way for `stat.run()` to detect that recording was incomplete, so the
summary output will silently contain wrong or zero statistics without any error
being reported to the user.

Why it is a bug: an incomplete recording yields a misleading summary presented as
if it were correct.

Suggested direction: change `RunWithOption`/`RunDirect` to return an `error`, have
the recorder report `enc.Encode`/`Flush` failures, and surface that error in
`stat.run()` (e.g. skip/annotate the summary and exit non-zero). This depends on
the same API change tracked for the other subcommands that call into the recorder.

## Triage

- Complexity: low
- Mechanical fix: no
- Requires user decision: yes
- Notes: recorder.RunWithOption has no return value so stat ignores recording errors. Depends on the #0028 API change; the UX (skip summary / annotate incomplete / exit non-zero) is a decision.

## Implementation Notes

Depends on #0028 adding an error return to RunWithOption/RunDirect. Then capture recErr in stat.run() and decide: Option A skip the summary and exit non-zero; Option B print the summary annotated as incomplete and exit non-zero; Option C print as-is but exit non-zero. Recommended Option B. Open question: expected behavior when recording fails silently (e.g. disk full).
