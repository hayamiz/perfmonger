---
title: Summarizer uses a zero-initialized StatRecord as the last record for one-record logs
type: bug
priority: medium
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger-core/summarizer/summarizer.go` around lines 132-146, the
decode loop reads records into a two-element ping-pong buffer `lst_records` using
an alternating index `idx`. When the log file contains exactly one data record
beyond the headers, the loop decodes into `lst_records[0]`, and the next decode
immediately returns `io.EOF`. The EOF branch executes `idx ^= 1` (flipping 0 to
1) and then breaks. `lst_record` is subsequently assigned `lst_records[1]`, which
is a **zero-initialized** `StatRecord`: its `Time` is the zero value and its
`Cpu`/`Disk`/`Net` pointers are nil.

The result is:

- `interval = lst_record.Time.Sub(fst_record.Time)` is a huge negative duration
  (about -1970 years), and
- the nil `Cpu`/`Disk`/`Net` fields cause all usage calculations to be skipped,

producing nonsense output. The correct last record is the one in `lst_records[0]`.

Why it is a bug: a perfectly valid single-record log yields garbage summary output
instead of a sensible (or clearly-empty) result.

Suggested direction: do not flip `idx` on the EOF branch, so `lst_record` resolves
to the slot that actually received the last decoded record; or, when the loop body
never completed a second full iteration, fall back to using `fst_record` as
`lst_record`. Add a guard for the zero/negative interval case.

## Triage

- Complexity: low
- Mechanical fix: yes
- Requires user decision: no
- Notes: For a one-record log the ping-pong index flip on EOF makes lst_record reference an uninitialized slot. Fix: don't flip the index on EOF (or detect the single-record case and use fst_record as lst_record). One file, no design decision.
