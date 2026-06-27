---
title: fingerprint runCommand always returns nil error, silently discarding command failures
type: bug
priority: medium
status: resolved
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

## Resolution

Changed `runCommand` in `core/cmd/perfmonger/fingerprint.go` so that when the
underlying command fails, the error is wrapped (`command %q failed: %v`),
appended to `f.errors`, and returned to the caller. Previously both branches
returned `(output, nil)`, silently discarding failures.

The non-fatal "continue collecting" semantics are preserved: callers use
`if output, _ := f.runCommand(...); len(output) > 0` and still proceed past a
failure, but the recorded error now makes `doWithMessage` report "failed" for
that step (instead of a misleading "done") and surfaces the failure in the
overall run.

Regression test added in `core/cmd/perfmonger/fingerprint_test.go`:
- `TestRunCommandRecordsErrorOnFailure` runs a non-existent command and asserts
  a non-nil error is returned and recorded in `f.errors` (this failed before the
  fix).
- `TestRunCommandSucceeds` asserts a successful command returns no error and
  leaves `f.errors` empty.

Verification (Go env sourced): `go test -cover ./...` in
`core/internal/perfmonger` passes (54.4% coverage); the `cmd/perfmonger`
package tests pass; `go vet` is clean (exit 0); binary rebuilt successfully.
