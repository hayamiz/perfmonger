---
title: plotformatter temporary files are never deleted when a panic occurs mid-processing
type: bug
priority: high
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger-core/plotformatter/plotformatter.go`, per-device disk and
CPU temporary files are created via `ioutil.TempFile` in `makeDiskDatTmpFile`
(around line 201) and `makeCpuDatTmpFile` (around line 230). Their deletion
(`os.Remove`) only happens in the normal code path at lines 467 and 481. The
defers registered at lines 385 and 420 only close the file descriptors — they do
not delete the files.

If any panic fires before reaching lines 467/481 — e.g. a gob decode error at line
347, an I/O error at line 355, or an `os.Create` failure at line 449 — all
in-progress temp files are left on disk with no subsequent cleanup, leaking disk
space across repeated failures.

Why it is a bug: failure paths leak temp files; repeated failures can accumulate
significant disk usage.

Suggested direction: register the `os.Remove` cleanup in the same `defer` that
closes each temp file (so it runs on panic as well as normal return), or convert
panics to errors (see related ticket) and ensure deferred removal runs on every
exit path.

## Triage

- Complexity: medium
- Mechanical fix: no
- Requires user decision: no
- Notes: Temp files (ioutil.TempFile) are only removed on the normal path; any panic before cleanup leaks them, and the existing defers only Close, not Remove. Depends on #0049 (panic→error) plus extending defers to delete.

## Implementation Notes

Extend the defers registered when disk_dat/cpu_dat temp files are created to also os.Remove the paths (e.g. `cleanup := func(){ os.Remove(path); f.Close() }; defer cleanup()`), so deletion happens on every exit (panic or error return). Best done together with #0049 so all error paths run cleanup. Verify by injecting an early error and confirming no temp files remain.
