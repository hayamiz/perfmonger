---
title: live run() always returns nil, swallowing recorder.RunWithOption failures
type: bug
priority: medium
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger/live.go` around lines 85-86, `run()` calls
`recorder.RunWithOption(cmd.RecorderOpt)` and unconditionally returns `nil`.

`RunWithOption`/`RunDirect` panics on hard errors (file creation failure, gob
encode failure, player process start failure) but silently swallows softer
failures, such as a broken player pipe (the `enc.Encode` error path around
`core/cmd/perfmonger-core/recorder/recorder.go` lines 429-431 merely breaks the
loop). When the player fails to start (the three player variables are set to nil at
recorder.go lines 319-323), the recorder writes raw gob to stdout instead of
reporting an error, and `run()` still returns `nil` to cobra, which exits 0.

Why it is a bug: live monitoring can fail or emit unusable raw gob to a terminal
while reporting success.

Suggested direction: give `RunWithOption`/`RunDirect` an `error` return (and stop
panicking for recoverable failures), then propagate it from `run()` so cobra
yields a non-zero exit code. Also treat "player failed to start" as a hard error
rather than silently falling back to raw gob on stdout. Related to the
exit-code-propagation pattern tracked for `play`, `stat`, and `fingerprint`.

## Triage

- Complexity: medium
- Mechanical fix: no
- Requires user decision: yes
- Notes: live run() always returns nil. Adding an error return to recorder RunWithOption/RunDirect is mechanical, but handling "player failed to start" (fatal vs raw-gob fallback) is an architectural decision.

## Implementation Notes

Three parts: (1) add an error return to recorder RunWithOption/RunDirect (mechanical); (2) capture enc.Encode/flush errors instead of just breaking, and return them; (3) decide how to handle player startup failure — currently it silently nils variables and continues. Decision: should a missing/failed player be fatal, or is raw gob to stdout an acceptable fallback? Also: if the player pipe breaks mid-recording, detect and report?
Update call sites: live.run(), stat.run(), record.executeRecord() to capture and propagate the error.
