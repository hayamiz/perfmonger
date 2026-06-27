---
title: GetInterruptUsage panics with index-out-of-range when Entries is empty
type: bug
priority: high
status: resolved
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

## Resolution

Resolved via strict TDD.

- Source: `core/internal/perfmonger/usage.go` — `GetInterruptUsage` now guards
  `len(i1.Entries) == 0 || len(i2.Entries) == 0` and returns
  `errors.New("No interrupt stat entries")` before dereferencing `Entries[0]`,
  eliminating the index-out-of-range panic. (`errors` was already imported.)
- Test: added `TestGetInterruptUsageEmptyEntries` to
  `core/internal/perfmonger/usage_test.go`. It builds two empty `InterruptStat`
  values via `NewInterruptStat()` and asserts a non-nil error. Before the fix it
  panicked with `index out of range [0] with length 0` at usage.go:219; after the
  fix it passes.

Verification (all passing):
- `cd core/internal/perfmonger && go test -count=1 ./...` — ok
- `go vet perfmonger_linux.go $(ls *.go | grep -v perfmonger_)` — clean
- `cd core/cmd/perfmonger && go build -o ../../../lib/exec/perfmonger_linux_amd64 .` — ok
