---
title: fingerprint runCommand always returns nil error, silently discarding command failures
type: bug
priority: medium
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger/fingerprint.go` around lines 136-143, `runCommand` returns
`(output, nil)` in both branches of its `if`/`else`. The comment says "don't treat
as fatal", but the error is not even appended to `f.errors`.

As a result, when a critical command such as `fdisk` or `lsblk` fails (permission
denied, command not found, etc.), the caller receives `output == nil` and
`err == nil`. The callers guard on `len(output) > 0`, so they silently skip saving
the corresponding file with no indication of failure, and `doWithMessage` reports
"done" even though no data was collected. The resulting fingerprint is silently
incomplete with no diagnostic output.

Why it is a bug: collection failures are invisible, producing an incomplete
fingerprint that appears successful.

Suggested direction: append the command error to `f.errors` (and/or return it) so
that failed collection steps are reported by `doWithMessage` and reflected in the
overall pass/fail indication, while still allowing the overall fingerprint run to
continue past non-critical failures.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: runCommand returns nil error in both branches, dropping command failures (fdisk/lsblk). Fix: append the error to f.errors and/or return it, preserving the non-fatal semantics. Straightforward.
