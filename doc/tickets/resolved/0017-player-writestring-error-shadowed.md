---
title: player WriteString error silently discarded due to inner-scope err shadowing
type: bug
priority: high
status: resolved
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger-core/player/player.go` around lines 278-284, the output
write path shadows the outer `err`:

```go
if str, err := printer.String(); err != nil {
    ...
} else {
    _, err = out.WriteString(str + "\n")   // assigns the INNER err
}
err = out.Flush()                          // overwrites the OUTER err
```

The `if str, err := ...` short-variable declaration introduces a new `err` scoped
to the `if`/`else` block. The assignment `_, err = out.WriteString(...)` inside the
`else` branch therefore writes into that inner `err`, which is discarded when the
block exits. The outer `err` is never updated by the write. The next statement,
`err = out.Flush()`, overwrites the outer `err` with the flush result.

As a result, a failure from `out.WriteString` (e.g. a broken downstream pipe or a
full disk) is completely invisible: the loop neither breaks nor reports an error,
and records are silently lost while the player reports success.

Why it is a bug: write errors during playback go undetected, producing a silent,
partial output with no diagnostic.

Suggested direction: assign the write result to the outer `err` (declare `str`
outside the `if`, or use a distinct variable name for the inner error and check it
explicitly), then break/propagate on a non-nil write error before the `Flush`.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: Variable shadowing bug in the output write path. Use a distinct variable (e.g. writeErr) for the WriteString error and check/break before Flush. No design choices.

## Resolution

Extracted the per-record output logic into a new `writeRecord(out *bufio.Writer,
str string) error` helper in `core/cmd/perfmonger-core/player/player.go`. The
helper captures the `WriteString` result in a distinct variable (`writeErr`),
returns immediately on a non-nil write error, and only calls `Flush` on success,
eliminating the inner-scope `err` shadowing. In `RunDirect`, the `printer.String()`
error variable was renamed to `perr` (so it no longer shadows the outer `err`),
and the loop now breaks when `writeRecord` returns a non-nil error, so write
failures (broken pipe / full disk) are surfaced instead of being silently
swallowed.

Test added (TDD): `core/cmd/perfmonger-core/player/player_test.go` —
`TestWriteRecordSurfacesWriteError` wraps a failing `io.Writer` in a small-buffer
`bufio.Writer` and asserts `writeRecord` returns a non-nil error. The test
initially failed to compile (`undefined: writeRecord`) before the helper existed,
then passed after the fix.

Verification: `go test -cover ./...` in `core/internal/perfmonger` passes; the
`player` package tests pass; `go vet` is clean; the binary rebuilds successfully.
(Pre-existing failures in fingerprint/plot/summarizer tests belong to other open
tickets and are unrelated to this change.)
