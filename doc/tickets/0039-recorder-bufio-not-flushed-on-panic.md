---
title: bufio.Writer not flushed before gzip.Writer.Close() on panic paths, corrupting output
type: bug
priority: medium
status: open
created: 2026-05-29
updated: 2026-06-27
---

## Description

Found by an automated code audit and verified by 3/3 skeptics.

In `core/cmd/perfmonger-core/recorder/recorder.go`, the gzip writer is wrapped by a
`bufio.Writer` (around line 366). In the normal termination path, `out.Flush()`
(around line 482) runs before the deferred `gzwriter.Close()` (around line 364),
so the ordering is correct.

However, if a panic occurs anywhere between the defer registration (lines 362-366)
and the explicit flush (line 482) — for example `enc.Encode(cheader)` panicking at
line 378, or any panic inside the sampling loop (see the unvalidated CPU index and
`/proc/net/dev` slicing issues) — Go's panic unwinding runs the deferred
`gzwriter.Close()` **without** ever calling `out.Flush()`. The data buffered in the
`bufio.Writer` is silently discarded and the output gzip file is corrupted or
incomplete.

Why it is a bug: any panic during recording produces a truncated/corrupt output
file rather than at least flushing what was buffered.

Suggested direction: register a deferred `out.Flush()` ordered to run before the
deferred `gzwriter.Close()` (defers run LIFO, so register the gzip close first then
the flush), or wrap the recording body in a recover that flushes-then-closes before
re-raising. Reducing in-loop panics (see related tickets) is complementary.

## Triage

- Complexity: medium
- Mechanical fix: yes
- Requires user decision: no
- Notes: Register `defer out.Flush()` ordered (LIFO) to run before the deferred gzwriter.Close(), so buffered data is flushed into the gzip writer before its footer is written, including on panic paths.
