---
title: Truncated gob/gzip stream causes player to panic instead of exiting gracefully
type: bug
priority: medium
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger-core/player/player.go` around lines 235-253 (header decode)
and lines 267-268 (main decode loop), decoding via `gob.Decode` of a `.pgr` /
`.pgr.gz` file that was truncated mid-record returns `io.ErrUnexpectedEOF`, not
`io.EOF`. The code checks only `if err == io.EOF` for clean termination and falls
through to `panic(err)` for any other error, including `io.ErrUnexpectedEOF`.

Truncated-but-partially-valid logs are a common real-world condition (the recorder
was killed mid-write, or the disk filled). Such a file crashes the player with an
unhandled panic instead of emitting the valid records that were fully written
before the truncation and then exiting cleanly.

Why it is a bug: ordinary partially-written input crashes the tool rather than
degrading gracefully.

Suggested direction: treat `io.ErrUnexpectedEOF` (and ideally any decode error at a
record boundary) as end-of-stream — emit the records decoded so far, optionally
warn on stderr, and return a clean (possibly non-zero) status rather than
panicking.

## Triage

- Complexity: high
- Mechanical fix: no
- Requires user decision: yes
- Notes: Truncated gob/gzip stream yields io.ErrUnexpectedEOF; the loop checks only io.EOF and panics on anything else. Fix needs a decision on whether truncation is a clean end-of-stream or a reported/partial-output condition.

## Implementation Notes

gob.Decode returns io.ErrUnexpectedEOF (not io.EOF) when a record is truncated mid-decode (recorder killed mid-write, disk full, dropped connection). The loop panics on any non-EOF error.

Options:
- Option A (simplest): treat io.ErrUnexpectedEOF like io.EOF and break, emitting all valid records. Two one-line changes; silently ignores truncation.
- Option B: detect record-boundary vs mid-record truncation (gob doesn't expose boundaries; complex, little behavioral gain).
- Option C: distinguish a "truncated" exit code (coordinate with #0020) so scripts can detect partial output.

Recommended: Option A for safety, combined with #0020 for optional truncation reporting. Add a test with a truncated .pgr verifying clean exit and decoded records.
