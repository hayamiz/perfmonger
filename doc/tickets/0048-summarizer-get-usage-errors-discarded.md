---
title: Summarizer discards all errors from Get*Usage helpers
type: bug
priority: medium
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger-core/summarizer/summarizer.go` around lines 153-176, the
code calls `GetCpuUsage`, `GetInterruptUsage`, `GetDiskUsage1`, and `GetNetUsage`,
assigning each one's error return into the same `err` variable without checking it
between calls. Line 176 (`_ = err`) then explicitly discards the last value, with a
comment about "preserving existing behavior".

As a result, a failure in any of the first three calls is immediately overwritten
by the next call's result and never surfaced. Callers receive partial output with
no indication of what went wrong.

Why it is a bug: error returns that exist specifically to report computation
failures are systematically dropped, hiding partial/invalid summaries.

Suggested direction: check each helper's error immediately after its call and
either return it (failing the summary) or accumulate and report it, rather than
chaining all four into one discarded `err`. Coordinate with the related interrupt
panics/Inf tickets, which are the failures most likely to be masked here.

## Triage

- Complexity: medium
- Mechanical fix: no
- Requires user decision: yes
- Notes: Errors from the four Get*Usage helpers are discarded. The fix needs a strategy decision: fail-hard vs accumulate-and-report vs warn-to-stderr, which affects partial output and the pager integration.

## Implementation Notes

Four calls (GetCpuUsage, GetInterruptUsage, GetDiskUsage1, GetNetUsage) assign to the same `err` without checking between calls, and the accumulated err is explicitly discarded ("preserve existing behavior"). Callers get partial output with no indication of failure.

Options:
- Option 1 (fail-hard, recommended): return the error as soon as any helper fails; the pager caller surfaces it. Safest/simplest; a single helper failure aborts the whole summary.
- Option 2 (accumulate-and-report): collect non-nil errors, set the corresponding usage pointers to nil, and either log to stderr or encode under an "errors" JSON key; return nil so output still renders. Changes output structure.
- Option 3 (warn-to-stderr per function): log each error and continue; warnings may be lost in the pager context.

Coordination: related to interrupt panic/Inf tickets (#0045/#0046/#0047) which may inform the strategy. Decision point: fail-hard vs partial output.
