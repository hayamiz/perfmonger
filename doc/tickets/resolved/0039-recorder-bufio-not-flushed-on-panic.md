---
title: bufio.Writer not flushed before gzip.Writer.Close() on panic paths, corrupting output
type: bug
priority: medium
status: resolved
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

## Resolution

Fixed via strict TDD in `core/cmd/perfmonger-core/recorder/recorder.go`.

- Extracted the gzip + bufio wrapping into a new testable helper
  `newGzipBufWriter(file io.Writer) (out *bufio.Writer, cleanup func())`. The
  returned `cleanup` (invoked via `defer cleanup()`) flushes the bufio buffer
  into the gzip writer **before** closing the gzip writer, so buffered bytes
  reach the gzip writer before its footer is written — even on panic paths.
- `cleanup` captures any in-flight panic with `recover()`, performs the flush
  and close, then re-raises the original panic. This guarantees the flush/close
  error never masks the original panic, while a flush/close error on the normal
  (non-panic) path is still surfaced via `panic`.
- `RunDirect`'s gzip branch now uses `out, cleanup = newGzipBufWriter(file)` +
  `defer cleanup()`, replacing the previous `defer gzwriter.Close()` that ran
  without a paired flush on panic. The existing normal-path explicit `out.Flush()`
  makes the deferred flush a harmless no-op on success.

Test added: `TestGzipBufWriterFlushesBufferOnPanic` in
`core/cmd/perfmonger-core/recorder/recorder_test.go`. It writes a payload into
the bufio buffer, panics without flushing, asserts the panic propagates, then
decompresses the gzip output and verifies all buffered bytes survived.

RED (before fix): `undefined: newGzipBufWriter` (initially `withGzipWriter`),
build failed. GREEN (after fix): test passes. Full recorder package `go test`
and `go vet` pass; the unified binary builds.
