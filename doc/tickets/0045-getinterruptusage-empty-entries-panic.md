---
title: GetInterruptUsage panics with index-out-of-range when Entries is empty
type: bug
priority: high
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/internal/perfmonger/usage.go` around line 219 (also referenced as lines
218-219), `GetInterruptUsage` unconditionally accesses `i1.Entries[0]` to obtain
`num_core`. There is no length guard before the access.

`ReadInterruptStat` can produce an `InterruptStat` with an empty `Entries` slice —
for example when `/proc/interrupts` contains only the header line and no interrupt
entries (possible in minimal container or VM environments), or when a partial log
is replayed, or in unit-test fakes. Calling `GetInterruptUsage` with such input
panics with "index out of range" instead of returning an error.

Why it is a bug: valid-but-empty interrupt data crashes the summarizer/usage path
rather than producing a clean error or empty result.

Suggested direction: guard with `len(i1.Entries) == 0` (and `len(i2.Entries) == 0`)
and return an error or an empty usage result before dereferencing `Entries[0]`.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: GetInterruptUsage accesses i1.Entries[0] without checking for empty (ReadInterruptStat can yield zero entries). Fix: guard `len(i1.Entries)==0 || len(i2.Entries)==0` and return an error. Unambiguous.
