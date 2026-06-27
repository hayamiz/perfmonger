---
title: GetInterruptUsage produces +Inf interrupt rates when the recording interval is zero
type: bug
priority: medium
status: open
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
