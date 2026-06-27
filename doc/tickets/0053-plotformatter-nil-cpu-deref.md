---
title: plotformatter nil pointer dereference if the first record has a nil Cpu field
type: bug
priority: medium
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger-core/plotformatter/plotformatter.go` around lines 319-320,
`records[0].Cpu.NumCore` is accessed without a nil check. `StatRecord.Cpu` is
declared as `*CpuStat` (a pointer; see `stat.go` line 141). If the gob-encoded
first record was written without a `Cpu` payload — e.g. a malformed or
partially-written log file — `Cpu` is nil after decoding, and the access at line
319 causes a nil-pointer-dereference panic.

Since `runPlotFormat` is reachable via `RunDirect` (a library API), any caller
feeding an unusual log file receives an unhandled panic rather than an error.

Why it is a bug: malformed input crashes the formatter (and its library callers)
instead of yielding a clean error.

Suggested direction: nil-check `records[0].Cpu` (and other pointer fields it
relies on) before dereferencing, and return an error describing the malformed
record rather than panicking.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: records[0].Cpu.NumCore is dereferenced without a nil check; Cpu is a *CpuStat, so a malformed log with nil Cpu panics. Fix: check records[0].Cpu != nil and return an error describing the malformed record.
