---
title: GetInterruptUsage produces +Inf interrupt rates when the recording interval is zero
type: bug
priority: medium
status: resolved
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/internal/perfmonger/usage.go` around lines 251-252, `GetInterruptUsage`
divides by `usage.Interval.Seconds()` without first checking that the interval is
positive. Unlike `GetDiskUsage1`, which explicitly returns an error when
`itv <= 0`, `GetInterruptUsage` performs the division unconditionally.

When two records carry identical timestamps — e.g. a clock hiccup, or the
summarizer being called with `first == last` record — the interval is zero and both
`core_usage.Device` and `core_usage.System` become `+Inf`, silently corrupting the
JSON output.

Why it is a bug: a zero/degenerate interval yields non-finite values in the output
instead of a clean error, mirroring a guard that `GetDiskUsage1` already has.

Suggested direction: add the same `itv <= 0` guard used in `GetDiskUsage1` and
return an error (or skip the computation) when the interval is not strictly
positive.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: GetInterruptUsage divides by Interval.Seconds() without the `itv <= 0` guard that GetDiskUsage1 already has, producing +Inf. Fix: add the same guard and return an error. Mechanical, mirrors existing pattern.

## Resolution

Added an interval guard to `GetInterruptUsage` in
`core/internal/perfmonger/usage.go`. After the empty-entries check, the
interval `t2.Sub(t1)` is computed once and, when its seconds value is
`<= 0`, the function returns `errors.New("negative interval")` — mirroring
the existing guard in `GetDiskUsage1`. This prevents the unconditional
division by `Interval.Seconds()` from producing `+Inf` interrupt rates on a
zero/degenerate interval (e.g. `first == last` record). The computed
interval is reused for `usage.Interval`.

Test added (TDD): `TestGetInterruptUsageZeroInterval` in
`core/internal/perfmonger/usage_test.go` builds valid single-core IRQ
entries with identical timestamps (`t1 == t2`) and asserts the call returns
a non-nil error and a nil usage. It fails before the fix (returns usage with
nil error) and passes after.

Verified:
- `cd core/internal/perfmonger && go test -count=1 ./...` — ok
- `cd core/internal/perfmonger && go vet perfmonger_linux.go $(ls *.go | grep -v perfmonger_)` — clean
- `cd core/cmd/perfmonger && go build -o ../../../lib/exec/perfmonger_linux_amd64 .` — ok
