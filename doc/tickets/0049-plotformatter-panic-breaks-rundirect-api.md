---
title: panic() used throughout runPlotFormat breaks the RunDirect library API
type: bug
priority: high
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger-core/plotformatter/plotformatter.go`, `RunDirect` (around
line 243) is a public Go API that promises to return `(*PlotMeta, error)`. It
delegates to `runPlotFormat`, which uses `panic()` in at least ten locations — for
example line 324 (cpu file creation failure), line 347 (gob decode error), line
449 (disk file creation failure), and lines 334, 355, 362, 372, 413, 457, 462.

When `RunDirect` is called from other Go code, any of these panics propagates as an
unrecoverable panic to the caller instead of being returned as an error value. The
caller cannot distinguish a recoverable I/O error from a genuine programming bug,
and the panic bypasses all normal error handling and logging.

Why it is a bug: a public API documented to return `error` instead crashes its
caller on ordinary I/O and decode failures.

Suggested direction: replace the `panic()` calls in `runPlotFormat` with `return
nil, err` (wrapping with context), so all recoverable failures are returned through
the `(*PlotMeta, error)` contract. Reserve panics, if any, for true invariant
violations.

## Triage

- Complexity: medium
- Mechanical fix: no
- Requires user decision: no
- Notes: runPlotFormat uses panic() in ~10 places but RunDirect is a public API returning (*PlotMeta, error). Conversion is mechanical but systematic across the function; flagged no because it's a larger coordinated refactor (basis for #0050/#0052).

## Implementation Notes

Replace all panic(err)/panic(msg) calls in runPlotFormat with `return nil, fmt.Errorf(...)` wrapping the original error (%w) plus context (os.Create CPU/Mem/Disk files, gob decode, GetDiskUsage1, missing disk device, GetCpuUsage, missing disk_dat, ReadFile of temp files). Verify: malformed log → error not panic; read-only output dir → error; plot command exits non-zero and reports the error. Coordinates with #0050 (cleanup on error path) and #0052 (flush errors).
