---
title: plotformatter ignores all bufio.Writer.Flush() return values
type: bug
priority: medium
status: resolved
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger-core/plotformatter/plotformatter.go`,
`bufio.Writer.Flush()` is called at lines 443 (disk_dat flush), 469 (disk file
flush), 472 (cpu_dat flush), 483 (cpu_writer flush), and 484 (mem_writer flush),
and in every case the returned error is ignored.

`Flush()` returns an error that must be checked: if the underlying file write fails
(disk full, I/O error, filesystem unmounted), the buffered data is silently
discarded and `Flush()` returns non-nil. As a result `runPlotFormat` returns
`(&meta, nil)` even when the output data files are incomplete or empty, and the
caller has no way to detect the data loss.

Why it is a bug: write failures during plot-data generation are silently swallowed,
producing incomplete output reported as success.

Suggested direction: check each `Flush()` return value and propagate any error
through `runPlotFormat`/`RunDirect`'s `(*PlotMeta, error)` contract (related to the
panic-to-error ticket).

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: Five bufio.Writer.Flush() calls discard their error, silently losing data while RunDirect returns nil. Fix: check each Flush error and return it with context (combine with #0049's panic→error refactor).

## Resolution

Fixed in `core/cmd/perfmonger-core/plotformatter/plotformatter.go`.

- Introduced a package-level `flushWriter` seam (mirroring the existing
  `closeTmpFile` seam) wrapping `*bufio.Writer.Flush`, so a flush failure can
  be injected in tests.
- Routed all five buffered-writer flushes (per-device disk_dat, the combined
  disk.dat `df_writer`, per-core cpu_dat, the combined cpu.dat `cpu_writer`,
  and `mem_writer`) through `flushWriter` and now check each return value,
  propagating any error via the existing `(*PlotMeta, error)` return path
  wrapped with context (device name / core id / output file path).
- Scoped to flush-error handling only; the broader panic→error refactor
  (#0049 / #0050) was intentionally left untouched.

Test added: `TestRunPlotFormatPropagatesFlushError` in
`plotformatter_test.go` injects a failing `flushWriter` and asserts
`runPlotFormat` returns a non-nil error that wraps the underlying flush error.

Verified: `go test -count=1 ./cmd/perfmonger-core/plotformatter/` passes and
the unified binary builds successfully.
