---
title: ReadNetStat slices /proc/net/dev lines at [0:7] without a length guard
type: bug
priority: low
status: resolved
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 2/3 skeptics.

In `core/internal/perfmonger/perfmonger_linux.go`, `ReadNetStat` evaluates
`line[0:7]` in the switch cases around lines 394 and 396 without first checking
`len(line) >= 7`.

If `/proc/net/dev` ever contains an empty line or a line shorter than 7 bytes —
not guaranteed against by any kernel ABI contract, and plausible in namespace or
container environments with unusual network stacks — the slice operation panics
with "index out of range", crashing the recording process. As with other in-loop
panics, this bypasses `out.Flush()`/`gzwriter.Close()`, leaving a corrupted output
file.

Why it is a bug: a malformed/short line from kernel-provided input crashes the
recorder.

Suggested direction: guard the slice with a `len(line) >= 7` check (skip or handle
short lines) before evaluating `line[0:7]`, or restructure the parsing to tokenize
fields rather than slicing fixed byte ranges.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: ReadNetStat compares line[0:7] without checking len(line) >= 7, risking a slice panic on short input. Fix: add a `if len(line) < 7 { continue }` guard, mirroring ReadCpuStat. Unambiguous.

## Resolution

Fixed via strict TDD in `core/internal/perfmonger/perfmonger_linux.go`.

- Extracted the `/proc/net/dev` line-parsing loop out of `ReadNetStat` into a
  new testable helper `parseNetStat(record *StatRecord, r io.Reader) error`.
  `ReadNetStat` now opens the file and delegates to it. This created a seam so
  the parser can be exercised with an in-memory reader.
- Added a `if len(line) < 7 { continue }` guard before the `line[0:7]`
  comparisons, so empty or short lines are skipped instead of panicking with
  "slice bounds out of range".

Test added: `TestParseNetStatShortLine` in `perfmonger_linux_test.go`, which
feeds the parser input containing an empty line and a 3-byte line and asserts
no panic plus that the valid `lo` entry is still parsed. Before the guard this
test panicked at `perfmonger_linux.go:399` with
`slice bounds out of range [:7] with length 0`; after the guard it passes.

Verification: `go test -count=1 -cover ./...` in `core/internal/perfmonger`
passes (54.4% coverage); `go vet` passes; the `perfmonger` binary rebuilds
cleanly. (Unrelated pre-existing failures remain in the `core/cmd` tree —
tickets #0024 and #0035 — and are outside this ticket's scope.)
