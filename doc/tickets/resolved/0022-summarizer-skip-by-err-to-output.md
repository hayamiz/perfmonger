---
title: Summarizer writes literal "skip by err" to output and returns nil on JSON failure
type: bug
priority: medium
status: resolved
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger-core/summarizer/summarizer.go` around lines 208-210, when
`printer.String()` fails (the error, `perr`, is non-nil), the code writes the raw
debug string `"skip by err"` to the `out` writer and ignores `perr` entirely.
`RunDirect` then returns `nil`.

`out` may be the pager's stdin pipe when invoked from
`core/cmd/perfmonger/summary.go`. That caller receives `runErr == nil`, treats it
as success, calls `pagerCmd.Wait()`, and exits with code 0. The user therefore
sees the literal string `"skip by err"` in place of valid JSON output — corrupting
any downstream consumer (e.g. `jq`) — while `perfmonger` reports success for a
failed serialization.

Why it is a bug: a serialization failure emits invalid output and is reported as
success, with the error neither logged nor propagated.

Suggested direction: on `printer.String()` failure, do not write a debug string to
`out`; instead return the error from `RunDirect` so the caller surfaces it and
exits non-zero. Optionally log it to stderr.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: When printer.String() fails, the code writes the literal "skip by err" to output and returns nil. Fix: remove the debug line, return the error (optionally log to stderr). Clear error-handling fix, no trade-offs.

## Resolution

Fixed in `core/cmd/perfmonger-core/summarizer/summarizer.go`. The JSON
serialization block previously did:

```go
if str, perr := printer.String(); perr != nil {
    fmt.Fprintln(out, "skip by err")
} else {
    fmt.Fprintln(out, str)
}
```

This was extracted into a small testable helper `writeJSON(printer, out)` which:

- Removes the debug literal `"skip by err"`.
- On `printer.String()` failure, logs the actual error to stderr and returns it
  (no partial/debug output is written to `out`).
- On success, writes the JSON as before.

`RunDirect` now returns the error from `writeJSON`, so a serialization failure
propagates to the caller (e.g. `core/cmd/perfmonger/summary.go`), which exits
non-zero instead of corrupting output and reporting success.

Test added to `core/cmd/perfmonger-core/summarizer/summarizer_test.go`:
`TestWriteJSONPropagatesPrinterError` builds a printer left in an unfinished
state (so `printer.String()` errors), then asserts `writeJSON` returns a
non-nil error, writes nothing to the output buffer, and never emits the
`"skip by err"` literal.

Verification:
- `cd core && go test -count=1 ./cmd/perfmonger-core/summarizer/` — ok
- `cd core/cmd/perfmonger && go build -o ../../../lib/exec/perfmonger_linux_amd64 .` — ok
