---
title: gzip.Reader opened in NewPerfmongerLogReader is never closed, skipping checksum validation
type: bug
priority: medium
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/internal/perfmonger/utils.go` around lines 21-22, when the input is
gzip-compressed, `NewPerfmongerLogReader` creates a `*gzip.Reader` (which
implements `io.Closer`) and returns it as a plain `io.Reader`. Neither
`NewPerfmongerLogReader` nor any caller — `RunDirect` in
`core/cmd/perfmonger-core/player/player.go` (around line 222), nor the summarizer
path — ever calls `Close()` on this reader.

`gzip.Reader.Close()` is required to read and validate the gzip trailer (CRC32
checksum plus uncompressed size). Without it, a silently truncated or corrupted
gzip stream whose data happens to decode without an intermediate error is
processed as if it were valid, producing wrong data with no error. The internal
`bufio.Reader` state held by the gzip reader is also never released.

Why it is a bug: corruption that gzip is specifically designed to detect goes
undetected, and a closer is leaked.

Suggested direction: return the underlying `io.Closer` (or an
`io.ReadCloser`) from `NewPerfmongerLogReader` so callers can `defer Close()` it,
and ensure the player/summarizer fully consume then close the gzip reader so the
trailer checksum is verified.

## Triage

- Complexity: medium
- Mechanical fix: no
- Requires user decision: yes
- Notes: gzip.Reader is never closed, skipping CRC validation and leaking state. Fix requires changing NewPerfmongerLogReader to return io.ReadCloser (breaking change) and deferring Close() at all three call sites. Coordinate with #0018.

## Implementation Notes

gzip.NewReader's closer is never called, skipping gzip trailer (CRC32) validation and leaking the internal bufio state.

Approach: change NewPerfmongerLogReader to return io.ReadCloser (wrap the plain bufio.Reader in a no-op closer when uncompressed; return the gzip.Reader directly when compressed). Add `defer input_reader.Close()` at all call sites (player.go, summarizer.go, plotformatter.go).

COORDINATE WITH #0018: both tickets change the NewPerfmongerLogReader return type and should be fixed together. Breaking API change but necessary for correctness.
