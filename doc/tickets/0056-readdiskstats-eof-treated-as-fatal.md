---
title: ReadDiskStats treats io.EOF from a partial Sscanf as fatal, breaking short-format lines
type: bug
priority: high
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/internal/perfmonger/perfmonger_linux.go` around lines 334-342,
`fmt.Sscanf` is used to parse a `/proc/diskstats` line. When `Sscanf` reads fewer
fields than requested (because the line ends), it returns `(n, io.EOF)`. For a
7-field diskstats line (old pre-2.6 kernels, or synthetic device entries), the
`Sscanf` call requesting 14 fields returns `(7, io.EOF)`.

The unconditional check at line 340 — `if err != nil { return err }` — returns
`io.EOF` to the caller before the `num_items == 7` branch at line 348 is ever
reached. The entire `ReadDiskStats` call then fails with a spurious EOF error
instead of gracefully handling the old/short format. `ReadCpuStat` avoids this by
checking `err == io.EOF` before the generic `err != nil`; `ReadDiskStats` omits
that.

Why it is a bug: valid short-format diskstats lines cause the whole disk-stat read
to fail.

Suggested direction: mirror the `ReadCpuStat` pattern — check for `err == io.EOF`
(and use the returned field count `num_items`) before the generic `err != nil`
return, so the 7-field branch is reached and short lines are handled.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: ReadDiskStats returns on any Sscanf error including io.EOF, breaking short-format lines; ReadCpuStat already handles io.EOF as a valid partial-read outcome. Fix: check `if err == io.EOF` before the generic error return, then branch on num_items. Mechanical, mirrors existing pattern.
