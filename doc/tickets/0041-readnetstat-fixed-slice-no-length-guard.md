---
title: ReadNetStat slices /proc/net/dev lines at [0:7] without a length guard
type: bug
priority: low
status: open
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
