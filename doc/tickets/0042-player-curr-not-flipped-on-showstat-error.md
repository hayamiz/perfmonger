---
title: player does not flip curr on showStat error, using wrong prev_rec for all later records
type: bug
priority: high
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger-core/player/player.go` around lines 271-292, the decode
loop uses a two-element ping-pong buffer indexed by `curr`, with
`cur_rec = &records[curr]` and `prev_rec = &records[curr^1]`, flipping `curr ^= 1`
at the end of each iteration (line 292).

When `showStat` returns an error (lines 271-276), the code calls `printer.Reset()`
and `continue`, which **skips** the `curr ^= 1` flip. On the next decode iteration,
`cur_rec` points to the same slot that just failed (overwriting it), and `prev_rec`
points to the record from two steps back instead of the immediately preceding one.

Every subsequent delta calculation (CPU%, disk IOPS, net throughput) is then
computed against stale data from two intervals ago, silently producing wrong output
for all records after any `showStat` error.

Why it is a bug: a single transient `showStat` error corrupts every subsequent
record's deltas without any indication.

Suggested direction: flip `curr` before the `continue` (so the buffer stays
consistent), or restructure the loop so the index advances regardless of whether
`showStat` succeeded. Add a test that injects a `showStat` error mid-stream and
verifies later deltas reference the correct previous record.

## Triage

- Complexity: high
- Mechanical fix: no
- Requires user decision: yes
- Notes: On showStat error the code `continue`s without flipping the ping-pong index, so all later records compute deltas against stale prev_rec. Fix needs a decision on whether failed records are skipped or output, and careful testing.

## Implementation Notes

The decode loop flips `curr ^= 1` only after successful output. When showStat fails it calls printer.Reset() and continue, skipping the flip, so prev_rec points two intervals back and all subsequent deltas are corrupted silently.

Options:
- Option A (minimal): add `curr ^= 1` before the continue. Single line; keeps buffer consistent. Side effect: records that fail showStat are skipped from output.
- Option B: restructure the loop so the flip always happens after processing a record regardless of output success. Clearer semantics, more restructuring.
- Option C: replace ping-pong with an explicit struct/circular buffer. Larger refactor.

Couples with #0020: if showStat fails, should the tool report failure or silently skip? Needs a test that injects a showStat error mid-stream and verifies subsequent deltas.
