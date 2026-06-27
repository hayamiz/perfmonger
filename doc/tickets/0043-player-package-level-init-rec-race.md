---
title: Package-level init_rec causes a data race and wrong elapsed_time across player calls
type: bug
priority: medium
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 2/3 skeptics.

In `core/cmd/perfmonger-core/player/player.go`, `init_rec` is declared as a
package-level variable (line 26) and written by each call to `RunDirect`
(line 256). It is read in `showStat` (line 114) to compute `elapsed_time`.

Because `RunDirect` (and `RunWithOption`/`Run`, which delegate to it) is intended
as a reusable library API, this shared mutable state is unsafe:

- If `RunDirect` is called concurrently from multiple goroutines, the writes to
  `init_rec` race and `elapsed_time` is computed against an arbitrary call's first
  record.
- Even sequentially, a second call on a different log file overwrites `init_rec`,
  so any interleaving or reuse produces `elapsed_time` values relative to the
  wrong baseline.

Why it is a bug: a library entry point holds per-invocation state in a package
global, producing incorrect output and a data race when used as designed.

Suggested direction: move `init_rec` into per-invocation local state (e.g. a struct
or a value threaded through the decode loop / passed to `showStat`) so each
`RunDirect` call has its own baseline record.

## Triage

- Complexity: medium
- Mechanical fix: no
- Requires user decision: yes
- Notes: Package-level init_rec is shared across RunDirect invocations, causing a data race and wrong elapsed_time. Fix requires threading per-invocation state; choice of mechanism (parameter vs context struct) is a design decision.

## Implementation Notes

init_rec is a package-level mutable var read by showStat and written by RunDirect, causing a data race on concurrent calls and cross-invocation pollution on sequential calls, violating the reusable-library contract.

Options:
- Option A (recommended): pass init_rec as a parameter to showStat. Simple, explicit, thread-safe; changes showStat signature.
- Option B: introduce a PlayerContext struct holding per-invocation state (InitRec, Printer, regex, ...). Larger refactor; better if more state accumulates.
- Option C: mutex (rejected — doesn't fix cross-invocation pollution or the library-semantics issue).

Could be combined with #0020/#0042 if those restructure the RunDirect loop.
