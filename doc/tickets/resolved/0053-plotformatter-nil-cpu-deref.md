---
title: plotformatter nil pointer dereference if the first record has a nil Cpu field
type: bug
priority: medium
status: resolved
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

## Resolution

Fixed via strict TDD in `core/cmd/perfmonger-core/plotformatter/`.

- Added a nil check for the first record's `Cpu` field in `runPlotFormat`
  (`plotformatter.go`), immediately after the first record is decoded and
  before `records[0].Cpu.NumCore` is dereferenced. When `Cpu` is nil it now
  returns `fmt.Errorf("malformed log file: first record has no CPU data")`
  instead of panicking.
- Added `TestRunPlotFormatNilCpuReturnsError` in `plotformatter_test.go`,
  which writes a gob log with a valid header pair and a first record whose
  `Cpu` field is nil, then asserts that `runPlotFormat` returns a non-nil
  error (no panic).

RED (before fix): test panicked with
`runtime error: invalid memory address or nil pointer dereference` at
`plotformatter.go:333` (the `records[0].Cpu.NumCore` access).

GREEN (after fix): `go test -count=1 ./cmd/perfmonger-core/plotformatter/`
passes; `go build` of the perfmonger binary succeeds.

Scope limited to this ticket; the broader panic->error refactor (#0049) was
not undertaken.
