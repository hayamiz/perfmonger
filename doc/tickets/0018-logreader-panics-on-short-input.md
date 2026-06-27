---
title: NewPerfmongerLogReader panics on empty or short input instead of returning an error
type: bug
priority: high
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/internal/perfmonger/utils.go` around lines 13-15, `NewPerfmongerLogReader`
calls `reader.Peek(2)` to sniff a gzip magic header. `bufio.Reader.Peek(2)` returns
an error (`io.EOF` or `io.ErrUnexpectedEOF`) when the underlying stream has fewer
than 2 bytes — for example an empty stdin, a zero-byte file, or a truncated log
file. The code responds with `panic(e)`.

Because `NewPerfmongerLogReader` returns only an `io.Reader` (not an `error`), there
is no clean way for callers to handle this condition. The panic propagates
uncaught all the way up — bypassing the `defer f.Close()` and the normal error
return paths in the `play` and `summary` code — crashing the process with a stack
trace instead of a user-friendly message, and exiting in a way cobra does not
capture as a proper command error.

Why it is a bug: running `perfmonger play`/`summary` on an empty or 1-byte file is
ordinary user input that should produce a clean error, not a process crash.

Suggested direction: change `NewPerfmongerLogReader` to return `(io.Reader, error)`
and propagate the `Peek` error (treating too-short input as an empty/invalid log),
or detect the short-read case and return a sentinel error that callers map to a
clean exit. Update all call sites accordingly.

## Triage

- Complexity: medium
- Mechanical fix: no
- Requires user decision: yes
- Notes: Fixing requires changing NewPerfmongerLogReader to return (io.Reader, error) (or io.ReadCloser, error), rippling across three call sites (player, summarizer, plotformatter) and the cobra commands. Coordinate with #0019/#0020.

## Implementation Notes

NewPerfmongerLogReader calls bufio.Reader.Peek(2), which errors on empty/short input, and currently panics rather than propagating — crashing the tool and bypassing file Close defers.

Approach: change signature from `func(io.Reader) io.Reader` to return an error; let callers distinguish "empty input" (not necessarily fatal) from "I/O error". Update call sites: player.go, summarizer.go, plotformatter.go, and the cobra entry points (needed for #0020).

Alternatives: sentinel-error wrapped reader (less idiomatic), or silently return an empty reader (masks corruption). Trade-off: breaking change to the library API but improves safety. Should be coordinated with #0019 (which also changes the return type to io.ReadCloser).
