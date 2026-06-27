---
title: GetInterruptUsage panics when i2.Entries is shorter than i1.Entries
type: bug
priority: high
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/internal/perfmonger/usage.go` around lines 235-236, the loop iterates over
every element of `i1.Entries` and reads `i2.Entries[idx]` (line 236) without
checking that `idx < len(i2.Entries)`.

If the number of IRQ entries changed between the two samples — e.g. a device was
added or removed between recording start and end — `i2.Entries` may be shorter than
`i1.Entries`, and the access panics with "index out of range". The existing error
check only validates `IrqNo`/`IrqType` equality for entries that exist; it does not
verify that `i2.Entries` is at least as long as `i1.Entries`.

Why it is a bug: a legitimate change in the IRQ table during a recording crashes
the usage computation.

Suggested direction: before the loop, verify the two slices have matching length
(and return an error on mismatch), or guard the per-iteration access with
`idx < len(i2.Entries)` and skip/handle entries that have no counterpart.

## Triage

- Complexity: medium
- Mechanical fix: yes
- Requires user decision: no
- Notes: The loop over i1.Entries indexes i2.Entries[idx] without a bounds check, panicking when i2 is shorter. Fix: add `if idx >= len(i2.Entries) { return nil, error }` alongside the existing IrqNo/IrqType validation. Clear-cut.
